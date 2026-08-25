// Package app 实现 bubbletea TUI：主菜单 + 各功能子页面。
// 设计参考 k9s / lazygit / btop / glow 的经典 Go TUI 布局：
// 全宽顶栏（品牌渐变 logo + 实时状态）+ 内容卡片 + 全宽底栏（快捷键），
// 克制配色、信息网格化，保留全部菜单选项。
package app

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// 品牌色（延续 mihomo UI 的粉色偏好 + 深色高级感）。
const (
	brandPink   = "#ec4899"
	brandCyan   = "#22d3ee"
	brandGreen  = "#34d399"
	brandYellow = "#fbbf24"
	brandRed    = "#f87171"
	brandDim    = "#94a3b8"
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

	// 页面/面板标题（青色加粗）
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(brandCyan))

	// 底栏按键高亮（粉色）
	barKey = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(brandPink))

	// 菜单选中项：粉色背景 + 白字（紧凑，无边框——卡片内嵌套 Border 会错乱）
	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffffff")).
				Background(lipgloss.Color(brandPink)).
				Padding(0, 1)

	// 普通菜单项文本
	itemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e8f0"))

	// 菜单编号（青色）
	numStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(brandCyan)).Render

	// 版本徽章
	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color(brandPink)).
			Padding(0, 1)

	// 内容卡片（紫色圆角边框）
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4c3a8f")).
			Padding(0, 2)

	// 卡片标题
	cardTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(brandCyan)).Render

	// 短标签（顶栏/信息卡状态行）
	shortLabel = lipgloss.NewStyle().Foreground(lipgloss.Color(brandDim)).Render

	// 结果页横幅
	successBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color(brandGreen)).
			Padding(0, 2)
	errorBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color(brandRed)).
			Padding(0, 2)
	infoBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color(brandCyan)).
			Padding(0, 2)
)

// ---------- 品牌渐变 ----------

// gradientText 在 from→to 色之间按字符做 HCL 感知渐变（glow/krab 风格 logo）。
func gradientText(s, from, to string) string {
	f, err1 := colorful.Hex(from)
	t, err2 := colorful.Hex(to)
	runes := []rune(s)
	if err1 != nil || err2 != nil || len(runes) == 0 {
		return s
	}
	var b strings.Builder
	for i, r := range runes {
		c := f.BlendHcl(t, float64(i)/float64(len(runes)-1))
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c.Hex())).Render(string(r)))
	}
	return b.String()
}

// logo 品牌渐变标识（粉 → 青）。
func logo() string { return gradientText("✦ BBRv3 Manager ✦", brandPink, brandCyan) }

// ---------- 底栏 ----------

// bottomBar 底部快捷键：粉色按键 + 灰色说明，上方短分隔线（跟随内容宽，不撑满）。
func (m Model) bottomBar(keys ...[2]string) string {
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, barKey.Render(k[0])+" "+dim(k[1]))
	}
	line := strings.Join(parts, "   ")
	rule := dim(strings.Repeat("─", lipgloss.Width(line)+2))
	return rule + "\n" + line
}

// ---------- 卡片内容 ----------

// blockWidth 多行字符串的最大行宽（ANSI 感知，取最长行）。
func blockWidth(s string) int {
	maxW := 0
	for _, l := range strings.Split(s, "\n") {
		if w := lipgloss.Width(l); w > maxW {
			maxW = w
		}
	}
	return maxW
}

// renderCard 渲染内容卡片（圆角紫边框，宽度跟随内容，不撑满）。
func (m Model) renderCard(content string) string {
	return cardStyle.Render(content)
}

// joinSections 用与卡片等宽的分隔线连接多个内容段（卡片内分区）。
// width <= 0 时按内容自然宽度计算。
func joinSections(width int, segs ...[]string) string {
	var lines []string
	for _, seg := range segs {
		lines = append(lines, seg...)
	}
	maxW := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > maxW {
			maxW = w
		}
	}
	if width <= 0 {
		width = maxW
	}
	rule := dim(strings.Repeat("─", maxInt(width, 12)))
	var b strings.Builder
	for i, seg := range segs {
		if i > 0 {
			b.WriteString("\n\n" + rule + "\n\n")
		}
		b.WriteString(strings.Join(seg, "\n"))
	}
	return b.String()
}

// statusRows 竖排状态行（信息卡内使用：label 灰色 + 值着色）。
func statusRows(algoVal, qdiscVal, kernel string) []string {
	return []string{
		shortLabel(i18n.T("menu.algoShort")+": ") + algoVal,
		shortLabel(i18n.T("menu.qdiscShort")+": ") + qdiscVal,
		shortLabel(i18n.T("menu.kernelShort")+": ") + kernel,
	}
}

// miniBar 迷你进度条（粉实心 + 灰空心，用于 CPU/内存）。
func miniBar(pct float64, width int) string {
	if width <= 0 {
		width = 10
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	fill := int(float64(width) * pct / 100)
	done := lipgloss.NewStyle().Foreground(lipgloss.Color(brandPink)).Render(strings.Repeat("█", fill))
	todo := dim(strings.Repeat("░", width-fill))
	return done + todo
}

// memSize 字节数转人类可读（GB，一位小数）。
func memSize(b uint64) string {
	return fmt.Sprintf("%.1fG", float64(b)/(1024*1024*1024))
}

// ---------- 选项列表 ----------

// itemRow 渲染用选项行。
type itemRow struct {
	num   string // 编号（含尾随点，如 "01."）
	label string // 已翻译标签
}

// padNum 编号补零成两位数（01-09）。
func padNum(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

// renderItems 渲染选项列表：未选中 = 青色编号 + 浅灰标签；
// 选中 = 粉色整行高亮白字（▸ 占位对齐，标签列与未选中行一致）。
func renderItems(items []itemRow, cursor int) string {
	var lines []string
	for i, it := range items {
		body := it.num + " " + it.label
		if i == cursor {
			lines = append(lines, selectedItemStyle.Render("▸ "+body))
		} else {
			lines = append(lines, "   "+numStyle(it.num)+" "+itemStyle.Render(it.label))
		}
	}
	return strings.Join(lines, "\n")
}

// ---------- 日志 / 进度 ----------

// progressBar 粉色进度条（下载任务）。
func progressBar(pct, width int) string {
	if width <= 0 {
		width = 30
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	fill := width * pct / 100
	done := lipgloss.NewStyle().Foreground(lipgloss.Color(brandPink)).Render(strings.Repeat("█", fill))
	todo := lipgloss.NewStyle().Foreground(lipgloss.Color("#3a3260")).Render(strings.Repeat("░", width-fill))
	return done + todo + "  " + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(brandPink)).Render(fmt.Sprintf("%d%%", pct))
}

// barWidth 进度条宽度：跟随终端宽度，夹在 [20, 60]。
func barWidth(w int) int {
	if w <= 0 {
		return 30
	}
	if n := w - 14; n < 20 {
		return 20
	} else if n > 60 {
		return 60
	} else {
		return n
	}
}

var pctRe = regexp.MustCompile(`(\d+)%`)

// lastPercent 从日志尾部找最近的百分比（下载进度行）。
func lastPercent(logs []string) (int, bool) {
	for i := len(logs) - 1; i >= 0; i-- {
		if m := pctRe.FindStringSubmatch(logs[i]); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				return n, true
			}
		}
	}
	return 0, false
}

// ---------- 结果页 ----------

// resultBanner 结果页横幅：✔ 成功绿 / ❌ 失败红 / 其他青。
func resultBanner(title string, failed bool) string {
	st := infoBanner
	switch {
	case failed || strings.ContainsAny(title, "✘❌"):
		st = errorBanner
	case strings.ContainsAny(title, "✔✓"):
		st = successBanner
	}
	return st.Render(title)
}

// ---------- 其他 ----------

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

// logsColored 着色日志行列表。
func logsColored(logs []string) []string {
	out := make([]string, 0, len(logs))
	for _, l := range logs {
		out = append(out, colorizeLog(l))
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
