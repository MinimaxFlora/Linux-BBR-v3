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
	content := cardStyle.Render(
		cyan(i18n.T("boot.init")) + "\n\n" +
			strings.Join(logsColored(m.logs), "\n") + "\n\n" +
			m.spinner.View()+" "+dim(i18n.T("boot.waiting")),
	)
	return "\n" + content + "\n\n"
}

// viewMenu 主菜单：顶栏 + 信息网格卡 + 菜单卡 + 底栏。
func (m Model) viewMenu() string {
	env := currentEnv()

	// 状态值（空值占位；非 bbr 黄色警示）
	algoVal := green(env.CurrentAlgo)
	if env.CurrentAlgo == "" {
		algoVal = dim("—")
	} else if env.CurrentAlgo != "bbr" {
		algoVal = yellow(env.CurrentAlgo)
	}
	qdiscVal := green(env.CurrentQdisc)
	if env.CurrentQdisc == "" {
		qdiscVal = dim("—")
	}

	// 信息卡：欢迎语 + 作者/项目 + 分隔线 + 紧凑状态行
	info := fmt.Sprintf("%s\n\n%s\n%s\n\n%s",
		dim("✧  ")+bold(i18n.T("boot.title")),
		pink("♥  ")+i18n.T("menu.author"),
		cyan("◆  ")+i18n.T("menu.repo"),
		statusCompact(algoVal, qdiscVal, cyan(kernelVersionHint())),
	)

	// 菜单卡
	var items strings.Builder
	items.WriteString(cardTitle(i18n.T("menu.choose")))
	items.WriteString("\n\n")
	for i, it := range menuItems {
		label := it.icon + " " + i18n.T(it.labelKey)
		if i == m.menuCursor {
			items.WriteString("  " + selectedItemStyle.Render("▸ "+it.num+". "+label) + "\n")
		} else {
			items.WriteString("  " + itemStyle.Render("  "+numStyle(it.num)+". "+label) + "\n")
		}
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.topBar(""))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.TrimSuffix(info, "\n")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.TrimSuffix(items.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(bottomBar(
		[2]string{"↑/↓", i18n.T("help.select")},
		[2]string{"1-9", i18n.T("help.run")},
		[2]string{"Enter", i18n.T("help.confirm")},
		[2]string{"L", i18n.T("help.lang")},
		[2]string{"q", i18n.T("help.quit")},
	))
	b.WriteString("\n")
	return b.String()
}

// viewQdisc 队列算法子菜单。
func (m Model) viewQdisc() string {
	var items strings.Builder
	items.WriteString(cardTitle("⚡ " + i18n.T("qdisc.title")))
	items.WriteString("\n\n")
	for i, opt := range qdiscOptions {
		label := i18n.T(opt.labelKey)
		if i == m.qdiscCursor {
			items.WriteString("  " + selectedItemStyle.Render("▸ "+opt.num+". "+label) + "\n")
		} else {
			items.WriteString("  " + itemStyle.Render("  "+numStyle(opt.num)+". "+label) + "\n")
		}
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.topBar("⚡ " + i18n.T("qdisc.title")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.TrimSuffix(items.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(bottomBar(
		[2]string{"↑/↓", i18n.T("help.select")},
		[2]string{"1-4", i18n.T("help.run")},
		[2]string{"Enter", i18n.T("help.confirm")},
		[2]string{"Esc", i18n.T("help.back")},
		[2]string{"q", i18n.T("help.quit")},
	))
	b.WriteString("\n")
	return b.String()
}

// viewProfile 内核类型选择页。
func (m Model) viewProfile() string {
	var items strings.Builder
	items.WriteString(cardTitle(i18n.T("profile.title")))
	items.WriteString("\n\n")
	rows := []struct {
		num      string
		labelKey string
		warnKey  string
	}{
		{"1", "profile.item1", ""},
		{"2", "profile.item2", "profile.warn"},
	}
	for i, it := range rows {
		label := i18n.T(it.labelKey)
		if i == m.profileCursor {
			items.WriteString("  " + selectedItemStyle.Render("▸ "+it.num+". "+label) + "\n")
		} else {
			items.WriteString("  " + itemStyle.Render("  "+numStyle(it.num)+". "+label) + "\n")
		}
		if i == 1 && m.profileCursor == 1 && it.warnKey != "" {
			items.WriteString("\n  " + yellow(i18n.T(it.warnKey)) + "\n")
		}
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.topBar(i18n.T("profile.title")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.TrimSuffix(items.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(bottomBar(
		[2]string{"↑/↓", i18n.T("help.select")},
		[2]string{"Enter", i18n.T("help.confirm")},
		[2]string{"Esc", i18n.T("help.back")},
		[2]string{"q", i18n.T("help.quit")},
	))
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
	b.WriteString(m.topBar(i18n.T("confirm.title")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.Join(lines, "\n")))
	b.WriteString("\n\n")
	b.WriteString(bottomBar(
		[2]string{"y", i18n.T("help.yes")},
		[2]string{"n", i18n.T("help.no")},
		[2]string{"Esc", i18n.T("help.cancel")},
	))
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
	b.WriteString(m.topBar(i18n.T("menu.item7")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.Join(lines, "\n")))
	b.WriteString("\n\n")
	b.WriteString(bottomBar(
		[2]string{"Enter", i18n.T("help.submit")},
		[2]string{"Esc", i18n.T("help.back")},
	))
	b.WriteString("\n")
	return b.String()
}

// viewVersion 版本列表选择页。
func (m Model) viewVersion() string {
	var items strings.Builder
	items.WriteString(cardTitle(i18n.T("version.title")))
	items.WriteString("\n\n")
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
			items.WriteString("  " + selectedItemStyle.Render("▸ "+label) + "\n")
		} else {
			items.WriteString("  " + itemStyle.Render("  "+numStyle(label)) + "\n")
		}
	}
	if len(m.tags) > 12 {
		items.WriteString(dim(fmt.Sprintf(i18n.T("version.total"), len(m.tags))) + "\n")
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.topBar(i18n.T("version.title")))
	b.WriteString("\n\n")
	b.WriteString(cardStyle.Render(strings.TrimSuffix(items.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(bottomBar(
		[2]string{"↑/↓", i18n.T("help.select")},
		[2]string{"Num", i18n.T("help.jump")},
		[2]string{"Enter", i18n.T("help.install")},
		[2]string{"Esc", i18n.T("help.back")},
	))
	b.WriteString("\n")
	return b.String()
}

// viewLog 任务日志页。
func (m Model) viewLog() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.topBar(m.logTitle))
	b.WriteString("\n\n")

	// 计算可见日志窗口
	visible := m.height - 8
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
	b.WriteString("\n\n")
	b.WriteString(bottomBar(
		[2]string{"↑/↓", i18n.T("help.scroll")},
		[2]string{"End", i18n.T("help.resume")},
		[2]string{"q", i18n.T("help.quit")},
	))
	b.WriteString("\n")
	return b.String()
}

// viewResult 结果页。
func (m Model) viewResult() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.topBar(m.resultTitle))
	b.WriteString("\n\n")

	// 成功/失败标题行
	if m.taskErr != nil {
		b.WriteString(red("✘ " + i18n.T("common.failed")))
	} else {
		b.WriteString(green("✔ " + i18n.T("common.success")))
	}
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
	b.WriteString(bottomBar(
		[2]string{"Enter", i18n.T("help.back")},
		[2]string{"q", i18n.T("help.quit")},
	))
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
