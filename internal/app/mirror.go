package app

import (
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/netutil"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------- 菜单 11：网络源设置（国内/国外） ----------

func menuMirrorSettings(m Model) (Model, tea.Cmd) {
	m.page = PageMirror
	m.mirrorCursor = 0
	return m, nil
}

// handleMirrorKey 网络源设置页：↑↓+Enter 或数字键选择，Esc 返回。
func (m Model) handleMirrorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	opts := netutil.MirrorValues()
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m.goMenu()
	case "up", "k":
		if m.mirrorCursor > 0 {
			m.mirrorCursor--
		}
		return m, nil
	case "down", "j":
		if m.mirrorCursor < len(opts)-1 {
			m.mirrorCursor++
		}
		return m, nil
	case "enter":
		return m.applyMirror(opts[m.mirrorCursor])
	}
	if n, ok := parseNum(msg.String()); ok && n >= 1 && n <= len(opts) {
		m.mirrorCursor = n - 1
		return m.applyMirror(opts[m.mirrorCursor])
	}
	return m, nil
}

// applyMirror 持久化网络源并展示结果。
func (m Model) applyMirror(v string) (Model, tea.Cmd) {
	if err := netutil.SetMirror(v); err != nil {
		return m.showResult(i18n.T("mirror.failTitle"), err.Error())
	}
	return m.showResult(i18n.T("mirror.savedTitle"), i18n.Tf("mirror.savedMsg", mirrorLabel(v)))
}

// mirrorLabel 网络源值 → 显示名（auto/direct 用 i18n，URL 原样）。
func mirrorLabel(v string) string {
	switch v {
	case netutil.MirrorAuto:
		return i18n.T("mirror.auto")
	case netutil.MirrorDirect:
		return i18n.T("mirror.direct")
	default:
		return v
	}
}

// mirrorStatusLabel 信息卡状态行的网络源短标签（auto/direct 短文案，URL 只显示主机名）。
func mirrorStatusLabel() string {
	switch v := netutil.CurrentMirror(); v {
	case netutil.MirrorAuto:
		return green(i18n.T("mirror.shortAuto"))
	case netutil.MirrorDirect:
		return cyan(i18n.T("mirror.shortDirect"))
	default:
		return cyan(mirrorHost(v))
	}
}

// mirrorHost 提取镜像 URL 的主机名（去协议与尾斜杠）。
func mirrorHost(u string) string {
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimSuffix(u, "/")
	return u
}
