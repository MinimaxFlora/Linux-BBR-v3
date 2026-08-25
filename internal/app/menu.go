package app

import (
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"

	tea "github.com/charmbracelet/bubbletea"
)

// menuEntry 主菜单项（原 12 项合并 4-7 为 1 项，共 10 项）。
type menuEntry struct {
	num      string
	labelKey string
	act      func(Model) (Model, tea.Cmd)
}

var menuItems = []menuEntry{
	{num: "1", labelKey: "menu.item1", act: menuInstallLatest},
	{num: "2", labelKey: "menu.item2", act: menuInstallSpecific},
	{num: "3", labelKey: "menu.item3", act: menuCheckStatus},
	{num: "4", labelKey: "menu.item4", act: menuQdiscSelect},
	{num: "5", labelKey: "menu.item5", act: menuAPACTuning},
	{num: "6", labelKey: "menu.item6", act: menuUninstall},
	{num: "7", labelKey: "menu.item7", act: menuSmartTuning},
	{num: "8", labelKey: "menu.item8", act: menuClearOptimizations},
	{num: "9", labelKey: "menu.item9", act: menuExtremeMode},
	{num: "10", labelKey: "menu.item10", act: menuCheckUpdate},
	{num: "11", labelKey: "menu.item11", act: menuMirrorSettings},
}

// qdiscOptions 加速模式子菜单（原菜单 4-7 合并而来）。
var qdiscOptions = []struct {
	num   string
	labelKey string
	qdisc string
}{
	{"1", "qdisc.item1", "fq"},
	{"2", "qdisc.item2", "fq_codel"},
	{"3", "qdisc.item3", "fq_pie"},
	{"4", "qdisc.item4", "cake"},
}

// handleKey 处理当前页面的按键。
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.page {
	case PageMenu:
		return m.handleMenuKey(msg)
	case PageQdisc:
		return m.handleQdiscKey(msg)
	case PageProfile:
		return m.handleProfileKey(msg)
	case PageConfirm:
		return m.handleConfirmKey(msg)
	case PageInput:
		return m.handleInputKey(msg)
	case PageVersion:
		return m.handleVersionKey(msg)
	case PageMirror:
		return m.handleMirrorKey(msg)
	case PageResult:
		return m.handleResultKey(msg)
	}
	return m, nil
}

// handleMenuKey 主菜单按键：数字直接触发，↑↓+Enter 选择，L 切换语言，q 退出。
func (m Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "l", "L":
		// 中英文切换
		if i18n.IsEn() {
			i18n.Set(i18n.Zh)
		} else {
			i18n.Set(i18n.En)
		}
		i18n.Persist()
		return m, nil
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
	// 数字键 1-9 直接触发
	for _, it := range menuItems {
		if msg.String() == it.num {
			return it.act(m)
		}
	}
	return m, nil
}

// handleQdiscKey 队列算法子菜单按键。
func (m Model) handleQdiscKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m.goMenu()
	case "up", "k":
		if m.qdiscCursor > 0 {
			m.qdiscCursor--
		}
		return m, nil
	case "down", "j":
		if m.qdiscCursor < len(qdiscOptions)-1 {
			m.qdiscCursor++
		}
		return m, nil
	case "enter":
		opt := qdiscOptions[m.qdiscCursor]
		return menuQdisc(opt.qdisc)(m)
	}
	// 数字键 1-4
	for i, opt := range qdiscOptions {
		if msg.String() == opt.num {
			m.qdiscCursor = i
			return menuQdisc(opt.qdisc)(m)
		}
	}
	return m, nil
}

// menuQdiscSelect 打开队列算法子菜单。
func menuQdiscSelect(m Model) (Model, tea.Cmd) {
	m.page = PageQdisc
	m.qdiscCursor = 0
	return m, nil
}

// handleProfileKey 内核类型选择（1 标准 / 2 Max）。
func (m Model) handleProfileKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		// 点错返回上一级（主菜单），不退出程序
		return m.goMenu()
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

// handleResultKey 结果页：esc 返回主菜单，enter/空格 返回主菜单，q 退出。
func (m Model) handleResultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "enter", " ":
		return m.goMenu()
	}
	return m, nil
}

// bbrProfile 索引转 Profile。
func bbrProfile(i int) bbr.Profile {
	if i == 1 {
		return bbr.ProfileMax
	}
	return bbr.ProfileStandard
}
