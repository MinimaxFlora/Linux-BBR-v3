package system

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
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
	log.Logf("正在应用亚太机器 TCP 调优...")

	if !sysctlSetAll(ctx, log,
		"net.ipv4.tcp_wmem="+apacWmem,
		"net.ipv4.tcp_rmem="+apacRmem,
		"net.ipv4.tcp_limit_output_bytes="+outputBytesDefault,
		"net.ipv4.tcp_slow_start_after_idle=0",
	) {
		log.Logf("✘ 亚太机器 TCP 调优应用失败，请检查当前内核是否支持这些 sysctl 项。")
		return fmt.Errorf("亚太 TCP 调优应用失败")
	}
	log.Logf("✔ 亚太机器 TCP 调优已立即生效")

	CleanSmartTuningConf(ctx, log)
	if err := AppendSysctlConf(ctx, log,
		"net.ipv4.tcp_wmem = "+apacWmem,
		"net.ipv4.tcp_rmem = "+apacRmem,
		"net.ipv4.tcp_limit_output_bytes = "+outputBytesDefault,
		"net.ipv4.tcp_slow_start_after_idle = 0",
	); err != nil {
		return err
	}
	log.Logf("✔ 亚太机器 TCP 调优已永久写入：%s", bbr.SysctlConfPath)
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
		log.Logf("✘ BBR + FQ 启用失败，请确认当前内核支持 BBR 和 fq。")
		return fmt.Errorf("BBR + FQ 启用失败")
	}
	log.Logf("✔ 已启用 BBR + FQ")
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
		log.Logf("✘ BBR v3 智能带宽优化应用失败，请检查当前内核是否支持这些 sysctl 项。")
		return nil, fmt.Errorf("智能带宽优化应用失败")
	}
	log.Logf("✔ BBR v3 智能带宽优化已立即生效")

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

	log.Logf("✔ 智能优化配置已永久写入：%s", bbr.SysctlConfPath)
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
	log.Logf("  线路模式：               %s", green(r.RegionLabel))
	log.Logf("  手动 RTT：                %s ms", green(r.RTTMS))
	log.Logf("  上传/下载：               %s Mbit/s", green(fmt.Sprintf("%d/%d", r.UploadMbps, r.DownloadMbps)))
	log.Logf("  推荐缓冲区：             %sMB", green(strconv.Itoa(r.BufferMB)))
	log.Logf("  内存保护上限：           %sMB", green(strconv.Itoa(r.CapMB)))
	log.Logf("  队列算法：               %s", green(SysctlGet(ctx, "net.core.default_qdisc")))
	log.Logf("  拥塞控制：               %s", green(SysctlGet(ctx, "net.ipv4.tcp_congestion_control")))
	log.Logf("  tcp_wmem:                 %s", green(SysctlGet(ctx, "net.ipv4.tcp_wmem")))
	log.Logf("  tcp_rmem:                 %s", green(SysctlGet(ctx, "net.ipv4.tcp_rmem")))
	log.Logf("  tcp_limit_output_bytes:   %s", green(SysctlGet(ctx, "net.ipv4.tcp_limit_output_bytes")))
	log.Logf("  tcp_slow_start_after_idle:%s", green(" "+SysctlGet(ctx, "net.ipv4.tcp_slow_start_after_idle")))
}

// ApplyExtremeTuning 应用疯批模式（原 apply_extreme_speedtest_tuning）。
func ApplyExtremeTuning(ctx context.Context, log execx.Logger) error {
	log.Logf("正在应用 BBR v3 疯批模式...")
	log.Logf("该模式只适合自有链路极限测速，不适合日常使用。")
	log.Logf("它会优先压榨吞吐，可能显著增加重传、抖动、排队延迟和内存占用。")

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
			log.Logf("✔ 当前网卡 %s 的 txqueuelen 已拉高到 %d", iface, extremeTxqueuelen)
		} else {
			log.Logf("⚠ 当前网卡 %s 设置 txqueuelen 失败，继续应用 TCP 参数", iface)
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
		log.Logf("✘ 疯批模式核心参数应用失败，请检查当前内核是否支持这些 sysctl 项。")
		return fmt.Errorf("疯批模式核心参数应用失败")
	}
	log.Logf("✔ 核心极限测速参数已立即生效")

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

	log.Logf("✔ 疯批模式配置已永久写入：%s", bbr.SysctlConfPath)
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
	log.Logf("正在清空本程序写入的网络优化配置...")
	CleanSysctlConf(ctx, log)
	CleanSmartTuningConf(ctx, log)
	_ = removeFile(ctx, bbr.ModulesLoadConfPath)
	ReloadSysctl(ctx)

	log.Logf("✔ 已清空网络优化持久配置")
	log.Logf("  已清理：%s 中的 BBR/qdisc/TCP buffer 参数", bbr.SysctlConfPath)
	log.Logf("  已删除：%s", bbr.ModulesLoadConfPath)
	log.Logf("  当前运行态参数可能要到重启后完全恢复为系统默认值。")
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
