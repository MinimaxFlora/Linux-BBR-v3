package system

import (
	"os"
	"strings"
)

// green 返回原文本（颜色由 TUI 日志视图按前缀渲染）。
func green(s string) string { return s }

// readProcMeminfo 读取 /proc/meminfo。
func readProcMeminfo() (string, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// TrimSpace 便捷函数。
func TrimSpace(s string) string { return strings.TrimSpace(s) }
