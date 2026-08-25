package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/system"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// usageTickMsg 定时刷新 CPU/内存使用率。
type usageTickMsg struct{}

// Page 页面状态。
type Page int

const (
	PageBoot Page = iota // 启动任务（快捷命令 + 安全缓解）
	PageMenu
	PageQdisc   // 队列算法选择（原菜单 4-7 合并）
	PageProfile // 内核类型选择
	PageConfirm // y/n 确认
	PageInput   // 文本/数字输入
	PageMirror  // 网络源设置（国内/国外）
	PageLog     // 任务执行日志
	PageResult  // 结果页
)

// taskEvent 任务事件：日志行或结束。
type taskEvent struct {
	line string
	err  error
	done bool
}

// uiLogger 把命令输出转发到任务事件通道。
type uiLogger struct {
	ch chan<- taskEvent
}

func (l *uiLogger) Log(line string) {
	select {
	case l.ch <- taskEvent{line: line}:
	default: // 通道满时丢弃，避免阻塞任务
	}
}

func (l *uiLogger) Logf(format string, args ...any) { l.Log(fmt.Sprintf(format, args...)) }

// taskState 任务运行状态（用指针共享，避免值类型 Model 复制导致事件通道丢失）。
type taskState struct {
	evCh       chan taskEvent
	taskCancel context.CancelFunc
}

// Model 根 TUI 模型。
type Model struct {
	ctx    context.Context
	cancel context.CancelFunc
	env    *system.Env
	task   *taskState

	width  int
	height int
	page   Page

	// 菜单
	menuCursor int

	// 任务
	logs      []string
	logTitle  string
	logScroll int
	logAuto   bool
	taskErr   error
	// 任务结束后需要继续交互（如安装后询问重启、版本选择）
	afterTask func(Model) (Model, tea.Cmd)

	// 结果页
	resultTitle string
	resultLines []string
	resultExtra string // 结果页底部的操作提示

	// 确认页
	confirmPrompt string
	confirmYes    func(Model) (Model, tea.Cmd)
	confirmNo     func(Model) (Model, tea.Cmd)

	// 输入页
	input       textinput.Model
	inputPrompt string
	inputErr    string
	inputCB     func(string) (Model, tea.Cmd)

	// 内核类型选择
	profileCursor int
	profileCB     func(bbr.Profile) (Model, tea.Cmd)

	// 队列算法选择
	qdiscCursor int

	// 网络源设置
	mirrorCursor int

	// 组件
	spinner spinner.Model

	// 系统资源（CPU/内存，定时刷新）
	cpuPrev system.CPUStat
	cpuPct  float64
	mem     system.MemStat

	// 下载进度状态（跨事件累积）
	dlLastPct int

	// 智能优化流程状态
	smartRegionCode  string
	smartRegionLabel string
	smartRTTMS       string
	smartUpload      int
	smartDownload    int

	// 安装流程状态
	installProfile bbr.Profile

	// 全局任务取消
	taskCancel context.CancelFunc
}

// NewModel 构造根模型。
func NewModel(ctx context.Context, env *system.Env) *Model {
	cctx, cancel := context.WithCancel(ctx)
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
	ti := textinput.New()
	ti.Prompt = "> "
	ti.CharLimit = 64
	return &Model{
		ctx:     cctx,
		cancel:  cancel,
		env:     env,
		task:    &taskState{},
		page:    PageBoot,
		spinner: sp,
		input:   ti,
		logAuto: true,
	}
}

// Init 启动任务：快捷命令 + 安全缓解，然后进入主菜单。
func (m Model) Init() tea.Cmd {
	m.page = PageBoot
	return tea.Batch(m.spinner.Tick, m.usageTick(), m.startBoot())
}

// usageTick 每 2 秒刷新一次 CPU/内存使用率（仅 Linux 有数据，其他平台保持零值）。
func (m Model) usageTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return usageTickMsg{} })
}

func (m Model) startBoot() tea.Cmd {
	ch := make(chan taskEvent, 512)
	m.task.evCh = ch
	ctx := m.ctx
	go func() {
		logger := &uiLogger{ch: ch}
		if err := system.InstallQuickCommand(ctx, logger); err != nil {
			// 快捷命令失败不阻断（原脚本同）
		}
		if err := system.ApplySecurityMitigations(ctx, logger); err != nil {
			ch <- taskEvent{line: i18n.Tf("sec.applyFail", err)}
		}
		ch <- taskEvent{done: true}
		close(ch)
	}()
	return m.listenTask(ch)
}

func (m Model) listenTask(ch chan taskEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return taskEvent{done: true, err: errors.New("任务通道意外关闭")}
		}
		return ev
	}
}

// Update 分发事件。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.page == PageLog && m.taskRunning() {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "up", "k":
				m.logAuto = false
				if m.logScroll > 0 {
					m.logScroll--
				}
				return m, nil
			case "down", "j":
				if m.logScroll < len(m.logs)-1 {
					m.logScroll++
				}
				return m, nil
			case "pgup":
				m.logAuto = false
				m.logScroll -= 5
				if m.logScroll < 0 {
					m.logScroll = 0
				}
				return m, nil
			case "pgdown":
				m.logScroll += 5
				if m.logScroll > len(m.logs)-1 {
					m.logScroll = len(m.logs) - 1
				}
				return m, nil
			case "end":
				m.logAuto = true
				return m, nil
			}
		}
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case usageTickMsg:
		// 两次采样差计算 CPU 使用率；内存直接读 /proc/meminfo
		if cur, err := system.ReadCPUStat(); err == nil {
			m.cpuPct = cur.Percent(m.cpuPrev)
			m.cpuPrev = cur
		}
		if mem, err := system.ReadMemStat(); err == nil {
			m.mem = mem
		}
		return m, m.usageTick()

	case taskEvent:
		return m.handleTaskEvent(msg)
	}
	return m, nil
}

func (m Model) taskRunning() bool { return m.task != nil && m.task.evCh != nil }

// handleTaskEvent 处理任务日志行/结束。
func (m Model) handleTaskEvent(ev taskEvent) (tea.Model, tea.Cmd) {
	if m.task == nil || m.task.evCh == nil {
		return m, nil
	}
	if !ev.done {
		m.logs = append(m.logs, ev.line)
		if m.logAuto {
			m.logScroll = len(m.logs) // 指向末尾
		}
		return m, m.listenTask(m.task.evCh)
	}
	// done
	m.task.evCh = nil
	if m.task.taskCancel != nil {
		m.task.taskCancel()
		m.task.taskCancel = nil
	}
	if ev.err != nil {
		m.taskErr = ev.err
	} else {
		m.taskErr = nil
	}
	// 启动流程完成 → 主菜单
	if m.page == PageBoot {
		m.page = PageMenu
		m.menuCursor = 0
		return m, nil
	}
	// 任务后有后续交互（重启确认 / 版本选择 / 保存确认等）
	if m.afterTask != nil {
		after := m.afterTask
		m.afterTask = nil
		return after(m)
	}
	m.page = PageResult
	return m, nil
}

// startTask 切换到日志页并启动后台任务。
func (m Model) startTask(title string, fn func(ctx context.Context, log execx.Logger) error) (Model, tea.Cmd) {
	m.page = PageLog
	m.logTitle = title
	m.logs = nil
	m.logScroll = 0
	m.logAuto = true
	m.taskErr = nil
	m.dlLastPct = 0

	tctx, cancel := context.WithCancel(m.ctx)
	m.task.taskCancel = cancel

	ch := make(chan taskEvent, 1024)
	m.task.evCh = ch
	go func() {
		logger := &uiLogger{ch: ch}
		err := fn(tctx, logger)
		ch <- taskEvent{done: true, err: err}
		close(ch)
	}()
	return m, m.listenTask(ch)
}

// goMenu 返回主菜单。
func (m Model) goMenu() (Model, tea.Cmd) {
	m.page = PageMenu
	m.menuCursor = 0
	return m, nil
}

// showResult 展示结果页：显示任务日志尾部 + 附加提示。
func (m Model) showResult(title string, extra string) (Model, tea.Cmd) {
	m.page = PageResult
	m.resultTitle = title
	m.resultExtra = extra
	return m, nil
}

// askConfirm 打开确认页。
func (m Model) askConfirm(prompt string, yes, no func(Model) (Model, tea.Cmd)) (Model, tea.Cmd) {
	m.page = PageConfirm
	m.confirmPrompt = prompt
	m.confirmYes = yes
	m.confirmNo = no
	return m, nil
}

// askInput 打开输入页。
func (m Model) askInput(prompt string, cb func(string) (Model, tea.Cmd)) (Model, tea.Cmd) {
	m.page = PageInput
	m.inputPrompt = prompt
	m.inputErr = ""
	m.inputCB = cb
	m.input.SetValue("")
	m.input.Focus()
	return m, textinput.Blink
}

// askProfile 打开内核类型选择页。
func (m Model) askProfile(cb func(bbr.Profile) (Model, tea.Cmd)) (Model, tea.Cmd) {
	m.page = PageProfile
	m.profileCursor = 0
	m.profileCB = cb
	return m, nil
}

// trimLogs 限制日志行数防止无限增长。
func (m Model) trimLogs() {
	if len(m.logs) > 5000 {
		m.logs = m.logs[len(m.logs)-4000:]
		m.logScroll = len(m.logs)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MenuLabel 返回主菜单项标签（供测试/日志）。
func MenuLabel(i int) string {
	if i >= 0 && i < len(menuItems) {
		return i18n.T(menuItems[i].labelKey)
	}
	return ""
}

var _ = strings.TrimSpace
