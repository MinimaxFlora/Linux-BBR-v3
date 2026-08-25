package app

import (
	"fmt"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
)

// View 渲染当前页面。
func (m Model) View() string {
	switch m.page {
	case PageBoot:
		return m.viewBoot()
	case PageMenu:
		return m.viewMenu()
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
	lines := []string{
		"",
		titleStyle.Render("(☆ω☆)✧*｡ 欢迎来到 BBR 管理脚本世界哒！ ✧*｡(☆ω☆)"),
		"",
		separator,
		cyan("正在初始化：安装快捷命令 + 应用安全缓解策略..."),
		"",
	}
	for _, l := range m.logs {
		lines = append(lines, colorizeLog(l))
	}
	lines = append(lines, "", m.spinner.View()+dim(" 请稍候..."))
	return strings.Join(lines, "\n")
}

// viewMenu 主菜单页。
func (m Model) viewMenu() string {
	env := currentEnv()
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("(☆ω☆)✧*｡ 欢迎来到 BBR 管理脚本世界哒！ ✧*｡(☆ω☆)"))
	b.WriteString("\n")
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s%s\n", cyan("当前 TCP 拥塞控制算法："), green(env.CurrentAlgo)))
	b.WriteString(fmt.Sprintf("%s%s\n", cyan("当前队列管理算法：    "), green(env.CurrentQdisc)))
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString(yellow(fmt.Sprintf("作者：MinimaxFlora  |  项目：%s", "github.com/"+bbr.RepoFullName())))
	b.WriteString("\n")
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("╭( ･ㅂ･)و ✧ 你可以选择以下操作哦："))
	b.WriteString("\n")

	for i, it := range menuItems {
		label := it.num + ". " + it.icon + " " + it.label
		if i == m.menuCursor {
			b.WriteString("  " + selectedItemStyle.Render("▸ "+label))
		} else {
			b.WriteString("  " + itemStyle.Render("  "+label))
		}
		b.WriteString("\n")
	}
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ 选择  ·  数字键 1-12 直接执行  ·  Enter 确认  ·  q 退出"))
	return b.String()
}

// viewProfile 内核类型选择页。
func (m Model) viewProfile() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("请选择要安装的内核类型："))
	b.WriteString("\n\n")
	items := []struct {
		num     string
		label   string
		warning string
	}{
		{"1", "BBR v3 标准版（推荐日常使用）", ""},
		{"2", "BBR v3 Max 激进吞吐版（自有链路测速实验）", "警告：BBR v3 Max 会提高探测和窗口策略的进攻性，但仍保留 loss/ECN/inflight 反馈闭环；只适合自有链路吞吐测试，不建议日常生产使用。"},
	}
	for i, it := range items {
		label := it.num + ". " + it.label
		if i == m.profileCursor {
			b.WriteString("  " + selectedItemStyle.Render("▸ "+label) + "\n")
		} else {
			b.WriteString("  " + itemStyle.Render("  "+label) + "\n")
		}
		if i == 1 && m.profileCursor == 1 {
			b.WriteString("\n  " + yellow(it.warning) + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ 选择  ·  Enter 确认  ·  Esc 返回  ·  q 退出"))
	return b.String()
}

// viewConfirm 确认页。
func (m Model) viewConfirm() string {
	lines := []string{"\n", yellow("╭( ･ㅂ･)و ✧ 确认操作"), "\n"}
	parts := strings.Split(m.confirmPrompt, "\n")
	for _, p := range parts {
		lines = append(lines, cyan(p))
	}
	lines = append(lines, "", yellow("请确认 (y/n): "), "",
		helpStyle.Render("y/Enter 确认  ·  n/Esc 取消"))
	return strings.Join(lines, "\n")
}

// viewInput 输入页。
func (m Model) viewInput() string {
	var b strings.Builder
	b.WriteString("\n")
	for _, p := range strings.Split(m.inputPrompt, "\n") {
		b.WriteString(cyan(p))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")
	if m.inputErr != "" {
		b.WriteString(red("✘ " + m.inputErr))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Enter 提交  ·  Esc 返回主菜单"))
	return b.String()
}

// viewVersion 版本列表选择页。
func (m Model) viewVersion() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("以下为适用于当前架构的版本："))
	b.WriteString("\n\n")
	// 可视窗口：最多显示 12 行
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
			b.WriteString("  " + selectedItemStyle.Render("▸ "+label) + "\n")
		} else {
			b.WriteString("  " + itemStyle.Render("  "+label) + "\n")
		}
	}
	if len(m.tags) > 12 {
		b.WriteString(dim(fmt.Sprintf("  （共 %d 个版本）", len(m.tags))) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ 选择  ·  数字键直接选择  ·  Enter 安装  ·  Esc 返回"))
	return b.String()
}

// viewLog 任务日志页。
func (m Model) viewLog() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render(m.logTitle))
	b.WriteString("\n")
	b.WriteString(separator)
	b.WriteString("\n")

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
	if len(m.logs) == 0 {
		b.WriteString(dim("（无输出）"))
		b.WriteString("\n")
	} else {
		for _, l := range m.logs[from:to] {
			b.WriteString(colorizeLog(l))
			b.WriteString("\n")
		}
	}
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString(m.spinner.View() + dim(" 执行中...  "))
	if !m.logAuto {
		b.WriteString(dim(fmt.Sprintf("（已暂停自动滚动，第 %d/%d 行）", m.logScroll+1, len(m.logs))))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ 滚动  ·  End 恢复自动滚动  ·  q 退出"))
	return b.String()
}

// viewResult 结果页。
func (m Model) viewResult() string {
	var b strings.Builder
	b.WriteString("\n")
	if m.taskErr != nil {
		b.WriteString(red(m.resultTitle))
	} else {
		b.WriteString(green(m.resultTitle))
	}
	b.WriteString("\n")
	b.WriteString(separator)
	b.WriteString("\n")

	// 展示最近日志（最后 15 行）
	start := len(m.logs) - 15
	if start < 0 {
		start = 0
	}
	for _, l := range m.logs[start:] {
		b.WriteString(colorizeLog(l))
		b.WriteString("\n")
	}
	if m.taskErr != nil {
		b.WriteString(red("✘ " + m.taskErr.Error()))
		b.WriteString("\n")
	}
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString(yellow(m.resultExtra))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Enter 返回主菜单  ·  q 退出"))
	return b.String()
}
