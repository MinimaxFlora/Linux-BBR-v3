package system

import (
	"context"
	"os"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"
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
// 首次运行（install.sh 或直接执行二进制）时把当前程序复制到 /usr/local/bin/b，
// 之后 `b` 直接执行本地已安装版本，不再联网下载；
// 后续更新通过 TUI 菜单"检测 TUI 更新"完成（替换安装路径）。
func InstallQuickCommand(ctx context.Context, log execx.Logger) error {
	if skip := getenvBool("BBRV3_SKIP_QUICK_COMMAND"); skip {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		log.Logf(i18n.T("sys.quickFail"))
		return err
	}
	// 已通过 b 运行（自身就是安装路径）时无需复制
	if exe == bbr.QuickCommandPath {
		return nil
	}
	if err := copyFile(ctx, exe, bbr.QuickCommandPath, 0o755); err != nil {
		log.Logf(i18n.T("sys.quickFail"))
		return err
	}
	log.Logf(i18n.Tf("sys.quickInstalled", bbr.QuickCommandPath))
	return nil
}
