// Package app 实现 bubbletea TUI：主菜单 + 各功能子页面。
package app

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// 样式主题（贴近原脚本配色：青/绿/黄/粉，TUI 化）。
var (
	cyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render
	green   = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render
	yellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render
	red     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render
	pink    = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Render
	dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render
	white   = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render
	boldPink = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213")).Render

	// 分隔线
	separator = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).
			Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	titleStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("213"))

	headerStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("39"))

	selectedItemStyle = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color("39")).
				Background(lipgloss.Color("237")).
				Padding(0, 1)

	itemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

// colorizeLog 按行首标记为日志行着色（✔ 绿 / ✘ 红 / ⚠ 黄 / 其他原色）。
func colorizeLog(line string) string {
	switch {
	case len(line) >= 2 && line[:2] == "✔":
		return green(line)
	case len(line) >= 2 && line[:2] == "✘":
		return red(line)
	case len(line) >= 2 && line[:2] == "⚠":
		return yellow(line)
	case line == "":
		return ""
	default:
		return dim(line)
	}
}

// fmtLog 便捷格式化（供任务日志使用，颜色留给视图层）。
func fmtLog(format string, args ...any) string { return fmt.Sprintf(format, args...) }
