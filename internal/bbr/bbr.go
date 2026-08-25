// Package bbr 包含从原 install.sh 移植的纯逻辑函数（不依赖系统命令，可单元测试）。
package bbr

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Profile 内核类型：standard（标准版）或 max（激进吞吐版）。
type Profile string

const (
	ProfileStandard Profile = "standard"
	ProfileMax      Profile = "max"
)

// 品牌标识：内核 LOCALVERSION、配置文件路径、卸载匹配的统一前缀。
// 原项目为 joeyblog，本仓库替换为 MinimaxFlora。
const (
	// Brand 显示名（界面/日志/MODULE_DESCRIPTION 使用，保留大小写）。
	Brand = "MinimaxFlora"
	// KernelBrand 内核包名品牌后缀。Debian 包名必须全小写
	// （linux-image-7.2.0-minimaxflora-bbrv3），不能用显示名的大写。
	KernelBrand = "minimaxflora"
	// KernelTag 内核 localversion 后缀。
	KernelTag = "bbrv3"

	// 版本字符串后缀，对应内核包名 linux-headers-7.2.0-minimaxflora-bbrv3-max_7.2.0-1_amd64.deb
	StandardVersionSuffix = "-" + KernelBrand + "-" + KernelTag
	MaxVersionSuffix      = "-" + KernelBrand + "-" + KernelTag + "-max"

	// 持久化配置文件路径（原 99-joeyblog.conf / joeyblog-qdisc.conf / 99-joeyblog-security.conf）
	SysctlConfPath          = "/etc/sysctl.d/99-minimaxflora.conf"
	ModulesLoadConfPath     = "/etc/modules-load.d/minimaxflora-qdisc.conf"
	SecurityModprobeConfPath = "/etc/modprobe.d/99-minimaxflora-security.conf"

	// 仓库与发布信息
	RepoOwner = "MinimaxFlora"
	RepoName  = "Linux-BBR-v3"

	// 快捷命令路径与远程拉取入口
	QuickCommandPath = "/usr/local/bin/b"
	// Go 二进制发布资产名（不带架构后缀，下载时按架构拼接）
	BinaryAssetBase = "bbrv3"
)

// RepoFullName 返回 owner/repo。
func RepoFullName() string { return RepoOwner + "/" + RepoName }

// VersionGE 比较两个版本号（语义同 sort -V）：current >= required。
func VersionGE(current, required string) bool {
	cm := versionParts(current)
	rm := versionParts(required)
	for i := 0; i < len(cm) || i < len(rm); i++ {
		var c, r int64
		if i < len(cm) {
			c = cm[i]
		}
		if i < len(rm) {
			r = rm[i]
		}
		if c != r {
			return c > r
		}
	}
	return true
}

var nonDigitRe = regexp.MustCompile(`[^0-9]+`)

// versionParts 把 "24.04" / "12" / "999" 拆成整数段。
func versionParts(v string) []int64 {
	parts := nonDigitRe.Split(strings.TrimSpace(v), -1)
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return []int64{0}
	}
	return out
}

// DebianVersionFromCodename 将 Debian 版本代号映射为版本号。
// bookworm→12, trixie→13, forky→14, sid/unstable→999；未知返回 ok=false。
func DebianVersionFromCodename(codename string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(codename)) {
	case "bookworm":
		return "12", true
	case "trixie":
		return "13", true
	case "forky":
		return "14", true
	case "sid", "unstable":
		return "999", true
	default:
		return "", false
	}
}

// IsPositiveNumber 判断字符串是否为严格正数（数字 > 0）。
func IsPositiveNumber(v string) bool {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	return err == nil && f > 0
}

// BufferCapMB 按内存容量（MemTotal KB）限制智能优化 buffer 上限，单位 MB。
// 与原脚本 get_tcp_buffer_cap_mb 一致：<512MB→16, <1GB→32, 否则 64；无法识别→64。
func BufferCapMB(memTotalKB int64) int {
	switch {
	case memTotalKB <= 0:
		return 64
	case memTotalKB < 524288:
		return 16
	case memTotalKB < 1048576:
		return 32
	default:
		return 64
	}
}

// CalcSmartBufferMB 按带宽(Mbit/s)与地区模式映射推荐 buffer，单位 MB，
// 结果不超过 capMB。与原脚本 calculate_smart_buffer_mb 一致。
func CalcSmartBufferMB(bandwidth float64, overseas bool, capMB int) int {
	bw := int(bandwidth)
	if bw <= 0 {
		bw = 1000
	}
	bufferMB := 16
	if overseas {
		switch {
		case bw < 500:
			bufferMB = 16
		case bw < 1000:
			bufferMB = 48
		default:
			bufferMB = 64
		}
	} else {
		switch {
		case bw < 500:
			bufferMB = 8
		case bw < 1000:
			bufferMB = 12
		case bw < 2000:
			bufferMB = 16
		case bw < 5000:
			bufferMB = 24
		case bw < 10000:
			bufferMB = 28
		default:
			bufferMB = 32
		}
	}
	if bufferMB > capMB {
		bufferMB = capMB
	}
	return bufferMB
}

// ArchFilter 将 uname -m 输出映射为 release tag 使用的架构名。
// aarch64→arm64, x86_64→x86_64；其他返回 ok=false。
func ArchFilter(unameArch string) (string, bool) {
	switch unameArch {
	case "aarch64":
		return "arm64", true
	case "x86_64":
		return "x86_64", true
	default:
		return "", false
	}
}

// ProfileLabel 返回内核类型的中文说明。
func ProfileLabel(p Profile) string {
	if p == ProfileMax {
		return "BBR v3 Max（激进吞吐内核）"
	}
	return "BBR v3 标准版"
}

// ExpectedInstalledVersion 由 release tag 与 profile 推导期望的已安装内核版本字符串。
// 例：tag "x86_64-7.2.0-max" + max → "7.2.0-minimaxflora-bbrv3-max"。
func ExpectedInstalledVersion(tag string, p Profile) string {
	v := strings.TrimPrefix(tag, "x86_64-")
	v = strings.TrimPrefix(v, "arm64-")
	v = strings.TrimSuffix(v, "-max")
	if p == ProfileMax {
		return v + MaxVersionSuffix
	}
	return v + StandardVersionSuffix
}

// VersionFromTag 提取 tag 中的内核版本号。
func VersionFromTag(tag string) string {
	v := strings.TrimPrefix(tag, "x86_64-")
	v = strings.TrimPrefix(v, "arm64-")
	return strings.TrimSuffix(v, "-max")
}

var debugAssetRe = regexp.MustCompile(`(?i)(-dbg_|-dbgsym_)`)

// IsDebugAsset 判断资产 URL 是否为 debug 包（原脚本拒绝发布/安装 *-dbg*.deb / *-dbgsym*.deb）。
func IsDebugAsset(url string) bool { return debugAssetRe.MatchString(url) }

// TagMatchesProfile 判断 tag 是否匹配架构与内核类型。
// 规则与原 jq 一致：tag 以 "<arch>-<数字>" 开头；max 要求以 -max 结尾，standard 要求不以 -max 结尾。
func TagMatchesProfile(tag, arch string, p Profile) bool {
	prefix := arch + "-"
	if !strings.HasPrefix(tag, prefix) || len(tag) <= len(prefix) {
		return false
	}
	rest := tag[len(prefix):]
	if rest == "" || !isDigit(rune(rest[0])) {
		return false
	}
	if p == ProfileMax {
		return strings.HasSuffix(tag, "-max")
	}
	return !strings.HasSuffix(tag, "-max")
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

// SortTagsByVersion 将 release tag 列表按内核版本升序排序（稳定）。
func SortTagsByVersion(tags []string) {
	sort.SliceStable(tags, func(i, j int) bool {
		return !VersionGE(VersionFromTag(tags[i]), VersionFromTag(tags[j]))
	})
}
