// Package system 封装系统级操作：环境检查、sysctl、qdisc、安全缓解、
// 网络调优、内核安装与管理、快捷命令。所有命令以 root 权限运行
// （main 启动时已处理非 root 自动 sudo 重启动）。
package system

import (
	"bufio"
	"errors"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"
)

// Env 描述当前系统环境。
type Env struct {
	// HasAPT 是否存在 apt-get（Debian/Ubuntu 判定）。
	HasAPT bool
	// Arch uname -m 原始输出（aarch64 / x86_64）。
	Arch string
	// ArchFilter release tag 使用的架构名（arm64 / x86_64）。
	ArchFilter string
	// CurrentAlgo 当前 TCP 拥塞控制算法。
	CurrentAlgo string
	// CurrentQdisc 当前默认队列算法。
	CurrentQdisc string
	// OS 解析自 /etc/os-release。
	OS *OSRelease
}

// OSRelease 解析后的 /etc/os-release 关键字段。
type OSRelease struct {
	ID         string
	VersionID  string
	Codename   string
	PrettyName string
}

// LoadEnv 收集运行环境信息（不修改系统）。
func LoadEnv(ctx context.Context) *Env {
	e := &Env{}
	e.HasAPT = execx.RunOK(ctx, "apt-get", "--version")
	e.Arch = execx.TryOutput(ctx, "uname", "-m")
	if f, ok := bbr.ArchFilter(strings.TrimSpace(e.Arch)); ok {
		e.ArchFilter = f
	}
	e.CurrentAlgo = strings.TrimSpace(execx.TryOutput(ctx, "sysctl", "-n", "net.ipv4.tcp_congestion_control"))
	e.CurrentQdisc = strings.TrimSpace(execx.TryOutput(ctx, "sysctl", "-n", "net.core.default_qdisc"))
	e.OS = ReadOSRelease()
	return e
}

// ReadOSRelease 读取并解析 /etc/os-release。
func ReadOSRelease() *OSRelease {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return nil
	}
	defer f.Close()

	o := &OSRelease{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v = strings.Trim(v, `"'`)
		switch k {
		case "ID":
			o.ID = v
		case "VERSION_ID":
			o.VersionID = v
		case "VERSION_CODENAME":
			o.Codename = v
		case "UBUNTU_CODENAME":
			if o.Codename == "" {
				o.Codename = v
			}
		case "PRETTY_NAME":
			o.PrettyName = v
		case "NAME":
			if o.PrettyName == "" {
				o.PrettyName = v
			}
		}
	}
	return o
}

// RequireAPT 检查是否 Debian/Ubuntu 系（apt-get 可用），不可用返回错误。
func RequireAPT(ctx context.Context) error {
	if execx.RunOK(ctx, "apt-get", "--version") {
		return nil
	}
	return errors.New(i18n.T("sys.aptOnly"))
}

// depPackage 命令名 → Debian/Ubuntu 软件包名映射。
// 注意：awk 是虚拟包（由 gawk/mawk 提供），sysctl 属于 procps，不能直接用命令名当包名。
var depPackage = map[string]string{
	"curl":   "curl",
	"wget":   "wget",
	"dpkg":   "dpkg",
	"awk":    "gawk",
	"sed":    "sed",
	"sysctl": "procps",
	"jq":     "jq",
}

// EnsureDeps 检查必需命令，缺失则通过 apt 安装。
// 与原脚本一致：安装失败仅警告，不阻断程序运行。
func EnsureDeps(ctx context.Context, log execx.Logger) error {
	required := []string{"curl", "wget", "dpkg", "awk", "sed", "sysctl", "jq"}
	var pkgs []string
	seen := map[string]bool{}
	for _, cmd := range required {
		if execx.HasCommand(cmd) {
			continue
		}
		if pkg := depPackage[cmd]; pkg != "" && !seen[pkg] {
			seen[pkg] = true
			pkgs = append(pkgs, pkg)
		}
	}
	if len(pkgs) == 0 {
		return nil
	}
	log.Logf(i18n.Tf("sys.deps", strings.Join(pkgs, " ")))
	_ = execx.RunOK(ctx, "apt-get", "update")
	args := append([]string{"install", "-y"}, pkgs...)
	if _, err := execx.Run(ctx, log, "apt-get", args...); err != nil {
		log.Logf(i18n.Tf("sys.depsWarn", err))
	}
	return nil
}

// CheckArch 校验架构，返回 release 架构名。
func CheckArch(e *Env) error {
	if e.ArchFilter == "" {
		return fmt.Errorf("%s", i18n.Tf("sys.archOnly", e.Arch))
	}
	return nil
}

// AssertSupportedKernelInstallSystem 限制旧系统安装 7.x 主线内核。
// 原脚本 assert_supported_kernel_install_system：最低支持 Ubuntu 24.04+ / Debian 12+。
func AssertSupportedKernelInstallSystem(ctx context.Context, o *OSRelease) error {
	if o == nil || o.ID == "" {
		return errors.New(i18n.T("sys.osUnknown"))
	}

	var minVersion, distroName string
	switch o.ID {
	case "ubuntu":
		minVersion, distroName = "24.04", "Ubuntu"
	case "debian":
		minVersion, distroName = "12", "Debian"
		if o.VersionID == "" {
			if v, ok := bbr.DebianVersionFromCodename(o.Codename); ok {
				o.VersionID = v
			}
		}
	default:
		name := o.PrettyName
		if name == "" {
			name = "未知系统"
		}
		return fmt.Errorf("%s", i18n.Tf("sys.osNotAllow", name))
	}

	if o.VersionID == "" || !bbr.VersionGE(o.VersionID, minVersion) {
		name := o.PrettyName
		if name == "" {
			name = o.ID
		}
		return fmt.Errorf("%s", i18n.Tf("sys.osTooOld", name, distroName, minVersion))
	}
	return nil
}

// IsRoot 判断当前是否 root（Windows 开发环境下视为 root 以便本地调试 TUI）。
func IsRoot() bool {
	if runtime.GOOS == "windows" {
		return true
	}
	return os.Geteuid() == 0
}

// KernelRelease 返回当前内核版本（uname -r），失败返回空。
func KernelRelease() string {
	return strings.TrimSpace(execx.TryOutput(context.Background(), "uname", "-r"))
}
