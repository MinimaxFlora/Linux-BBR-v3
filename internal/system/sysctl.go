package system

import (
	"context"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
)

// SysctlGet 读取 sysctl 参数值（空字符串表示读取失败）。
func SysctlGet(ctx context.Context, key string) string {
	return strings.TrimSpace(execx.TryOutput(ctx, "sysctl", "-n", key))
}

// SysctlSet 立即设置 sysctl 参数。
func SysctlSet(ctx context.Context, log execx.Logger, key, value string) error {
	_, err := execx.Run(ctx, log, "sysctl", "-w", key+"="+value)
	return err
}

// SysctlSetQuiet 立即设置 sysctl 参数，失败静默忽略（用于附加参数）。
func SysctlSetQuiet(ctx context.Context, key, value string) {
	_ = execx.RunOK(ctx, "sysctl", "-w", key+"="+value)
}

// cleanSysctlConf 从持久化配置中按关键字删除行（对应 clean_smart_tuning_conf）。
// keys 为参数名前缀；删掉 SYSCTL_CONF 中所有包含这些 key 的行。
func cleanSysctlConf(ctx context.Context, log execx.Logger, confPath string, keys []string) {
	_ = execx.RunOK(ctx, "touch", confPath)
	for _, k := range keys {
		// sed -i '/net.core.default_qdisc/d' file
		_, _ = execx.Run(ctx, log, "sed", "-i", "/"+k+"/d", confPath)
	}
}

// 原脚本 clean_sysctl_conf 与 clean_smart_tuning_conf 覆盖的全部参数 key。
var smartTuningKeys = []string{
	"net.core.rmem_max", "net.core.wmem_max", "net.core.optmem_max",
	"net.core.netdev_max_backlog", "net.core.somaxconn",
	"net.ipv4.tcp_wmem", "net.ipv4.tcp_rmem",
	"net.ipv4.tcp_limit_output_bytes", "net.ipv4.tcp_slow_start_after_idle",
	"net.ipv4.tcp_notsent_lowat", "net.ipv4.tcp_autocorking",
	"net.ipv4.tcp_no_metrics_save", "net.ipv4.tcp_mtu_probing",
	"net.ipv4.tcp_fastopen", "net.ipv4.tcp_window_scaling",
	"net.ipv4.tcp_moderate_rcvbuf", "net.ipv4.tcp_ecn",
}

var sysctlConfKeys = []string{"net.core.default_qdisc", "net.ipv4.tcp_congestion_control"}

// CleanSysctlConf 清理 SYSCTL_CONF 中的 qdisc/拥塞控制行。
func CleanSysctlConf(ctx context.Context, log execx.Logger) {
	cleanSysctlConf(ctx, log, bbr.SysctlConfPath, sysctlConfKeys)
}

// CleanSmartTuningConf 清理 SYSCTL_CONF 中的智能优化/TCP buffer 行。
func CleanSmartTuningConf(ctx context.Context, log execx.Logger) {
	cleanSysctlConf(ctx, log, bbr.SysctlConfPath, smartTuningKeys)
}

// AppendSysctlConf 向 SYSCTL_CONF 追加多行配置。
func AppendSysctlConf(ctx context.Context, log execx.Logger, lines ...string) error {
	content := strings.Join(lines, "\n") + "\n"
	return appendFile(ctx, log, bbr.SysctlConfPath, content)
}

// ReloadSysctl 重新加载 sysctl 配置。
func ReloadSysctl(ctx context.Context) {
	_ = execx.RunOK(ctx, "sysctl", "--system")
}
