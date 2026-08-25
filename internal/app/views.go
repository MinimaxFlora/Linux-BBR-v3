package app

import (
	"fmt"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"
)

// View 渲染当前页面。
func (m Model) View() string {
	switch m.page {
	case PageBoot:
		return m.viewBoot()
	case PageMenu:
		return m.viewMenu()
	case PageQdisc:
		return m.viewQdisc()
	case PageProfile:
		return m.viewProfile()
	case PageConfirm:
		return m.viewConfirm()
	case PageInput:
		return m.viewInput()
	case PageVersion:
		return m.viewVersion()
	case PageLog:
		return m.viewLog()
	case PageResult:
		return m.viewResult()
	}
	return ""
}

// viewBoot 启动任务页。
func (m Model) viewBoot() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(header())
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(
		cyan(i18n.T("boot.init")) + "\n\n" +
			strings.Join(logsColored(m.logs), "\n") + "\n\n" +
			m.spinner.View()+" "+dim(i18n.T("boot.waiting")),
	))
	return b.String()
}

// viewMenu 主菜单：两个大框（顶部信息卡 + 菜单卡）。
func (m Model) viewMenu() string {
	env := currentEnv()

	// 顶部大框：品牌 + 欢迎 + 语言/版本 + 状态 + 作者
	var top strings.Builder
	top.WriteString(titleStyle.Render("✦ BBRv3 Manager ✦") + "  " + brandBadge.Render(Version))
	top.WriteString("\n\n")
	top.WriteString(dim("✧  ") + bold(i18n.T("boot.title")))
	top.WriteString("\n\n")
	top.WriteString(langStyle.Render("🌐 "+i18n.T("menu.lang")+": "+langLabel()) + "   " +
		dim(i18n.T("menu.ver")+": ") + bold(Version))
	top.WriteString("\n\n")
	top.WriteString(statusLine(env.CurrentAlgo, env.CurrentQdisc, kernelVersionHint()))
	top.WriteString("\n\n")
	top.WriteString(authorLine())

	// 菜单大框
	var items strings.Builder
	items.WriteString(headerStyle.Render(i18n.T("menu.choose")))
	items.WriteString("\n\n")
	for i, it := range menuItems {
		label := it.num + ". " + it.icon + " " + i18n.T(it.labelKey)
		if i == m.menuCursor {
			items.WriteString("  " + selectedItemStyle.Render("▸ "+label) + "\n")
		} else {
			items.WriteString("  " + itemStyle.Render("  "+label) + "\n")
		}
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(cardStyle.Render(strings.TrimSuffix(top.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.TrimSuffix(items.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render(i18n.T("menu.help")))
	b.WriteString("\n")
	return b.String()
}

// viewQdisc 队列算法子菜单（标题与选项合成一个卡片）。
func (m Model) viewQdisc() string {
	var body strings.Builder
	body.WriteString(headerStyle.Render("⚡ "+i18n.T("qdisc.title")))
	body.WriteString("\n\n")
	for i, opt := range qdiscOptions {
		label := opt.num + ". " + i18n.T(opt.labelKey)
		if i == m.qdiscCursor {
			body.WriteString("  " + selectedItemStyle.Render("▸ "+label) + "\n")
		} else {
			body.WriteString("  " + itemStyle.Render("  "+label) + "\n")
		}
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(miniHeader("⚡ "+i18n.T("qdisc.title")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.TrimSuffix(body.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render(i18n.T("qdisc.help")))
	b.WriteString("\n")
	return b.String()
}

// viewProfile 内核类型选择页（标题与选项合成一个卡片）。
func (m Model) viewProfile() string {
	var body strings.Builder
	body.WriteString(headerStyle.Render(i18n.T("profile.title")))
	body.WriteString("\n\n")
	items := []struct {
		num     string
		labelKey string
		warnKey  string
	}{
		{"1", "profile.item1", ""},
		{"2", "profile.item2", "profile.warn"},
	}
	for i, it := range items {
		label := it.num + ". " + i18n.T(it.labelKey)
		if i == m.profileCursor {
			body.WriteString("  " + selectedItemStyle.Render("▸ "+label) + "\n")
		} else {
			body.WriteString("  " + itemStyle.Render("  "+label) + "\n")
		}
		if i == 1 && m.profileCursor == 1 && it.warnKey != "" {
			body.WriteString("\n  " + yellow(i18n.T(it.warnKey)) + "\n")
		}
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(miniHeader(i18n.T("profile.title")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.TrimSuffix(body.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render(i18n.T("profile.help")))
	b.WriteString("\n")
	return b.String()
}

// viewConfirm 确认页。
func (m Model) viewConfirm() string {
	var lines []string
	lines = append(lines, yellow(i18n.T("confirm.title")))
	for _, p := range strings.Split(m.confirmPrompt, "\n") {
		lines = append(lines, cyan(p))
	}
	lines = append(lines, "", yellow(i18n.T("confirm.prompt")))
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(miniHeader(i18n.T("confirm.title")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.Join(lines, "\n")))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render(i18n.T("confirm.help")))
	b.WriteString("\n")
	return b.String()
}

// viewInput 输入页。
func (m Model) viewInput() string {
	var lines []string
	for _, p := range strings.Split(m.inputPrompt, "\n") {
		lines = append(lines, cyan(p))
	}
	lines = append(lines, "", m.input.View())
	if m.inputErr != "" {
		lines = append(lines, red("✘ "+m.inputErr))
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(miniHeader(i18n.T("menu.item4") + " / " + i18n.T("menu.item7")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.Join(lines, "\n")))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render(i18n.T("input.help")))
	b.WriteString("\n")
	return b.String()
}

// viewVersion 版本列表选择页（标题与列表合成一个卡片）。
func (m Model) viewVersion() string {
	var body strings.Builder
	body.WriteString(headerStyle.Render(i18n.T("version.title")))
	body.WriteString("\n\n")
	start := 0
	if len(m.tags) > 12 {
		start = m.tagCursor - 6
		if start < 0 {
			start = 0
		}
		if start+12 > len(m.tags) {
			start = len(m.tags) - 12
		}
	}
	for i := start; i < len(m.tags) && i < start+12; i++ {
		label := fmt.Sprintf("%2d. %s", i+1, m.tags[i])
		if i == m.tagCursor {
			body.WriteString("  " + selectedItemStyle.Render("▸ "+label) + "\n")
		} else {
			body.WriteString("  " + itemStyle.Render("  "+label) + "\n")
		}
	}
	if len(m.tags) > 12 {
		body.WriteString(dim(fmt.Sprintf(i18n.T("version.total"), len(m.tags))) + "\n")
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(miniHeader(i18n.T("version.title")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.TrimSuffix(body.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render(i18n.T("version.help")))
	b.WriteString("\n")
	return b.String()
}

// viewLog 任务日志页。
func (m Model) viewLog() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(miniHeader(m.logTitle))
	b.WriteString("\n\n")

	// 计算可见日志窗口
	visible := m.height - 10
	if visible < 5 {
		visible = 5
	}
	from := m.logScroll - visible + 1
	if from < 0 {
		from = 0
	}
	to := m.logScroll + 1
	if to > len(m.logs) {
		to = len(m.logs)
	}
	var body strings.Builder
	if len(m.logs) == 0 {
		body.WriteString(dim("（…）"))
	} else {
		for _, l := range m.logs[from:to] {
			body.WriteString(colorizeLog(l))
			body.WriteString("\n")
		}
	}
	b.WriteString(cardStyle.Render(strings.TrimSuffix(body.String(), "\n")))
	b.WriteString("\n\n")
	status := m.spinner.View() + " " + dim(i18n.T("log.running"))
	if !m.logAuto {
		status += dim(fmt.Sprintf(i18n.T("log.scroll"), m.logScroll+1, len(m.logs)))
	}
	b.WriteString(status)
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(i18n.T("log.help")))
	b.WriteString("\n")
	return b.String()
}

// viewResult 结果页。
func (m Model) viewResult() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(miniHeader(m.resultTitle))
	b.WriteString("\n\n")

	start := len(m.logs) - 15
	if start < 0 {
		start = 0
	}
	var body strings.Builder
	for _, l := range m.logs[start:] {
		body.WriteString(colorizeLog(l))
		body.WriteString("\n")
	}
	if m.taskErr != nil {
		body.WriteString(red("✘ " + m.taskErr.Error()))
		body.WriteString("\n")
	}
	b.WriteString(cardStyle.Render(strings.TrimSuffix(body.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(yellow(m.resultExtra))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render(i18n.T("result.enter") + "  ·  " + i18n.T("result.quit")))
	b.WriteString("\n")
	return b.String()
}

// logsColored 着色日志行。
func logsColored(logs []string) []string {
	out := make([]string, 0, len(logs))
	for _, l := range logs {
		out = append(out, colorizeLog(l))
	}
	return out
}
