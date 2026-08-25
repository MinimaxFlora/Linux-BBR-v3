package app

import (
	"context"
	"strings"
	"testing"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/system"

	tea "github.com/charmbracelet/bubbletea"
)

func testModel() Model {
	env := &system.Env{
		HasAPT:       true,
		Arch:         "x86_64",
		ArchFilter:   "x86_64",
		CurrentAlgo:  "bbr",
		CurrentQdisc: "fq",
		OS:           nil,
	}
	SetEnv(env)
	m := *NewModel(context.Background(), env)
	m.page = PageMenu
	return m
}

// press 模拟按键并断言返回 Model 类型。
func press(m Model, msg tea.KeyMsg) Model {
	nm, _ := m.handleKey(msg)
	mm, ok := nm.(Model)
	if !ok {
		panic("handleKey 返回类型异常")
	}
	return mm
}

func TestMenuViewContainsAllItems(t *testing.T) {
	m := testModel()
	v := m.View()
	for _, it := range menuItems {
		if !strings.Contains(v, it.label) {
			t.Errorf("主菜单缺少选项: %s", it.label)
		}
	}
	// 系统状态显示
	if !strings.Contains(v, "bbr") || !strings.Contains(v, "fq") {
		t.Error("主菜单应显示当前 TCP 算法与队列算法")
	}
	// 品牌信息
	if !strings.Contains(v, "MinimaxFlora") {
		t.Error("主菜单应显示 MinimaxFlora 品牌")
	}
}

func TestMenuKeyNavigation(t *testing.T) {
	m := testModel()
	if m.menuCursor != 0 {
		t.Fatalf("初始光标应为 0, got %d", m.menuCursor)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.menuCursor != 1 {
		t.Fatalf("↓ 后光标应为 1, got %d", m.menuCursor)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.menuCursor != 0 {
		t.Fatalf("↑ 后光标应为 0, got %d", m.menuCursor)
	}
}

func TestMenuNumberKeyOpensProfile(t *testing.T) {
	m := testModel()
	// 数字键 1 → 安装最新版 → 先选内核类型
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if m.page != PageProfile {
		t.Fatalf("按 1 应进入内核类型选择页, got page=%d", m.page)
	}
	// 选 Max (2)
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	// 进入任务日志页（安装任务启动）
	if m.page != PageLog {
		t.Fatalf("确认 Max 后应进入任务日志页, got page=%d", m.page)
	}
	if m.installProfile != bbr.ProfileMax {
		t.Errorf("installProfile 应为 max, got %v", m.installProfile)
	}
}

func TestQdiscMenuSetsValues(t *testing.T) {
	m := testModel()
	// 数字键 4 → BBR + FQ
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	if m.page != PageLog {
		t.Fatalf("按 4 应进入任务日志页, got page=%d", m.page)
	}
}

func TestConfirmAndInputViews(t *testing.T) {
	m := testModel()
	m, _ = m.askConfirm("测试确认 (y/n)", nil, nil)
	v := m.View()
	if !strings.Contains(v, "测试确认") {
		t.Error("确认页应显示提示")
	}
	m, _ = m.askInput("请输入测试值: ", nil)
	v = m.View()
	if !strings.Contains(v, "请输入测试值") {
		t.Error("输入页应显示提示")
	}
}

func TestResultViewShowsLog(t *testing.T) {
	m := testModel()
	m.logs = []string{"✔ 某操作成功", "第二行"}
	m, _ = m.showResult("✔ 完成", "按 Enter 返回")
	v := m.View()
	if !strings.Contains(v, "✔ 完成") {
		t.Error("结果页应显示标题")
	}
	if !strings.Contains(v, "✔ 某操作成功") {
		t.Error("结果页应显示日志尾部")
	}
}
