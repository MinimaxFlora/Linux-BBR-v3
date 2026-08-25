// Package netutil 封装网络操作：GitHub release 查询/下载、Ookla speedtest。
package netutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
func FetchReleases(ctx context.Context, token string) ([]Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", bbr.RepoFullName())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
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

// FetchTagCommit 获取 tag 指向的 commit SHA。
// 固定 tag release 的 target_commitish 可能是分支名（如 "master"），
// 需要走 git ref API 拿真实 SHA；annotated tag 需再解引用一次。
func FetchTagCommit(ctx context.Context, token, tag string) (string, error) {
	sha, typ, err := gitRefObject(ctx, token, "tags/"+tag)
	if err != nil {
		return "", err
	}
	if typ == "tag" {
		// annotated tag：tag 对象再指向 commit
		sha, typ, err = gitTagObject(ctx, token, sha)
		if err != nil {
			return "", err
		}
	}
	if typ != "commit" {
		return "", fmt.Errorf("tag %s 指向 %s（非 commit），无法比对", tag, typ)
	}
	return sha, nil
}

// gitRefObject 查询 git ref（refs/tags/<tag>）指向的对象。
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

// gitTagObject 查询 annotated tag 对象指向的 commit。
func gitTagObject(ctx context.Context, token, tagSha string) (sha, typ string, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/tags/%s", bbr.RepoFullName(), tagSha)
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
func Download(ctx context.Context, url, dest string, progress func(downloaded, total int64)) error {
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
