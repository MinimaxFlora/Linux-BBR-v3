package system

import "testing"

// 依赖包名映射必须覆盖全部必需命令，且不能用虚拟包名/不存在包名。
func TestDepPackageMapping(t *testing.T) {
	required := []string{"curl", "wget", "dpkg", "awk", "sed", "sysctl", "jq"}
	for _, cmd := range required {
		if depPackage[cmd] == "" {
			t.Errorf("缺少 %s 的包名映射", cmd)
		}
	}
	// 关键修正：awk 是虚拟包（apt 直接装 awk 会失败），必须映射到 gawk；
	// sysctl 属于 procps 包。
	if depPackage["awk"] != "gawk" {
		t.Errorf("awk 应映射到 gawk, got %q", depPackage["awk"])
	}
	if depPackage["sysctl"] != "procps" {
		t.Errorf("sysctl 应映射到 procps, got %q", depPackage["sysctl"])
	}
}
