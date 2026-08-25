package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// menuEntry 主菜单项（与 install.sh 的 12 项一一对应）。
type menuEntry struct {
	num   string
	icon  string
	label string
	act   func(Model) (Model, tea.Cmd)
}

var menuItems = []menuEntry{
	{num: " 1", icon: "🚀", label: "安装或更新 BBR v3 (最新版)", act: menuInstallLatest},
	{num: " 2", icon: "📚", label: "指定版本安装", act: menuInstallSpecific},
	{num: " 3", icon: "🔍", label: "检查 BBR v3 状态", act: menuCheckStatus},
	{num: " 4", icon: "⚡", label: "启用 BBR + FQ", act: menuQdisc("fq")},
	{num: " 5", icon: "⚡", label: "启用 BBR + FQ_CODEL", act: menuQdisc("fq_codel")},
	{num: " 6", icon: "⚡", label: "启用 BBR + FQ_PIE", act: menuQdisc("fq_pie")},
	{num: " 7", icon: "⚡", label: "启用 BBR + CAKE", act: menuQdisc("cake")},
	{num: " 8", icon: "🌏", label: "亚太机器 TCP 调优", act: menuAPACTuning},
	{num: " 9", icon: "🗑️", label: "卸载 BBR 内核", act: menuUninstall},
	{num: "10", icon: "🧠", label: "BBR v3 智能带宽优化", act: menuSmartTuning},
	{num: "11", icon: "🧹", label: "清空网络优化配置", act: menuClearOptimizations},
	{num: "12", icon: "🧨", label: "BBR v3 疯批模式（极限测速挑战）", act: menuExtremeMode},
}

// handleKey 处理当前页面的按键。
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.page {
	case PageMenu:
		return m.handleMenuKey(msg)
	case PageProfile:
		return m.handleProfileKey(msg)
	case PageConfirm:
		return m.handleConfirmKey(msg)
	case PageInput:
		return m.handleInputKey(msg)
	case PageVersion:
		return m.handleVersionKey(msg)
	case PageResult:
		return m.handleResultKey(msg)
	}
	return m, nil
}

// handleMenuKey 主菜单按键：数字直接触发，↑↓+Enter 选择，q 退出。
func (m Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
		return m, nil
	case "down", "j":
		if m.menuCursor < len(menuItems)-1 {
			m.menuCursor++
		}
		return m, nil
	case "enter":
		it := menuItems[m.menuCursor]
		return it.act(m)
	case "0":
		return m, tea.Quit
	}
	// 数字键 1-12 直接触发
	for _, it := range menuItems {
		if msg.String() == it.num[1:] {
			return it.act(m)
		}
	}
	return m, nil
}

// handleProfileKey 内核类型选择（1 标准 / 2 Max）。
func (m Model) handleProfileKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.profileCursor > 0 {
			m.profileCursor--
		}
		return m, nil
	case "down", "j":
		if m.profileCursor < 1 {
			m.profileCursor++
		}
		return m, nil
	case "enter":
		profile := bbrProfile(m.profileCursor)
		return m.profileCB(profile)
	case "1":
		return m.profileCB(bbrProfile(0))
	case "2":
		return m.profileCB(bbrProfile(1))
	}
	return m, nil
}

// handleConfirmKey 确认页：y/enter 是，n/esc 否。
func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		if m.confirmYes != nil {
			return m.confirmYes(m)
		}
	case "n", "N", "esc", "q":
		if m.confirmNo != nil {
			return m.confirmNo(m)
		}
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// handleInputKey 输入页：enter 提交并校验。
func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m.goMenu()
	case "enter":
		val := m.input.Value()
		if m.inputCB != nil {
			return m.inputCB(val)
		}
		return m.goMenu()
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

// handleVersionKey 版本选择：↑↓+Enter。
func (m Model) handleVersionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m.goMenu()
	case "up", "k":
		if m.tagCursor > 0 {
			m.tagCursor--
		}
		return m, nil
	case "down", "j":
		if m.tagCursor < len(m.tags)-1 {
			m.tagCursor++
		}
		return m, nil
	case "pgup":
		m.tagCursor -= 5
		if m.tagCursor < 0 {
			m.tagCursor = 0
		}
		return m, nil
	case "pgdown":
		m.tagCursor += 5
		if m.tagCursor > len(m.tags)-1 {
			m.tagCursor = len(m.tags) - 1
		}
		return m, nil
	case "enter":
		if m.tagCB != nil && len(m.tags) > 0 {
			return m.tagCB(m.tags[m.tagCursor])
		}
	}
	// 数字键直接选择编号
	if n, ok := parseNum(msg.String()); ok && n >= 1 && n <= len(m.tags) {
		m.tagCursor = n - 1
		return m.tagCB(m.tags[m.tagCursor])
	}
	return m, nil
}

// handleResultKey 结果页：enter 返回菜单。
func (m Model) handleResultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "enter", " ":
		return m.goMenu()
	}
	return m, nil
}
