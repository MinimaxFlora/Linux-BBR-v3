package system

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"
)

// 调优参数常量（与原脚本一致）。
const (
	apacWmem = "4096 16384 12582912"
	apacRmem = "4096 131072 33554432"

	smartWmemMin = "4096 65536"
	smartRmemMin = "4096 87380"

	extremeBufferBytes = 1073741824
	extremeOutputBytes = 268435456
	extremeBacklog     = 1000000
	extremeTxqueuelen  = 100000

	outputBytesDefault = "4194304"
)

// ApplyAPACTuning 应用亚太机器 TCP 调优（原 apply_apac_tuning）。
func ApplyAPACTuning(ctx context.Context, log execx.Logger) error {
	log.Logf(i18n.T("apac.applying"))

	if !sysctlSetAll(ctx, log,
		"net.ipv4.tcp_wmem="+apacWmem,
		"net.ipv4.tcp_rmem="+apacRmem,
		"net.ipv4.tcp_limit_output_bytes="+outputBytesDefault,
		"net.ipv4.tcp_slow_start_after_idle=0",
	) {
		log.Logf(i18n.T("apac.fail"))
		return fmt.Errorf("亚太 TCP 调优应用失败")
	}
	log.Logf(i18n.T("apac.ok"))

	CleanSmartTuningConf(ctx, log)
	if err := AppendSysctlConf(ctx, log,
		"net.ipv4.tcp_wmem = "+apacWmem,
		"net.ipv4.tcp_rmem = "+apacRmem,
		"net.ipv4.tcp_limit_output_bytes = "+outputBytesDefault,
		"net.ipv4.tcp_slow_start_after_idle = 0",
	); err != nil {
		return err
	}
	log.Logf(i18n.Tf("apac.saved", bbr.SysctlConfPath))
	logAPACResult(ctx, log)
	return nil
}

func logAPACResult(ctx context.Context, log execx.Logger) {
	log.Logf("  tcp_wmem:                 %s", green(SysctlGet(ctx, "net.ipv4.tcp_wmem")))
	log.Logf("  tcp_rmem:                 %s", green(SysctlGet(ctx, "net.ipv4.tcp_rmem")))
	log.Logf("  tcp_limit_output_bytes:   %s", green(SysctlGet(ctx, "net.ipv4.tcp_limit_output_bytes")))
	log.Logf("  tcp_slow_start_after_idle:%s", green(" "+SysctlGet(ctx, "net.ipv4.tcp_slow_start_after_idle")))
}

// EnableBBRFQ 启用 bbr + fq 拥塞控制与队列算法（原 apply_smart_bandwidth_tuning 前半段）。
func EnableBBRFQ(ctx context.Context, log execx.Logger) error {
	if ok, msg := LoadQdiscModule(ctx, log, "fq"); !ok {
		log.Logf("⚠ %s", msg)
	}
	if !sysctlSetAll(ctx, log,
		"net.core.default_qdisc=fq",
		"net.ipv4.tcp_congestion_control=bbr",
	) {
		log.Logf(i18n.T("smart.enableFail"))
		return fmt.Errorf("BBR + FQ 启用失败")
	}
	log.Logf(i18n.T("smart.enable"))
	return nil
}

// BufferResult 智能优化的计算结果。
type BufferResult struct {
	RegionLabel  string
	RTTMS        string
	UploadMbps   int
	DownloadMbps int
	BufferMB     int
	CapMB        int
	BufferBytes  int
}

// MemTotalKB 读取 /proc/meminfo 的 MemTotal（KB）。
func MemTotalKB() int64 {
	data, err := readProcMeminfo()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if n, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					return n
				}
			}
		}
	}
	return 0
}

// SmartApplyBuffers 计算并按带宽/地区应用智能 buffer（原 apply_smart_bandwidth_tuning 后半段）。
// regionCode: "asia" 或 "overseas"；返回计算结果用于界面展示。
func SmartApplyBuffers(ctx context.Context, log execx.Logger, uploadMbps, downloadMbps int, regionCode, regionLabel, rttMS string) (*BufferResult, error) {
	capMB := bbr.BufferCapMB(MemTotalKB())
	bufferMB := bbr.CalcSmartBufferMB(float64(uploadMbps), regionCode == "overseas", capMB)
	bufferBytes := bufferMB * 1024 * 1024

	if !sysctlSetAll(ctx, log,
		"net.core.rmem_max="+strconv.Itoa(bufferBytes),
		"net.core.wmem_max="+strconv.Itoa(bufferBytes),
		"net.ipv4.tcp_wmem="+smartWmemMin+" "+strconv.Itoa(bufferBytes),
		"net.ipv4.tcp_rmem="+smartRmemMin+" "+strconv.Itoa(bufferBytes),
		"net.ipv4.tcp_limit_output_bytes="+outputBytesDefault,
		"net.ipv4.tcp_slow_start_after_idle=0",
	) {
		log.Logf(i18n.T("smart.applyFail"))
		return nil, fmt.Errorf("智能带宽优化应用失败")
	}
	log.Logf(i18n.T("smart.applied"))

	CleanSysctlConf(ctx, log)
	CleanSmartTuningConf(ctx, log)
	if err := AppendSysctlConf(ctx, log,
		"net.core.default_qdisc=fq",
		"net.ipv4.tcp_congestion_control=bbr",
		"net.core.rmem_max = "+strconv.Itoa(bufferBytes),
		"net.core.wmem_max = "+strconv.Itoa(bufferBytes),
		"net.ipv4.tcp_wmem = "+smartWmemMin+" "+strconv.Itoa(bufferBytes),
		"net.ipv4.tcp_rmem = "+smartRmemMin+" "+strconv.Itoa(bufferBytes),
		"net.ipv4.tcp_limit_output_bytes = "+outputBytesDefault,
		"net.ipv4.tcp_slow_start_after_idle = 0",
	); err != nil {
		return nil, err
	}

	log.Logf(i18n.Tf("smart.saved", bbr.SysctlConfPath))
	res := &BufferResult{
		RegionLabel:  regionLabel,
		RTTMS:        rttMS,
		UploadMbps:   uploadMbps,
		DownloadMbps: downloadMbps,
		BufferMB:     bufferMB,
		CapMB:        capMB,
		BufferBytes:  bufferBytes,
	}
	logSmartResult(ctx, log, res)
	return res, nil
}

func logSmartResult(ctx context.Context, log execx.Logger, r *BufferResult) {
	log.Logf(i18n.Tf("smart.region", green(r.RegionLabel)))
	log.Logf(i18n.Tf("smart.rtt", green(r.RTTMS)))
	log.Logf(i18n.Tf("smart.bw", green(fmt.Sprintf("%d/%d", r.UploadMbps, r.DownloadMbps))))
	log.Logf(i18n.Tf("smart.buffer", green(strconv.Itoa(r.BufferMB))))
	log.Logf(i18n.Tf("smart.cap", green(strconv.Itoa(r.CapMB))))
	log.Logf("  队列算法：               %s", green(SysctlGet(ctx, "net.core.default_qdisc")))
	log.Logf("  拥塞控制：               %s", green(SysctlGet(ctx, "net.ipv4.tcp_congestion_control")))
	log.Logf("  tcp_wmem:                 %s", green(SysctlGet(ctx, "net.ipv4.tcp_wmem")))
	log.Logf("  tcp_rmem:                 %s", green(SysctlGet(ctx, "net.ipv4.tcp_rmem")))
	log.Logf("  tcp_limit_output_bytes:   %s", green(SysctlGet(ctx, "net.ipv4.tcp_limit_output_bytes")))
	log.Logf("  tcp_slow_start_after_idle:%s", green(" "+SysctlGet(ctx, "net.ipv4.tcp_slow_start_after_idle")))
}

// ApplyExtremeTuning 应用疯批模式（原 apply_extreme_speedtest_tuning）。
func ApplyExtremeTuning(ctx context.Context, log execx.Logger) error {
	log.Logf(i18n.T("extreme.doing"))
	log.Logf(i18n.T("extreme.tip1"))
	log.Logf(i18n.T("extreme.tip2"))

	if ok, msg := LoadQdiscModule(ctx, log, "fq"); !ok {
		log.Logf("⚠ %s", msg)
	}
	if !sysctlSetAll(ctx, log,
		"net.core.default_qdisc=fq",
		"net.ipv4.tcp_congestion_control=bbr",
	) {
		log.Logf("✘ BBR + FQ 启用失败，请确认当前内核支持 BBR 和 fq。")
		return fmt.Errorf("BBR + FQ 启用失败")
	}
	log.Logf("✔ 已启用 BBR + FQ")

	ApplyQdiscToActiveInterfaces(ctx, log, "fq")

	// txqueuelen 拉高（原 ip link set dev ... txqueuelen 100000）
	for _, iface := range DefaultRouteInterfaces(ctx) {
		if execx.RunOK(ctx, "ip", "link", "set", "dev", iface, "txqueuelen", strconv.Itoa(extremeTxqueuelen)) {
			log.Logf(i18n.Tf("extreme.txq", iface, extremeTxqueuelen))
		} else {
			log.Logf(i18n.Tf("extreme.txqF", iface))
		}
	}

	if !sysctlSetAll(ctx, log,
		"net.core.rmem_max="+strconv.Itoa(extremeBufferBytes),
		"net.core.wmem_max="+strconv.Itoa(extremeBufferBytes),
		"net.ipv4.tcp_wmem=4096 1048576 "+strconv.Itoa(extremeBufferBytes),
		"net.ipv4.tcp_rmem=4096 1048576 "+strconv.Itoa(extremeBufferBytes),
		"net.ipv4.tcp_limit_output_bytes="+strconv.Itoa(extremeOutputBytes),
		"net.ipv4.tcp_slow_start_after_idle=0",
	) {
		log.Logf(i18n.T("extreme.coreF"))
		return fmt.Errorf("疯批模式核心参数应用失败")
	}
	log.Logf(i18n.T("extreme.core"))

	// 附加参数：失败静默忽略
	for _, kv := range []string{
		"net.core.netdev_max_backlog=" + strconv.Itoa(extremeBacklog),
		"net.core.optmem_max=" + strconv.Itoa(extremeBufferBytes),
		"net.core.somaxconn=65535",
		"net.ipv4.tcp_notsent_lowat=4294967295",
		"net.ipv4.tcp_autocorking=0",
		"net.ipv4.tcp_no_metrics_save=1",
		"net.ipv4.tcp_mtu_probing=1",
		"net.ipv4.tcp_fastopen=3",
		"net.ipv4.tcp_window_scaling=1",
		"net.ipv4.tcp_moderate_rcvbuf=1",
		"net.ipv4.tcp_ecn=0",
	} {
		key, val, _ := strings.Cut(kv, "=")
		SysctlSetQuiet(ctx, key, val)
	}

	CleanSysctlConf(ctx, log)
	CleanSmartTuningConf(ctx, log)
	if err := AppendSysctlConf(ctx, log,
		"net.core.default_qdisc=fq",
		"net.ipv4.tcp_congestion_control=bbr",
		"net.core.rmem_max = "+strconv.Itoa(extremeBufferBytes),
		"net.core.wmem_max = "+strconv.Itoa(extremeBufferBytes),
		"net.core.optmem_max = "+strconv.Itoa(extremeBufferBytes),
		"net.core.netdev_max_backlog = "+strconv.Itoa(extremeBacklog),
		"net.core.somaxconn = 65535",
		"net.ipv4.tcp_wmem = 4096 1048576 "+strconv.Itoa(extremeBufferBytes),
		"net.ipv4.tcp_rmem = 4096 1048576 "+strconv.Itoa(extremeBufferBytes),
		"net.ipv4.tcp_limit_output_bytes = "+strconv.Itoa(extremeOutputBytes),
		"net.ipv4.tcp_slow_start_after_idle = 0",
		"net.ipv4.tcp_notsent_lowat = 4294967295",
		"net.ipv4.tcp_autocorking = 0",
		"net.ipv4.tcp_no_metrics_save = 1",
		"net.ipv4.tcp_mtu_probing = 1",
		"net.ipv4.tcp_fastopen = 3",
		"net.ipv4.tcp_window_scaling = 1",
		"net.ipv4.tcp_moderate_rcvbuf = 1",
		"net.ipv4.tcp_ecn = 0",
	); err != nil {
		return err
	}

	log.Logf(i18n.Tf("extreme.saved", bbr.SysctlConfPath))
	logExtremeResult(ctx, log)
	return nil
}

func logExtremeResult(ctx context.Context, log execx.Logger) {
	log.Logf("  队列算法：               %s", green(SysctlGet(ctx, "net.core.default_qdisc")))
	log.Logf("  拥塞控制：               %s", green(SysctlGet(ctx, "net.ipv4.tcp_congestion_control")))
	log.Logf("  tcp_wmem:                 %s", green(SysctlGet(ctx, "net.ipv4.tcp_wmem")))
	log.Logf("  tcp_rmem:                 %s", green(SysctlGet(ctx, "net.ipv4.tcp_rmem")))
	log.Logf("  tcp_limit_output_bytes:   %s", green(SysctlGet(ctx, "net.ipv4.tcp_limit_output_bytes")))
	log.Logf("  tcp_slow_start_after_idle:%s", green(" "+SysctlGet(ctx, "net.ipv4.tcp_slow_start_after_idle")))
}

// ClearNetworkOptimizations 清空本程序写入的网络优化配置（原 clear_network_optimizations）。
func ClearNetworkOptimizations(ctx context.Context, log execx.Logger) error {
	log.Logf(i18n.T("clear.doing"))
	CleanSysctlConf(ctx, log)
	CleanSmartTuningConf(ctx, log)
	_ = removeFile(ctx, bbr.ModulesLoadConfPath)
	ReloadSysctl(ctx)

	log.Logf(i18n.T("clear.done"))
	log.Logf(i18n.Tf("clear.cleaned", bbr.SysctlConfPath))
	log.Logf(i18n.Tf("clear.deleted", bbr.ModulesLoadConfPath))
	log.Logf(i18n.T("clear.note"))
	return nil
}

// sysctlSetAll 依次设置多个 sysctl（key=value），全部成功返回 true。
func sysctlSetAll(ctx context.Context, log execx.Logger, kvs ...string) bool {
	for _, kv := range kvs {
		key, val, _ := strings.Cut(kv, "=")
		if err := SysctlSet(ctx, log, key, val); err != nil {
			return false
		}
	}
	return true
}
