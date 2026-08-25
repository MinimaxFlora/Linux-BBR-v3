// Package netutil 封装网络操作：GitHub release 下载（走镜像）、版本信息 ini、Ookla speedtest。
// 版本检测完全绕开 GitHub API：CLI 构建 commit 与最新内核版本由 version.ini 提供。
package netutil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
)

// Asset 是 GitHub release 资产（按 CI 产物规则构造的下载项）。
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
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
