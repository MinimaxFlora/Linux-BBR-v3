package system

import (
	"context"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"
)

// LoadQdiscModule 检查并加载队列算法模块（对应原脚本 load_qdisc_module）。
// 返回 (可用, 错误消息)；可用表示当前内核支持该 qdisc。
func LoadQdiscModule(ctx context.Context, log execx.Logger, qdiscName string) (bool, string) {
	moduleName := "sch_" + strings.ReplaceAll(qdiscName, "-", "_")
	previousQdisc := SysctlGet(ctx, "net.core.default_qdisc")

	// 已加载则跳过 modprobe
	if !moduleLoaded(ctx, moduleName) {
		_ = execx.RunOK(ctx, "modprobe", moduleName)
	}

	// 试设 default_qdisc 验证可用性，然后恢复原值
	if execx.RunOK(ctx, "sysctl", "-w", "net.core.default_qdisc="+qdiscName) {
		applied := SysctlGet(ctx, "net.core.default_qdisc")
		if previousQdisc != "" {
			_ = execx.RunOK(ctx, "sysctl", "-w", "net.core.default_qdisc="+previousQdisc)
		}
		if applied == qdiscName {
			return true, ""
		}
	}

	log.Logf(i18n.Tf("qdisc.loading", moduleName))
	if execx.RunOK(ctx, "modprobe", moduleName) {
		if execx.RunOK(ctx, "sysctl", "-w", "net.core.default_qdisc="+qdiscName) {
			applied := SysctlGet(ctx, "net.core.default_qdisc")
			if previousQdisc != "" {
				_ = execx.RunOK(ctx, "sysctl", "-w", "net.core.default_qdisc="+previousQdisc)
			}
			if applied == qdiscName {
				log.Logf(i18n.Tf("qdisc.available", qdiscName))
				return true, ""
			}
		}
	}
	return false, i18n.Tf("qdisc.unavail", qdiscName, moduleName)
}

// moduleLoaded 检查模块是否已加载（lsmod 精确匹配）。
func moduleLoaded(ctx context.Context, moduleName string) bool {
	out := execx.TryOutput(ctx, "lsmod")
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == moduleName {
			return true
		}
	}
	return false
}

// EnsureIPRoute2 确保 ip/tc 可用，缺失则安装 iproute2。
func EnsureIPRoute2(ctx context.Context, log execx.Logger) bool {
	if execx.HasCommand("ip") && execx.HasCommand("tc") {
		return true
	}
	log.Logf(i18n.T("qdisc.iproute2"))
	_ = execx.RunOK(ctx, "apt-get", "update")
	if execx.RunOK(ctx, "apt-get", "install", "-y", "iproute2") {
		return true
	}
	log.Logf(i18n.T("qdisc.iproute2F"))
	return false
}

// DefaultRouteInterfaces 获取默认路由出口网卡（IPv4 + IPv6，去重）。
func DefaultRouteInterfaces(ctx context.Context) []string {
	out := execx.TryOutput(ctx, "ip", "-o", "route", "show", "default")
	out += "\n" + execx.TryOutput(ctx, "ip", "-o", "-6", "route", "show", "default")
	seen := map[string]bool{}
	var ifaces []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "dev" && i+1 < len(fields) {
				if !seen[fields[i+1]] {
					seen[fields[i+1]] = true
					ifaces = append(ifaces, fields[i+1])
				}
			}
		}
	}
	return ifaces
}

// ApplyQdiscToActiveInterfaces 让当前默认出口网卡立即切换队列算法（tc qdisc replace）。
// 返回 (是否全部/部分成功, 是否失败)。
func ApplyQdiscToActiveInterfaces(ctx context.Context, log execx.Logger, qdiscName string) (applied bool, failed bool) {
	if !EnsureIPRoute2(ctx, log) {
		return false, false
	}
	ifaces := DefaultRouteInterfaces(ctx)
	if len(ifaces) == 0 {
		log.Logf(i18n.T("qdisc.noIface"))
		return false, false
	}
	for _, iface := range ifaces {
		if execx.RunOK(ctx, "tc", "qdisc", "replace", "dev", iface, "root", qdiscName) {
			log.Logf(i18n.Tf("qdisc.ifaceOk", iface, qdiscName))
			applied = true
		} else {
			log.Logf(i18n.Tf("qdisc.ifaceFail", iface, qdiscName))
			failed = true
		}
	}
	return applied, failed
}

// PersistQdiscModule 根据队列算法决定是否写入开机模块加载配置（原 persist_qdisc_module）。
// fq 为内置队列：删除 MODULES_CONF；其他模块存在则写入。
func PersistQdiscModule(ctx context.Context, log execx.Logger, qdiscName string) {
	moduleName := "sch_" + strings.ReplaceAll(qdiscName, "-", "_")
	if qdiscName == "fq" {
		_ = removeFile(ctx, bbr.ModulesLoadConfPath)
		log.Logf(i18n.T("qdisc.persist1"))
		return
	}
	modinfoOK := execx.RunOK(ctx, "modinfo", moduleName)
	if modinfoOK || moduleLoaded(ctx, moduleName) {
		_ = writeFile(ctx, bbr.ModulesLoadConfPath, moduleName+"\n", 0o644)
		log.Logf(i18n.Tf("qdisc.persist2", moduleName))
	} else {
		_ = removeFile(ctx, bbr.ModulesLoadConfPath)
		log.Logf(i18n.Tf("qdisc.persist3", qdiscName))
	}
}
