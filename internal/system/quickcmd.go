package system

import (
	"context"
	"fmt"
	"os"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
)

// getenv 读取环境变量。
func getenv(key string) string { return os.Getenv(key) }

// getenvBool 读取布尔环境变量（"1"/"true" 为真）。
func getenvBool(key string) bool {
	switch getenv(key) {
	case "1", "true", "TRUE":
		return true
	}
	return false
}

// InstallQuickCommand 安装 /usr/local/bin/b 快捷命令（对应 install_quick_command）。
// Go 版快捷命令每次联网从 GitHub Releases 拉取最新编译好的二进制后执行，
// 不再使用本地缓存脚本。
func InstallQuickCommand(ctx context.Context, log execx.Logger) error {
	if skip := getenvBool("BBRV3_SKIP_QUICK_COMMAND"); skip {
		return nil
	}

	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
export BBRV3_SKIP_QUICK_COMMAND=1

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  ASSET="%s-linux-amd64" ;;
  aarch64) ASSET="%s-linux-arm64" ;;
  *)
    echo "不支持的架构: $ARCH" >&2
    exit 1
    ;;
esac

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT
BIN="$TMPDIR/%s"

if ! curl -fsSL -o "$BIN" "https://github.com/%s/releases/download/bbrv3-cli/$ASSET"; then
  echo "下载最新版失败，请检查网络连接。" >&2
  exit 1
fi
chmod +x "$BIN"
exec "$BIN"
`, bbr.BinaryAssetBase, bbr.BinaryAssetBase, bbr.BinaryAssetBase, bbr.RepoFullName())

	if err := writeFile(ctx, bbr.QuickCommandPath, script, 0o755); err != nil {
		log.Logf("提示：快捷命令 b 安装失败，不影响当前运行。")
		return err
	}
	return nil
}
