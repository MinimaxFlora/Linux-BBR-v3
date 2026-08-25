// Package netutil 封装网络操作：GitHub release 查询/下载、Ookla speedtest。
package netutil

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
)

// Asset 是 GitHub release 资产。
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Release 是 GitHub release（仅含需要的字段）。
type Release struct {
	TagName         string  `json:"tag_name"`
	TargetCommitish string  `json:"target_commitish"`
	PublishedAt     string  `json:"published_at"`
	Assets          []Asset `json:"assets"`
}

// GitHubAPIError 表示 GitHub API 返回的业务错误。
type GitHubAPIError struct {
	Message string
}

func (e *GitHubAPIError) Error() string {
	return e.Message
}

// IsRateLimit 判断是否为 API rate limit 错误。
func (e *GitHubAPIError) IsRateLimit() bool {
	return strings.Contains(strings.ToLower(e.Message), "rate limit exceeded")
}

// FetchReleases 获取仓库全部 releases（对应原 gh_api_get + check_release_api_response）。
// token 为空时匿名请求。返回错误可能是 *GitHubAPIError。
// 直连 API 失败时（如国内网络）自动降级到 releases.atom（github.com 域，可走镜像代理）。
func FetchReleases(ctx context.Context, token string) ([]Release, error) {
	base := fmt.Sprintf("https://api.github.com/repos/%s/releases", bbr.RepoFullName())
	releases, err := fetchReleasesOnce(ctx, token, base)
	if err == nil {
		return releases, nil
	}
	var apiErr *GitHubAPIError
	if errors.As(err, &apiErr) {
		// API 已响应（如 rate limit）：业务错误不降级
		return nil, err
	}
	atomReleases, atomErr := fetchReleasesAtom(ctx)
	if atomErr == nil {
		return atomReleases, nil
	}
	return nil, fmt.Errorf("获取 release 失败（直连: %v；atom 降级: %v）", err, atomErr)
}

func fetchReleasesOnce(ctx context.Context, token, url string) ([]Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	// 镜像请求不携带 token（第三方代理不应收到用户凭据；公开仓库匿名可用）
	if token != "" && !IsMirrorURL(url) {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}

	// 先检查 API 错误对象（{"message": "..."}）
	var apiErr struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		return nil, &GitHubAPIError{Message: apiErr.Message}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回 HTTP %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("GitHub API 返回数据格式异常，无法继续: %w", err)
	}
	return releases, nil
}

// NonDebugAssets 返回该 release 中非 debug 的资产（过滤 *-dbg_* / *-dbgsym_*）。
func (r Release) NonDebugAssets() []Asset {
	var out []Asset
	for _, a := range r.Assets {
		if !bbr.IsDebugAsset(a.BrowserDownloadURL) {
			out = append(out, a)
		}
	}
	return out
}

// FetchBranchHead 获取分支最新 commit SHA（release-cli 由 push master 触发，
// master head 即最新二进制资产对应的源码 commit）。
// 直连 API 失败时降级到 commits.atom（github.com 域，可走镜像代理）。
func FetchBranchHead(ctx context.Context, token, branch string) (string, error) {
	sha, typ, err := gitRefObject(ctx, token, "heads/"+branch)
	if err == nil {
		if typ != "commit" {
			return "", fmt.Errorf("branch %s 指向 %s（非 commit）", branch, typ)
		}
		return sha, nil
	}
	var apiErr *GitHubAPIError
	if errors.As(err, &apiErr) {
		return "", err
	}
	sha, atomErr := fetchBranchHeadAtom(ctx, branch)
	if atomErr == nil {
		return sha, nil
	}
	return "", fmt.Errorf("获取分支 %s 最新 commit 失败（直连: %v；atom 降级: %v）", branch, err, atomErr)
}

// gitRefObject 查询 git ref（refs/heads/<branch> 或 refs/tags/<tag>）指向的对象。
func gitRefObject(ctx context.Context, token, ref string) (sha, typ string, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/ref/%s", bbr.RepoFullName(), ref)
	var resp struct {
		Object struct {
			SHA  string `json:"sha"`
			Type string `json:"type"`
		} `json:"object"`
	}
	if err := ghGetJSON(ctx, token, url, &resp); err != nil {
		return "", "", err
	}
	return resp.Object.SHA, resp.Object.Type, nil
}

// ghGetJSON 通用 GitHub API GET（带 token），解析 JSON 到 out，处理 API 错误对象。
// api.github.com 不支持镜像代理，仅直连；调用方按需做 atom 降级。
func ghGetJSON(ctx context.Context, token, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	var apiErr struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		return &GitHubAPIError{Message: apiErr.Message}
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API 返回 HTTP %d", resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}

// Download 下载 URL 到本地路径，进度回调（downloaded, total 字节；total 未知为 -1）。
// 直连失败时自动依次尝试镜像源（静默切换，全部失败返回汇总错误）。
func Download(ctx context.Context, url, dest string, progress func(downloaded, total int64)) error {
	var errs []string
	for _, u := range candidateURLs(url) {
		err := downloadOnce(ctx, u, dest, progress)
		if err == nil {
			return nil
		}
		_ = os.Remove(dest) // 清理失败源的残留文件
		errs = append(errs, fmt.Sprintf("%s: %v", u, err))
	}
	return fmt.Errorf("所有下载源均失败：%s", strings.Join(errs, "; "))
}

func downloadOnce(ctx context.Context, url, dest string, progress func(downloaded, total int64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "bbrv3/"+bbr.RepoFullName())
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败：HTTP %d (%s)", resp.StatusCode, url)
	}

	total := resp.ContentLength
	if progress != nil {
		progress(0, total)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 256*1024)
	var downloaded int64
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	return nil
}

// ---- API 降级：github.com 域 atom feed（镜像可代理） ----

// atomFeed / atomEntry 解析 GitHub atom feed（releases.atom / commits.atom）。
type atomFeed struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID      string `xml:"id"`
	Title   string `xml:"title"`
	Updated string `xml:"updated"`
}

// fetchReleasesAtom 通过 releases.atom 获取 release 列表（含 tag 名与发布时间，
// 不含资产信息——资产由调用方按 CI 产物规则构造）。github.com 域可走镜像代理。
func fetchReleasesAtom(ctx context.Context) ([]Release, error) {
	feedURL := fmt.Sprintf("https://github.com/%s/releases.atom", bbr.RepoFullName())
	var errs []string
	for _, u := range candidateURLs(feedURL) {
		rels, err := fetchReleasesAtomOnce(ctx, u)
		if err == nil {
			return rels, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", u, err))
	}
	return nil, fmt.Errorf("%s", strings.Join(errs, "; "))
}

func fetchReleasesAtomOnce(ctx context.Context, url string) ([]Release, error) {
	body, err := fetchAtomBody(ctx, url)
	if err != nil {
		return nil, err
	}
	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析 atom 失败: %w", err)
	}
	var rels []Release
	for _, e := range feed.Entries {
		tag := strings.TrimSpace(e.Title)
		if tag == "" {
			continue
		}
		// GitHub 的 releases.atom 用 <updated> 记录时间（格式与 API 的 published_at 一致）
		rels = append(rels, Release{TagName: tag, PublishedAt: strings.TrimSpace(e.Updated)})
	}
	if len(rels) == 0 {
		return nil, errors.New("atom 无 release 条目")
	}
	return rels, nil
}

var commitIDRe = regexp.MustCompile(`Commit/([0-9a-f]{40})`)

// fetchBranchHeadAtom 通过 commits.atom 获取分支最新 commit SHA（第一项）。
func fetchBranchHeadAtom(ctx context.Context, branch string) (string, error) {
	feedURL := fmt.Sprintf("https://github.com/%s/commits/%s.atom", bbr.RepoFullName(), branch)
	var errs []string
	for _, u := range candidateURLs(feedURL) {
		sha, err := fetchBranchHeadAtomOnce(ctx, u)
		if err == nil {
			return sha, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", u, err))
	}
	return "", fmt.Errorf("%s", strings.Join(errs, "; "))
}

func fetchBranchHeadAtomOnce(ctx context.Context, url string) (string, error) {
	body, err := fetchAtomBody(ctx, url)
	if err != nil {
		return "", err
	}
	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return "", fmt.Errorf("解析 atom 失败: %w", err)
	}
	if len(feed.Entries) == 0 {
		return "", errors.New("atom 无 commit 条目")
	}
	m := commitIDRe.FindStringSubmatch(feed.Entries[0].ID)
	if m == nil {
		return "", fmt.Errorf("无法从 atom 提取 commit SHA: %s", feed.Entries[0].ID)
	}
	return m[1], nil
}

// fetchAtomBody GET atom feed 并返回 body（限 4MB）。
func fetchAtomBody(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/atom+xml")
	req.Header.Set("User-Agent", "bbrv3/"+bbr.RepoFullName())
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d (%s)", resp.StatusCode, url)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}
