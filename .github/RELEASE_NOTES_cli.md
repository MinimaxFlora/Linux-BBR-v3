## ✨ BBRv3 Go TUI 更新内容

### 🆕 新增：快捷命令 bbr

- 快捷命令由 `b` 更名为 **`bbr`**（安装路径 `/usr/local/bin/bbr`），首次运行程序时自动安装，之后直接输入 `bbr` 即可运行。
- `bbr` **直接执行本地已安装版本**，不再每次联网下载；更新请用 TUI 菜单 10「检测 TUI 更新」，新版本会直接替换本地 `bbr`，下次运行即生效。
- 老版本用户的旧 `b` 命令会在首次运行新程序时自动移除。

### 🌐 新增：国内/国外网络源支持

- 默认**自动模式**：直连 GitHub，失败自动依次尝试国内镜像（`gh-proxy.kejizero.xyz` / `gh-proxy.com` / `ghfast.top`），静默切换。
- TUI 菜单 11「网络源设置」可切换自动 / 仅直连 / 固定镜像，持久化到 `/etc/bbrv3/mirror`；环境变量 `BBRV3_MIRROR` 优先级最高。
- **版本检测改用 version.ini，完全绕开 GitHub API**（镜像无法代理 api.github.com，实测 403/404）：CI 构建时生成 version.ini（含 CLI commit 与最新内核版本），TUI 下载比对；内核安装按版本号直接构造下载 URL。
- 覆盖 install.sh 首次安装、内核 `.deb` 下载、TUI 自更新与版本检测。

### ⚙️ 改进：TUI 内一键自更新

- 检测更新按 **commit SHA** 与 master 最新构建比对（固定 tag 无法比版本号），发现新版本可直接在 TUI 内下载安装，无需手动下载。
- 子页面按 `Esc` 返回主菜单（点错可回退），不再直接退出程序。

### 📌 历史更新

- 2026-08-25：版本检测改用 version.ini（编译时生成，完全绕开 GitHub API），菜单 2 改为直接输入版本号
- 2026-08-25：新增国内/国外网络源支持（自动切换镜像，菜单 11 可配置）
- 2026-08-25：快捷命令 `b` 改名为 `bbr`，自动移除旧 `b`
- 2026-08-25：`b` 快捷命令改为直接执行本地版本，更新走 TUI 菜单 10 自更新
- 2026-08-25：检测更新改用 master 分支最新 commit 作为基准（而非固定 tag 指向的旧 commit）
- 2026-08-25：检测更新修复 commit 比对 + 支持 TUI 内一键更新
- 2026-08-25：主信息卡实时显示 CPU/内存使用率 + 菜单新增 TUI 检测更新
- 2026-08-25：TUI 全面美化（k9s/lazygit 专业框架、渐变 logo、结果横幅、下载进度条）
- 2026-08-25：i18n 中英切换（`L` 键）+ 菜单 4-7 合并为加速模式子菜单
- 2026-08-25：一键安装脚本 install.sh（自动检测架构下载执行）
- 2026-08-25：Go + bubbletea TUI 重写（品牌 MinimaxFlora）

---

**使用方式**

```bash
# 首次安装（自动下载最新版并安装 bbr 快捷命令）
bash <(curl -fsSL https://raw.githubusercontent.com/MinimaxFlora/Linux-BBR-v3/master/install.sh)

# 之后直接运行
bbr
```

内核构建与安装见各架构 release（如 `x86_64-<版本>` / `arm64-<版本>-max`）。
