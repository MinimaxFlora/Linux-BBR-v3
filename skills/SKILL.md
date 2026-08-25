---
name: linux-bbr-v3
description: Use when working on Linux-BBR-v3 (Go TUI for BBRv3 kernel).
version: 0.1.0
author: MinimaxFlora, Hermes Agent
license: MIT
platforms: [linux, macos, windows]
metadata:
  hermes:
    tags: [bbrv3, linux-kernel, bubbletea, tui, golang]
    related_skills: [bubbletea-tui-dev, github-actions-workflows, linux-kernel-custom-build]
---

# Linux-BBR-v3 项目速览

## When to Use

- 需要在 [MinimaxFlora/Linux-BBR-v3](https://github.com/MinimaxFlora/Linux-BBR-v3)（本机仓库 `D:\text\Linux-BBR-v3`，默认分支 master）上做任何开发、修改、构建、发布工作。
- 需要快速了解项目架构、TUI 菜单、快捷命令机制、CI 发布流程或开发约定。
- TUI 编码细节（bubbletea 布局/动画/i18n/异步任务）另见 `bubbletea-tui-dev` skill。

## 项目定位

- [byJoey/Actions-bbr-v3](https://github.com/byJoey/Actions-bbr-v3) 的 **Go + bubbletea TUI 重写版**：Debian/Ubuntu VPS 的 BBRv3 内核安装与网络加速管理程序。
- 品牌 `joeyblog` → `MinimaxFlora`（内核包名、配置路径、卸载匹配、MODULE_DESCRIPTION）。
- 内核主线 Linux 7.x，BBRv3 补丁固定（`patches/bbrv3-linux-7.x.patch`），内核自动跟随 kernel.org 最新 stable。

## 技术栈与结构

Go 1.26 + charmbracelet/bubbletea + lipgloss。`internal/` 各包职责：

| 包 | 职责 |
| --- | --- |
| `bbr` | 纯逻辑：版本比较 / tag 解析 / buffer 计算（可单测） |
| `execx` | 命令执行器：流式输出转发到 TUI 日志 |
| `system` | sysctl / qdisc / 安全缓解 / 网络调优 / 内核安装 / 快捷命令 |
| `netutil` | GitHub release API + Ookla speedtest |
| `app` | bubbletea TUI：主菜单 + 子页面（表单/进度日志/结果） |
| `i18n` | 中英切换（`L` 键）；`BBRV3_LANG` 环境变量 > `/etc/bbrv3/lang` |

## 关键行为

### TUI 菜单（10 项，menu.go）

1. 安装或更新 BBR v3（最新版，选标准/Max）
2. 指定版本安装
3. 检查 BBR v3 状态
4. 启用 BBR 加速模式（子菜单：FQ / FQ_CODEL / FQ_PIE / CAKE）
5. 亚太机器 TCP 调优
6. 卸载 BBR 内核
7. BBR v3 智能带宽优化
8. 清空网络优化配置
9. BBR v3 疯批模式（极限测速挑战）
10. 检测 TUI 更新

按键：数字 1-10 直接执行、`↑/↓`+Enter 选择、`L` 中英切换、`q` 退出；子页面 `Esc` 返回主菜单（不退出）。

### 快捷命令 bbr

- `/usr/local/bin/bbr` 是**二进制本体**（非脚本）：首次运行（install.sh 或直接跑二进制）时把当前程序复制到该路径，之后 `bbr` 直接执行本地版，**不再每次联网下载**。
- 旧 `/usr/local/bin/b` 自动移除（迁移）。
- **自更新（菜单 10）替换安装路径** `/usr/local/bin/bbr`（`os.Stat(QuickCommandPath)` 存在则替换它，否则退回 `os.Executable()`）；复制/替换用「写 `.new` + rename」原子操作。

### 自更新检测

- 固定 tag `bbrv3-cli`（每次 push 覆盖上传），`Version` 硬编码 v1.0.0 → **无法比版本号，用 commit SHA**：本地 `Commit`（CI 注入 `GITHUB_SHA::8`）vs master head。
- dev 构建（`Commit="dev"`）跳过比对。

### 启动流程

- 非 root 自动 `sudo -E` 重启动（密码提示在 TUI 前的正常终端）。
- root 后先：装 `bbr` 快捷命令 + 写 Dirty Frag 缓解（`/etc/modprobe.d/99-minimaxflora-security.conf`），然后进菜单。
- `BBRV3_DEV=1` 跳过系统检查（非 Linux 预览 TUI）；`BBRV3_SKIP_QUICK_COMMAND=1` 跳过快捷命令安装。

## CI / 发布

- **build.yml**（内核）：schedule 每日 + 手动触发；tag 格式 `x86_64|arm64-<版本>[-max]`；release body 为 emoji 分节模板（matrix 变量填充）；构建后回写配置基线（回写前 `git pull --rebase`）。
- **release-cli.yml**（CLI）：push master / 手动触发构建 amd64+arm64 二进制，覆盖上传 `bbrv3-cli`；**每次推送 `gh release edit` 刷新说明**（`.github/RELEASE_NOTES_cli.md`）。
- 内核包名全小写 `-minimaxflora-bbrv3[-max]`（Debian 包名规范）。

## 常用命令

```bash
go build ./... && go vet ./... && go test ./...   # 构建+静态检查+测试
BBRV3_DEV=1 go run .                                # 开发预览（跳过系统检查）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bbrv3-linux-amd64 .
actionlint .github/workflows/*.yml                  # 验证 workflow（本机装在 $LOCALAPPDATA/Temp/actionlint.exe）
```

## 开发约定与坑

- **Model 值类型**：`Init()` 里赋值无效；跨方法共享运行时状态必须用指针（如 `task *taskState` 持事件通道）。
- lipgloss v1.1.0 盒模型：`Width(w)` 含内边距；分区线宽度必须 = 实际内容宽，否则折行。
- TUI 风格（用户偏好）：品牌粉 `#ec4899` + 青色辅助、深色底；主界面**信息卡 + 菜单卡**两个大框，左对齐不撑满；无顶栏；信息卡竖排分区；中英切换（`L`）必备；不要过度花哨（ASCII 大 LOGO/箭头轮换动画曾被撤销）。
- 无头验证渲染：`BBRV3_DUMP=1` 测试把 `View()` 写文件，Python 剥 ANSI 检查每行宽度。
- 本机（Windows）无 gh CLI：GitHub 操作经 `git credential fill` 取 token 调 REST API；artifact 下载 302→Azure Blob 必须 `curl -L`（urllib 转发 Authorization 会 401）。
- README 四语：`README.md`（中）/ `README_EN` / `README_JA` / `README_KO`，顶部互链，15 节结构一致；release 说明 emoji 分节（`## ✨` + `### 🆕/♻️/⚙️/📌`）。

## 验证

改动后必须通过：`go build ./... && go vet ./... && go test ./...`；workflow 改动用 actionlint + `python -c "import yaml; yaml.safe_load(open(f))"`；推送后按需用 REST API 读回验证远端（release body / raw 文件）。

## 相关

- `bubbletea-tui-dev` skill：TUI 开发全流程（布局/动画/i18n/异步任务/无头验证）。
- `github-actions-workflows` skill：workflow 排错与 CI 经验。
- `linux-kernel-custom-build` / `bbr3-kernel-maintenance` / `bbrv3-kernel-maintenance`：内核补丁刷新与构建。
