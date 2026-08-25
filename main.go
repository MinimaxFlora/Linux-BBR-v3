// Linux-BBR-v3 是 Actions-bbr-v3 的 Go + bubbletea TUI 重写版。
// 功能与原 shell 版完全一致（12 项菜单 + 安全缓解 + b 快捷命令），
// 品牌标识由 joeyblog 替换为 MinimaxFlora。
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/app"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/system"

	tea "github.com/charmbracelet/bubbletea"
)

// plainLogger 把日志输出到普通终端（TUI 启动前的依赖安装阶段）。
type plainLogger struct{}

func (plainLogger) Log(line string)             { fmt.Println(line) }
func (plainLogger) Logf(format string, a ...any) { fmt.Printf(format+"\n", a...) }

func main() {
	// 非 root 时通过 sudo 重新执行本程序（密码提示在 TUI 之前的正常终端）。
	if !system.IsRoot() {
		if _, err := exec.LookPath("sudo"); err != nil {
			fmt.Println("此程序需要 root 权限。请先安装 sudo，或直接以 root 运行。")
			os.Exit(1)
		}
		args := append([]string{"-E"}, os.Args[1:]...)
		cmd := exec.Command("sudo", args...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	ctx := context.Background()

	// 初始化语言（环境变量 BBRV3_LANG / /etc/bbrv3/lang）
	i18n.Init()

	// 开发模式：跳过 Debian/apt/架构检查，便于在非 Linux 机器预览 TUI。
	devMode := os.Getenv("BBRV3_DEV") == "1"

	// 仅支持 Debian/Ubuntu 系
	if !devMode {
		if err := system.RequireAPT(ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// 安装必需依赖（curl wget dpkg awk sed sysctl jq）
	if !devMode {
		if err := system.EnsureDeps(ctx, plainLogger{}); err != nil {
			fmt.Printf("依赖安装失败：%v\n", err)
			os.Exit(1)
		}
	}

	// 收集环境信息并校验架构
	env := system.LoadEnv(ctx)
	if !devMode {
		if err := system.CheckArch(env); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	app.SetEnv(env)
	app.SetKernelVersion(env.Arch + " / " + system.KernelRelease())

	// 启动 TUI
	p := tea.NewProgram(app.NewModel(ctx, env), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("程序异常退出：%v\n", err)
		os.Exit(1)
	}
}
