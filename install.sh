#!/usr/bin/env bash
#
# Linux-BBR-v3 一键安装脚本
# 自动检测架构 → 从 bbrv3-cli release 下载最新 Go TUI 二进制 → 执行
#
# 用法：
#   bash <(curl -fsSL https://github.com/MinimaxFlora/Linux-BBR-v3/raw/refs/heads/master/install.sh)
#   或（sh/dash 亦可，脚本会自动切换到 bash 执行）：
#   sh -c "$(curl -fsSL https://github.com/MinimaxFlora/Linux-BBR-v3/raw/refs/heads/master/install.sh)"
#
# 注：使用 github.com 的 raw/refs/heads/master 路径（页面域缓存即时）；
#     raw.githubusercontent.com 的 CDN 缓存可能滞后，获取到旧版脚本。
# 首次运行后程序会自动安装 `bbr` 快捷命令，之后可直接输入 bbr 运行。

# 本脚本使用 bash 特性（set -o pipefail、数组等）。
# 若被 sh/dash 执行（如 `sh -c "$(curl ...)"`，此时 shebang 不生效），
# 自动检测并切换到 bash 重新执行，避免 "set: Illegal option -o pipefail"。
if [ -z "${BASH_VERSION:-}" ]; then
    if command -v bash >/dev/null 2>&1; then
        if [ -f "$0" ] && [ -r "$0" ]; then
            exec bash "$0" "$@"
        fi
        exec bash -c "$(curl -fsSL "https://github.com/MinimaxFlora/Linux-BBR-v3/raw/refs/heads/master/install.sh")"
    fi
    echo "检测到当前未使用 bash，且系统未安装 bash。请改用：bash <(curl -fsSL https://github.com/MinimaxFlora/Linux-BBR-v3/raw/refs/heads/master/install.sh)" >&2
    exit 1
fi

set -euo pipefail

# 仓库与 CLI release 信息
REPO="MinimaxFlora/Linux-BBR-v3"
CLI_TAG="bbrv3-cli"

# 自动检测架构
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)
        ASSET="bbrv3-linux-amd64"
        ;;
    aarch64)
        ASSET="bbrv3-linux-arm64"
        ;;
    *)
        echo -e "\033[31m不支持的架构：$ARCH（仅支持 x86_64 / aarch64）\033[0m" >&2
        exit 1
        ;;
esac

# 下载最新版二进制到临时目录（脚本退出时自动清理）
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT
BIN="$TMPDIR/bbrv3"
URL="https://github.com/$REPO/releases/download/$CLI_TAG/$ASSET"

# 下载源：BBRV3_MIRROR 环境变量可指定（direct=仅直连 / auto=自动 / URL=固定镜像），
# 默认自动：直连 GitHub 失败后依次尝试国内镜像（ghproxy 风格：<镜像>/<完整 URL>）。
MIRRORS=()
case "${BBRV3_MIRROR:-auto}" in
    direct)
        MIRRORS=("")
        ;;
    auto|"")
        MIRRORS=("" "https://gh-proxy.kejizero.xyz/" "https://gh-proxy.com/" "https://ghfast.top/")
        ;;
    *)
        MIRRORS=("$BBRV3_MIRROR")
        ;;
esac

echo -e "\033[36m正在下载 BBRv3 管理程序（${ARCH}）...\033[0m"
downloaded=0
for m in "${MIRRORS[@]}"; do
    if curl -fsSL -o "$BIN" "${m}${URL}"; then
        downloaded=1
        break
    fi
done
if [ "$downloaded" -ne 1 ]; then
    echo -e "\033[31m下载失败：$URL\033[0m" >&2
    echo -e "\033[33m请检查网络连接，或确认 $CLI_TAG release 已发布。\033[0m" >&2
    echo -e "\033[33m国内环境可设置 BBRV3_MIRROR=https://ghfast.top/ 指定镜像后重试。\033[0m" >&2
    exit 1
fi

# 基本校验：文件非空
if [[ ! -s "$BIN" ]]; then
    echo -e "\033[31m下载文件为空，可能 $CLI_TAG release 尚未发布，请稍后重试。\033[0m" >&2
    exit 1
fi

chmod +x "$BIN"
echo -e "\033[1;32m✔ 下载完成，正在启动...\033[0m"

# 启动（非 root 时程序内部会自动通过 sudo 重新执行）
"$BIN"
