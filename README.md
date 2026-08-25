# Linux-BBR-v3

一个用于 Debian/Ubuntu VPS 的 BBRv3 内核安装与网络加速管理程序。

本项目是 [byJoey/Actions-bbr-v3](https://github.com/byJoey/Actions-bbr-v3) 的 **Go + [bubbletea](https://github.com/charmbracelet/bubbletea) TUI 重写版**：

- 原 shell 版 `install.sh` 重写为 Go 程序，**功能与选项完全一致**（12 项菜单、安全缓解、`b` 快捷命令）。
- 品牌标识由 `joeyblog` 替换为 `MinimaxFlora`（内核包名、配置路径、卸载匹配、MODULE_DESCRIPTION）。
- 内核 `.deb` 包从本仓库 GitHub Releases 下载。

## 运行

```bash
# 一键安装并运行（自动检测架构，下载最新版二进制后启动）
bash <(curl -fsSL https://raw.githubusercontent.com/MinimaxFlora/Linux-BBR-v3/master/install.sh)
```

首次运行后程序会自动安装 `b` 快捷命令，之后可直接运行：

```bash
b
```

`b` 命令直接执行本地已安装版本，不再联网下载；更新请用 TUI 菜单 10「检测 TUI 更新」。

以 root 运行时会先完成：

1. 安装 `/usr/local/bin/b` 快捷命令（复制当前二进制到本地，之后直接执行）。
2. 写入 Dirty Frag 风险面收敛规则（`/etc/modprobe.d/99-minimaxflora-security.conf`）。

然后进入交互菜单。

## 支持环境

| 项目 | 要求 |
| --- | --- |
| 最低支持系统 | Ubuntu 24.04+ / Debian 12+ |
| 推荐系统 | Ubuntu 24.04+ / Debian 12+ |
| 包管理器 | `apt-get` |
| 架构 | `x86_64` / `aarch64` |
| 引导方式 | 建议使用 GRUB |
| 使用场景 | VPS / 云服务器 / 独立服务器 |

不建议在树莓派、NanoPi 等依赖 U-Boot 或厂商定制内核链路的设备上使用。

Debian testing/unstable 如果缺少 `VERSION_ID`，程序会按 `VERSION_CODENAME` 识别 `bookworm`、`trixie`、`forky` 和 `sid`。Alpine Linux 暂不支持（release 产物为 `.deb`）。

本项目当前内核主线为 Linux 7.x。安装内核时会按最低支持系统拦截过旧环境，避免 kernel panic。旧系统仍可使用状态检查、网络调优、清空优化和卸载功能。

## 菜单功能

```text
 1. 🚀 安装或更新 BBR v3 (最新版)
 2. 📚 指定版本安装
 3. 🔍 检查 BBR v3 状态
 4. ⚡ 启用 BBR + FQ
 5. ⚡ 启用 BBR + FQ_CODEL
 6. ⚡ 启用 BBR + FQ_PIE
 7. ⚡ 启用 BBR + CAKE
 8. 🌏 亚太机器 TCP 调优
 9. 🗑️ 卸载 BBR 内核
10. 🧠 BBR v3 智能带宽优化
11. 🧹 清空网络优化配置
12. 🧨 BBR v3 疯批模式（极限测速挑战）
```

常用流程：

1. 选择 `1` 安装或更新 BBRv3 内核，并按提示选择标准版或 Max 极限版。
2. 按提示重启系统。
3. 重新运行，选择 `3` 检查 BBRv3 状态。
4. 按需选择 `4` 到 `7` 设置队列算法。
5. 亚太线路机器可选择 `8` 写入 TCP 收发窗口与空闲慢启动调优。
6. 不确定线路参数时可选择 `10` 自动测速并按带宽档位计算 TCP 缓冲区。
7. 做自有链路极限测速挑战时可选择 `12` 写入激进冲速率参数。
8. 需要撤回调优时可选择 `11` 清空程序写入的网络优化配置。

## 内核与 BBR 策略

```text
BBRv3 补丁固定，内核自动跟随 kernel.org 最新 stable 更新。
```

当前 patch 选择规则：

```text
linux-7.0.y -> patches/bbrv3-linux-7.0.patch
linux-7.1.y -> patches/bbrv3-linux-7.1.patch
```

内核包由 GitHub Actions 构建并发布到 Releases，包名示例（Debian 包名规范要求全小写，`minimaxflora` 为品牌后缀）：

```text
linux-headers-7.2.0-minimaxflora-bbrv3_7.2.0-1_amd64.deb        （标准版）
linux-headers-7.2.0-minimaxflora-bbrv3-max_7.2.0-1_amd64.deb    （Max 版）
```

release tag 格式：

```text
x86_64-7.1.8
arm64-7.1.8
x86_64-7.1.8-max
arm64-7.1.8-max
```

Max 版会提高 Startup、ProbeBW 和 cwnd 策略的进攻性，但保留 BBRv3 的 loss、ECN、inflight 和 ProbeBW 反馈闭环，只适合自有链路吞吐测试，不建议日常生产使用。

## 检查 BBRv3 状态

选择 `3` 后检查：

- `tcp_bbr` 模块版本是否为 `3`。
- 当前 TCP 拥塞控制算法是否为 `bbr`。
- Dirty Frag 相关模块黑名单是否写入。

## 加速模式

| 菜单 | 拥塞控制 | 队列算法 |
| --- | --- | --- |
| 4 | `bbr` | `fq` |
| 5 | `bbr` | `fq_codel` |
| 6 | `bbr` | `fq_pie` |
| 7 | `bbr` | `cake` |

选择后立即尝试应用配置，并询问是否永久写入 `/etc/sysctl.d/99-minimaxflora.conf`。

程序不仅写入 `net.core.default_qdisc`，还会把当前默认路由出口网卡的 root qdisc 立即替换为所选算法。需要模块加载的队列算法会写入 `/etc/modules-load.d/minimaxflora-qdisc.conf`。

## 亚太机器 TCP 调优

选择 `8` 后立即应用并永久写入：

```text
net.ipv4.tcp_wmem = 4096 16384 12582912
net.ipv4.tcp_rmem = 4096 131072 33554432
net.ipv4.tcp_limit_output_bytes = 4194304
net.ipv4.tcp_slow_start_after_idle = 0
```

## BBR v3 智能带宽优化

选择 `10` 后：

- 优先安装并运行 Ookla 官方 `speedtest 1.2.0`，自动尝试附近测速服务器并获取上传/下载带宽；检测到 Python 版 `speedtest-cli` 会先移除。测速失败时提示手动输入上传带宽。测速节点延迟被隐藏，不参与 RTT 计算。
- 自动启用 `bbr` 拥塞控制和 `fq` 队列算法。
- 根据上传带宽和地区模式映射推荐 TCP buffer 档位，按机器内存设置上限。
- RTT 由用户手动输入（v2rayN 测出的真实链接延迟），手动选择亚太、美欧或手动 RTT + buffer 档位。

## BBR v3 疯批模式

选择 `12` 后强制启用 `bbr` + `fq`，写入：

```text
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr
net.core.rmem_max = 1073741824
net.core.wmem_max = 1073741824
net.core.optmem_max = 1073741824
net.core.netdev_max_backlog = 1000000
net.core.somaxconn = 65535
net.ipv4.tcp_wmem = 4096 1048576 1073741824
net.ipv4.tcp_rmem = 4096 1048576 1073741824
net.ipv4.tcp_limit_output_bytes = 268435456
net.ipv4.tcp_slow_start_after_idle = 0
net.ipv4.tcp_notsent_lowat = 4294967295
net.ipv4.tcp_autocorking = 0
net.ipv4.tcp_no_metrics_save = 1
net.ipv4.tcp_mtu_probing = 1
net.ipv4.tcp_fastopen = 3
net.ipv4.tcp_window_scaling = 1
net.ipv4.tcp_moderate_rcvbuf = 1
net.ipv4.tcp_ecn = 0
```

同时把当前默认出口网卡的运行态 `txqueuelen` 拉高到 `100000`（不写入持久配置）。核心参数应用失败会中止；附加参数失败不阻断。

## 清空网络优化配置

选择 `11` 后清理本程序写入的持久配置（default_qdisc、拥塞控制、TCP buffer 全系列），并删除 `/etc/modules-load.d/minimaxflora-qdisc.conf`。只清空网络优化配置，不卸载 BBR 内核，也不移除 Dirty Frag 缓解规则。运行态参数可能需要重启后完全恢复默认。

## 安全缓解

启动时写入 `/etc/modprobe.d/99-minimaxflora-security.conf`：

- `esp4` / `esp6` / `rxrpc` 黑名单（`blacklist` + `install /bin/false`），用于收敛 Dirty Frag 风险面。已加载模块会尝试卸载；被占用时重启后生效。

CVE-2026-31431 对应的 AEAD userspace 接口在新构建内核中由内核配置侧收敛（`# CONFIG_CRYPTO_USER_API_AEAD is not set`），不再额外写入 `algif_aead` 黑名单；若旧版本已写入，会在当前运行内核确认关闭后自动移除。

## 卸载

选择 `9` 卸载由本项目安装的 `MinimaxFlora` 内核包并更新引导配置。卸载后建议重启。

## 构建

本地构建（Windows / macOS / Linux 均可交叉编译）：

```bash
# Linux amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bbrv3-linux-amd64 .
# Linux arm64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bbrv3-linux-arm64 .
```

开发预览（跳过系统检查，便于在非 Linux 机器查看 TUI）：

```bash
BBRV3_DEV=1 go run .
```

测试：

```bash
go test ./...
```

GitHub Actions 的 `release-cli.yml` workflow（与内核构建独立）会在推送 master 或手动触发时构建并发布二进制到 `bbrv3-cli` release，TUI 菜单 10「检测 TUI 更新」从中拉取最新版本替换本地 `b`。`build.yml` 只负责内核构建与发布。

## 目录结构

```text
install.sh                      # 一键脚本：自动检测架构 → 下载最新二进制 → 执行
main.go                         # 入口（非 root 自动 sudo 重启动）
internal/
├── bbr/                       # 纯逻辑：版本比较 / tag 解析 / buffer 计算（可单测）
├── execx/                     # 命令执行器：流式输出转发到 TUI 日志
├── system/                    # sysctl / qdisc / 安全缓解 / 网络调优 / 内核安装 / 快捷命令
├── netutil/                   # GitHub release API + Ookla speedtest
└── app/                       # bubbletea TUI：主菜单 + 子页面（表单/进度日志/结果）
scripts/                       # 内核构建脚本（CI 用，与原项目一致，包名已品牌化）
patches/                       # BBRv3 补丁（固定）
x86-64.config / arm64.config   # 内核配置基线
.github/workflows/build.yml      # 内核自动构建 + 发布（schedule / 手动触发）
.github/workflows/release-cli.yml # Go TUI 二进制发布到 bbrv3-cli release（push master / 手动触发）
```

## 免责声明

内核升级有风险。安装前建议确认 VPS 控制台、救援模式或旧内核启动项可用。使用本项目构建或安装的内核造成的系统启动失败、网络异常或数据损失，由使用者自行承担。
