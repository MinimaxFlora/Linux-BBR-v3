// Package app 实现 bubbletea TUI：主菜单 + 各功能子页面。
package app

import (
	"fmt"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"

	"github.com/charmbracelet/lipgloss"
)

// 品牌色（用户偏好粉色系，延续 mihomo UI 的 #ec4899）。
const (
	brandPink   = "#ec4899"
	brandCyan   = "#22d3ee"
	brandPurple = "#a78bfa"
	brandGreen  = "#34d399"
	brandYellow = "#fbbf24"
	brandRed    = "#f87171"
	brandDim    = "#9ca3af"
	brandBg     = "#1f1147"
)

// 样式主题：深色渐变卡片 + 粉色品牌色。
var (
	// 文本色
	cyan   = lipgloss.NewStyle().Foreground(lipgloss.Color(brandCyan)).Render
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color(brandGreen)).Render
	yellow = lipgloss.NewStyle().Foreground(lipgloss.Color(brandYellow)).Render
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color(brandRed)).Render
	pink   = lipgloss.NewStyle().Foreground(lipgloss.Color(brandPink)).Render
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color(brandDim)).Render
	bold   = lipgloss.NewStyle().Bold(true).Render

	// 品牌标题（粉色渐变强调）
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(brandPink))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(brandCyan))

	// 菜单选中项：粉色高亮块（无边框，避免卡片内嵌套边框错乱）
	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffffff")).
				Background(lipgloss.Color(brandPink)).
				Padding(0, 1)

	// 普通菜单项
	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(brandYellow)).
			Padding(0, 1)

	// 卡片容器：渐变紫粉色边框
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(brandPurple)).
			Padding(0, 2)

	// 状态徽章
	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color(brandCyan)).
			Padding(0, 1)

	// 品牌徽章（粉色）
	brandBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color(brandPink)).
			Padding(0, 1)

	// 页脚帮助
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(brandDim))

	// 分隔线
	separator = lipgloss.NewStyle().
			Foreground(lipgloss.Color(brandPurple)).
			Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 语言徽章（当前语言高亮）
	langStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(brandCyan)).
			Bold(true)
)

// header 渲染品牌头部卡片（主菜单顶部大框：品牌+欢迎+状态+作者）。
func header() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("✦ BBRv3 Manager ✦") + "  " + brandBadge.Render(Version))
	b.WriteString("\n\n")
	b.WriteString(dim("✧  ") + bold(i18n.T("boot.title")))
	b.WriteString("\n\n")
	b.WriteString(langStyle.Render("🌐 "+i18n.T("menu.lang")+": "+langLabel()) + "   " +
		dim(i18n.T("menu.ver")+": ") + bold(Version))
	return cardStyle.Render(b.String())
}

// statusBlock 竖排状态块：label: value 每行一个（TCP/队列/内核）。
func statusBlock(algo, qdisc, kernel string) string {
	return fmt.Sprintf("%s\n%s\n%s",
		dim(i18n.T("menu.algo")+": ") + algo,
		dim(i18n.T("menu.qdisc")+": ") + qdisc,
		dim(i18n.T("menu.kernel")+": ") + kernel)
}

// divider 信息卡内的细分隔线。
var divider = strings.Repeat("─", 40)

// miniHeader 子页面顶部的小标题行（避免每页重复大卡片）。
func miniHeader(title string) string {
	return titleStyle.Render("✦ BBRv3 Manager ✦") + "  " + brandBadge.Render(Version) +
		"   " + dim("|") + "   " + headerStyle.Render(title)
}

// langLabel 返回当前语言显示名。
func langLabel() string {
	if i18n.IsEn() {
		return "English"
	}
	return "中文"
}

// kernelVersionHint 从环境读取内核版本显示（尽力而为）。
func kernelVersionHint() string {
	if k := kernelVersion; k != "" {
		return k
	}
	return "7.x"
}

// colorizeLog 按行首标记为日志行着色。
func colorizeLog(line string) string {
	switch {
	case strings.HasPrefix(line, "✔"):
		return green(line)
	case strings.HasPrefix(line, "✘"):
		return red(line)
	case strings.HasPrefix(line, "⚠"):
		return yellow(line)
	case line == "":
		return ""
	default:
		return dim(line)
	}
}

// fmtLog 便捷格式化。
func fmtLog(format string, args ...any) string { return fmt.Sprintf(format, args...) }
