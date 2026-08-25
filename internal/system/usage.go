// Package system 系统操作：sysctl、dpkg、模块、调优、系统资源。
package system

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// CPUStat /proc/stat 的 cpu 汇总采样（idle 含 iowait，与 top 口径一致）。
type CPUStat struct {
	Idle  uint64
	Total uint64
}

// ReadCPUStat 读取 /proc/stat 第一行（cpu 汇总）。非 Linux 返回错误。
func ReadCPUStat() (CPUStat, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return CPUStat{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)[1:] // user nice system idle iowait irq softirq steal ...
		var vals [8]uint64
		for i := 0; i < len(fields) && i < 8; i++ {
			n, _ := strconv.ParseUint(fields[i], 10, 64)
			vals[i] = n
		}
		var total uint64
		for _, v := range vals {
			total += v
		}
		return CPUStat{Idle: vals[3] + vals[4], Total: total}, nil
	}
	return CPUStat{}, sc.Err()
}

// Percent 计算相对上一次采样的 CPU 使用率（0-100）。prev 为零值时返回 0。
func (c CPUStat) Percent(prev CPUStat) float64 {
	dt := c.Total - prev.Total
	if dt == 0 {
		return 0
	}
	di := c.Idle - prev.Idle
	return 100 * (1 - float64(di)/float64(dt))
}

// MemStat 内存使用情况。
type MemStat struct {
	Used  uint64
	Total uint64
}

// ReadMemStat 读取 /proc/meminfo 的内存使用（used = total - available）。非 Linux 返回错误。
func ReadMemStat() (MemStat, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemStat{}, err
	}
	defer f.Close()

	var total, available uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			total, _ = strconv.ParseUint(strings.Fields(line)[1], 10, 64)
		case strings.HasPrefix(line, "MemAvailable:"):
			available, _ = strconv.ParseUint(strings.Fields(line)[1], 10, 64)
		}
		if total > 0 && available > 0 {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return MemStat{}, err
	}
	if total == 0 {
		return MemStat{}, os.ErrNotExist
	}
	return MemStat{Used: total - available, Total: total}, nil
}
