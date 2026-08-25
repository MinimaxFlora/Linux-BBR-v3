package netutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
)

// VersionINI 版本信息文件（bbrv3-cli release 资产 version.ini，CI 每次构建刷新，
// 用于完全绕开 GitHub API 的版本检测）：
//
//	[cli]
//	commit=14635050        # CLI 二进制构建时的 GITHUB_SHA::8
//	built=2026-08-25T13:06:56Z
//	[kernel]
//	version=7.2.0          # 最新已发布内核版本（tag 按 <arch>-<version>[-max] 构造）
type VersionINI struct {
	CLICommit     string // [cli] commit
	CLIBuilt      string // [cli] built
	KernelVersion string // [kernel] version
}

// ParseVersionINI 解析 version.ini 文本（容忍注释、空行、未知键）。
func ParseVersionINI(content string) (*VersionINI, error) {
	ini := &VersionINI{}
	section := ""
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch section {
		case "cli":
			switch k {
			case "commit":
				ini.CLICommit = v
			case "built":
				ini.CLIBuilt = v
			}
		case "kernel":
			switch k {
			case "version":
				ini.KernelVersion = v
			}
		}
	}
	if ini.CLICommit == "" && ini.KernelVersion == "" {
		return nil, errors.New("version.ini 内容为空或格式异常")
	}
	return ini, nil
}

// FetchVersionINI 下载并解析 bbrv3-cli 资产的 version.ini（走镜像，国内可用）。
func FetchVersionINI(ctx context.Context) (*VersionINI, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/bbrv3-cli/version.ini", bbr.RepoFullName())
	tmp := filepath.Join(os.TempDir(), "bbrv3-version.ini")
	_ = os.Remove(tmp)
	if err := Download(ctx, url, tmp, nil); err != nil {
		return nil, err
	}
	defer os.Remove(tmp)
	b, err := os.ReadFile(tmp)
	if err != nil {
		return nil, err
	}
	return ParseVersionINI(string(b))
}
