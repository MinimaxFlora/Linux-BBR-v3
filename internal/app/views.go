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

// pageFrame 标准子页面框架：全宽顶栏（标题面包屑） + 内容卡 + 全宽底栏。
func (m Model) pageFrame(title, content string, keys ...[2]string) string {
	var b strings.Builder
	b.WriteString(m.topBar(title))
	b.WriteString("\n\n")
	b.WriteString(m.renderCard(content))
	b.WriteString("\n\n")
	b.WriteString(m.bottomBar(keys...))
	b.WriteString("\n")
	return b.String()
}

// viewBoot 启动任务页（顶栏 + 初始化卡片，无交互按键）。。
func (m Model) viewBoot() string {
	var lines []string
	lines = append(lines, cyan(i18n.T("boot.init")))
	if len(m.logs) > 0 {
		lines = append(lines, "")
		lines = append(lines, logsColored(m.logs)...)
	}
	lines = append(lines, "", pink(m.spinner.View())+"  "+dim(i18n.T("boot.waiting")))
	var b strings.Builder
	b.WriteString(m.topBar(""))
	b.WriteString("\n\n")
	b.WriteString(m.renderCard(strings.Join(lines, "\n")))
	b.WriteString("\n")
	return b.String()
}

// viewMenu 主菜单：信息卡 + 菜单卡 + 底栏。
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

	// 信息卡：欢迎语 / 作者·项目 / 状态（竖排分区）
	info := joinSections(0,
		[]string{dim("✧  ") + bold(i18n.T("boot.title"))},
		[]string{
			pink("♥  ") + dim(i18n.T("menu.authorLabel")+"：") + itemStyle.Render("MinimaxFlora"),
			cyan("◆  ") + dim(i18n.T("menu.repoLabel")+"：") + itemStyle.Render(i18n.T("menu.repoUrl")),
		},
		statusRows(algoVal, qdiscVal, cyan(kernelVersionHint())),
	)

	// 菜单卡：标题 + 选项列表
	rows := make([]itemRow, 0, len(menuItems))
	for _, it := range menuItems {
		rows = append(rows, itemRow{num: padNum(it.num) + ".", icon: it.icon, label: i18n.T(it.labelKey)})
	}
	menu := joinSections(0,
		[]string{cardTitle(i18n.T("menu.choose"))},
		strings.Split(renderItems(rows, m.menuCursor), "\n"),
	)

	var b strings.Builder
	b.WriteString(m.topBar(""))
	b.WriteString("\n\n")
	b.WriteString(m.renderCard(info))
	b.WriteString("\n\n")
	b.WriteString(m.renderCard(menu))
	b.WriteString("\n\n")
	b.WriteString(m.bottomBar(
		[2]string{"↑/↓", i18n.T("help.select")},
		[2]string{"1-9", i18n.T("help.run")},
		[2]string{"Enter", i18n.T("help.confirm")},
		[2]string{"L", i18n.T("help.lang")},
		[2]string{"q", i18n.T("help.quit")},
	))
	b.WriteString("\n")
	return b.String()
}

// viewQdisc 队列算法子菜单（原菜单 4-7 合并）。
func (m Model) viewQdisc() string {
	rows := make([]itemRow, 0, len(qdiscOptions))
	for _, opt := range qdiscOptions {
		rows = append(rows, itemRow{num: padNum(opt.num) + ".", icon: "🚦", label: i18n.T(opt.labelKey)})
	}
	content := joinSections(0,
		[]string{cardTitle("⚡ " + i18n.T("qdisc.title"))},
		strings.Split(renderItems(rows, m.qdiscCursor), "\n"),
	)
	return m.pageFrame("⚡ "+i18n.T("qdisc.title"), content,
		[2]string{"↑/↓", i18n.T("help.select")},
		[2]string{"1-4", i18n.T("help.run")},
		[2]string{"Enter", i18n.T("help.confirm")},
		[2]string{"Esc", i18n.T("help.back")},
		[2]string{"q", i18n.T("help.quit")},
	)
}

// viewProfile 内核类型选择页（1 标准 / 2 Max）。
func (m Model) viewProfile() string {
	rows := []itemRow{
		{num: "01.", icon: "🚀", label: i18n.T("profile.item1")},
		{num: "02.", icon: "⚡", label: i18n.T("profile.item2")},
	}
	lines := strings.Split(renderItems(rows, m.profileCursor), "\n")
	if m.profileCursor == 1 {
		lines = append(lines, "", yellow(i18n.T("profile.warn")))
	}
	content := joinSections(0,
		[]string{cardTitle(i18n.T("profile.title"))},
		lines,
	)
	return m.pageFrame(i18n.T("profile.title"), content,
		[2]string{"↑/↓", i18n.T("help.select")},
		[2]string{"Enter", i18n.T("help.confirm")},
		[2]string{"Esc", i18n.T("help.back")},
		[2]string{"q", i18n.T("help.quit")},
	)
}

// viewConfirm 确认页。
func (m Model) viewConfirm() string {
	var lines []string
	for _, p := range strings.Split(m.confirmPrompt, "\n") {
		lines = append(lines, itemStyle.Render(p))
	}
	lines = append(lines, "", yellow(i18n.T("confirm.prompt")))
	content := joinSections(0,
		[]string{cardTitle(i18n.T("confirm.title"))},
		lines,
	)
	return m.pageFrame(i18n.T("confirm.title"), content,
		[2]string{"y", i18n.T("help.yes")},
		[2]string{"n", i18n.T("help.no")},
		[2]string{"Esc", i18n.T("help.cancel")},
	)
}

// viewInput 输入页（智能优化带宽 / RTT）。
func (m Model) viewInput() string {
	var lines []string
	for _, p := range strings.Split(m.inputPrompt, "\n") {
		lines = append(lines, itemStyle.Render(p))
	}
	lines = append(lines, "", m.input.View())
	if m.inputErr != "" {
		lines = append(lines, "", red("✘ "+m.inputErr))
	}
	content := joinSections(0,
		[]string{cardTitle(i18n.T("menu.item7"))},
		lines,
	)
	return m.pageFrame(i18n.T("menu.item7"), content,
		[2]string{"Enter", i18n.T("help.submit")},
		[2]string{"Esc", i18n.T("help.back")},
	)
}

// viewVersion 版本列表选择页（窗口滚动，编号直达）。
func (m Model) viewVersion() string {
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
	rows := make([]itemRow, 0, len(m.tags))
	for i := start; i < len(m.tags) && i < start+12; i++ {
		rows = append(rows, itemRow{num: fmt.Sprintf("%d.", i+1), label: m.tags[i]})
	}
	lines := strings.Split(renderItems(rows, m.tagCursor-start), "\n")
	if len(m.tags) > 12 {
		lines = append(lines, "", dim(fmt.Sprintf(i18n.T("version.total"), len(m.tags))))
	}
	content := joinSections(0,
		[]string{cardTitle(i18n.T("version.title"))},
		lines,
	)
	return m.pageFrame(i18n.T("version.title"), content,
		[2]string{"↑/↓", i18n.T("help.select")},
		[2]string{"Num", i18n.T("help.jump")},
		[2]string{"Enter", i18n.T("help.install")},
		[2]string{"Esc", i18n.T("help.back")},
	)
}

// viewLog 任务日志页（下载时显示进度条）。
func (m Model) viewLog() string {
	visible := m.height - 12
	if visible < 5 {
		visible = 5
	}
	pct, hasPct := lastPercent(m.logs)
	if hasPct {
		visible -= 2
		if visible < 3 {
			visible = 3
		}
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
		body.WriteString(dim("…"))
	} else {
		for _, l := range m.logs[from:to] {
			body.WriteString(colorizeLog(l))
			body.WriteString("\n")
		}
	}

	var b strings.Builder
	b.WriteString(m.topBar(m.logTitle))
	b.WriteString("\n\n")
	if hasPct {
		b.WriteString(progressBar(pct, barWidth(m.width)))
		b.WriteString("\n\n")
	}
	b.WriteString(m.renderCard(strings.TrimSuffix(body.String(), "\n")))
	b.WriteString("\n\n")
	status := pink(m.spinner.View()) + " " + dim(i18n.T("log.running"))
	if !m.logAuto {
		status += dim(fmt.Sprintf(i18n.T("log.scroll"), m.logScroll+1, len(m.logs)))
	}
	b.WriteString(status)
	b.WriteString("\n\n")
	b.WriteString(m.bottomBar(
		[2]string{"↑/↓", i18n.T("help.scroll")},
		[2]string{"End", i18n.T("help.resume")},
		[2]string{"q", i18n.T("help.quit")},
	))
	b.WriteString("\n")
	return b.String()
}

// viewResult 结果页：横幅 + 日志尾部 + 附加提示。
func (m Model) viewResult() string {
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

	var b strings.Builder
	b.WriteString(m.topBar(m.resultTitle))
	b.WriteString("\n\n")
	b.WriteString(resultBanner(m.resultTitle, m.taskErr != nil))
	b.WriteString("\n\n")
	b.WriteString(m.renderCard(strings.TrimSuffix(body.String(), "\n")))
	b.WriteString("\n\n")
	b.WriteString(yellow(m.resultExtra))
	b.WriteString("\n\n")
	b.WriteString(m.bottomBar(
		[2]string{"Enter", i18n.T("help.back")},
		[2]string{"q", i18n.T("help.quit")},
	))
	b.WriteString("\n")
	return b.String()
}
