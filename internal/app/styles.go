// Package app 实现 bubbletea TUI：主菜单 + 各功能子页面。
// 设计参考 k9s / lazygit / btop 的经典 TUI 布局：
// 顶部状态栏 + 内容区 + 底部快捷键栏，克制配色，信息网格化。
package app

import (
	"fmt"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"

	"github.com/charmbracelet/lipgloss"
)

// 品牌色（延续 mihomo UI 的粉色偏好 + 深色高级感）。
const (
	brandPink   = "#ec4899"
	brandCyan   = "#22d3ee"
	brandPurple = "#a78bfa"
	brandGreen  = "#34d399"
	brandYellow = "#fbbf24"
	brandRed    = "#f87171"
	brandDim    = "#94a3b8"
	// 深色背景基调
	bgTop    = "#1e1b3a" // 顶栏深紫
	bgBottom = "#141228" // 底栏更深
	bgAccent = "#2d2a55" // 选中背景
)

// 样式主题。
var (
	// 文本色
	cyan   = lipgloss.NewStyle().Foreground(lipgloss.Color(brandCyan)).Render
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color(brandGreen)).Render
	yellow = lipgloss.NewStyle().Foreground(lipgloss.Color(brandYellow)).Render
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color(brandRed)).Render
	pink   = lipgloss.NewStyle().Foreground(lipgloss.Color(brandPink)).Render
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color(brandDim)).Render
	bold   = lipgloss.NewStyle().Bold(true).Render

	// 品牌标题（粉色）
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(brandPink))

	// 页面标题（青色）
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(brandCyan))

	// 顶部状态栏（整条深紫背景）
	topBarStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#e2e8f0")).
			Background(lipgloss.Color(bgTop)).
			Padding(0, 1)

	// 底部快捷键栏（整条深色背景）
	bottomBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cbd5e1")).
			Background(lipgloss.Color(bgBottom)).
			Padding(0, 1)

	// 底栏按键高亮（粉色）
	barKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(brandPink))

	// 菜单选中项：粉色背景 + 白字（紧凑，无边框）
	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffffff")).
				Background(lipgloss.Color(brandPink)).
				Padding(0, 1)

	// 普通菜单项
	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e2e8f0")).
			Padding(0, 1)

	// 菜单编号（青色）
	numStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(brandCyan)).
			Render

	// 版本徽章
	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color(brandPink)).
			Padding(0, 1)

	// 内容卡片（紫色圆角边框，克制）
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4c3a8f")).
			Padding(0, 2)

	// 卡片标题（内嵌小节标题）
	cardTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(brandCyan)).
			Render

	// 信息网格 label（灰色，统一宽度 16 列）
	infoLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(brandDim)).
			Width(16).
			Render

	// 短标签（信息卡状态行）
	shortLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(brandDim)).
			Render

	// 帮助行（备用）
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(brandDim))
)

// topBar 顶部状态栏：✦ BBRv3 v1.0.0  ❯ 页面标题  │  TCP/队列/内核  │  🌐 语言
func (m Model) topBar(pageTitle string) string {
	env := currentEnv()

	left := titleStyle.Render("✦ BBRv3") + " " + badgeStyle.Render(Version)
	if pageTitle != "" {
		left += dim("  ❯  ") + headerStyle.Render(pageTitle)
	}

	algo := env.CurrentAlgo
	if algo == "" {
		algo = "—"
	}
	algoVal := green(algo)
	if algo != "bbr" && algo != "—" {
		algoVal = yellow(algo)
	}
	qdisc := env.CurrentQdisc
	if qdisc == "" {
		qdisc = "—"
	}
	state := fmt.Sprintf("TCP: %s  队列: %s  内核: %s",
		algoVal, green(qdisc), cyan(kernelVersionHint()))

	lang := langLabel()

	return topBarStyle.Render(strings.Join(
		[]string{left, state, "🌐 " + lang},
		dim("  │  "),
	))
}

// bottomBar 底部快捷键栏（k9s 风格：整条深色背景 + 粉色按键）。
func bottomBar(keys ...[2]string) string {
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, barKey.Render(k[0])+" "+dim(k[1]))
	}
	return bottomBarStyle.Render(strings.Join(parts, "   "))
}

// statusCompact 紧凑状态行（短标签一行：TCP 算法 / 队列算法 / 内核）。
func statusCompact(algoVal, qdiscVal, kernel string) string {
	return fmt.Sprintf("%s %s    %s %s    %s %s",
		shortLabel(i18n.T("menu.algoShort")+": "), algoVal,
		shortLabel(i18n.T("menu.qdiscShort")+": "), qdiscVal,
		shortLabel(i18n.T("menu.kernelShort")+": "), kernel,
	)
}

// miniHeader 子页面顶部的小标题行（顶栏内已含标题时可不使用）。
func miniHeader(title string) string {
	return titleStyle.Render("✦ BBRv3") + " " + badgeStyle.Render(Version) +
		dim("  ❯  ") + headerStyle.Render(title)
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
