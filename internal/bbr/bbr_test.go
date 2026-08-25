package bbr

import "testing"

func TestVersionGE(t *testing.T) {
	cases := []struct {
		current, required string
		want              bool
	}{
		{"24.04", "24.04", true},
		{"24.04", "22.04", true},
		{"22.04", "24.04", false},
		{"12", "12", true},
		{"12", "11", true},
		{"11", "12", false},
		{"999", "12", true},
		{"25.10", "24.04", true},
		{"7.1.8", "7.0.11", true},
		{"7.0.12", "7.1.0", false},
	}
	for _, c := range cases {
		if got := VersionGE(c.current, c.required); got != c.want {
			t.Errorf("VersionGE(%q, %q) = %v, want %v", c.current, c.required, got, c.want)
		}
	}
}

func TestDebianVersionFromCodename(t *testing.T) {
	cases := map[string]string{
		"bookworm": "12", "trixie": "13", "forky": "14", "sid": "999", "unstable": "999",
	}
	for in, want := range cases {
		got, ok := DebianVersionFromCodename(in)
		if !ok || got != want {
			t.Errorf("DebianVersionFromCodename(%q) = %q,%v want %q", in, got, ok, want)
		}
	}
	if _, ok := DebianVersionFromCodename("noble"); ok {
		t.Error("noble 不应映射成功")
	}
}

func TestIsPositiveNumber(t *testing.T) {
	if !IsPositiveNumber("1000") {
		t.Error("1000 应为正数")
	}
	if !IsPositiveNumber("0.5") {
		t.Error("0.5 应为正数")
	}
	if IsPositiveNumber("0") || IsPositiveNumber("-1") || IsPositiveNumber("abc") || IsPositiveNumber("") {
		t.Error("0/-1/abc/空 不应为正数")
	}
}

func TestBufferCapMB(t *testing.T) {
	if got := BufferCapMB(0); got != 64 {
		t.Errorf("未知内存应返回 64, got %d", got)
	}
	if got := BufferCapMB(400 * 1024); got != 16 { // 400MB → 16
		t.Errorf("400MB 应返回 16, got %d", got)
	}
	if got := BufferCapMB(512 * 1024); got != 32 { // 512MB 恰好 → 32（原脚本 <524288 才 16）
		t.Errorf("512MB 应返回 32, got %d", got)
	}
	if got := BufferCapMB(1024 * 1024); got != 64 { // 1GB → 64（不小于 1GB）
		t.Errorf("1GB 应返回 64, got %d", got)
	}
	if got := BufferCapMB(900 * 1024); got != 32 { // 900MB → 32
		t.Errorf("900MB 应返回 32, got %d", got)
	}
}

func TestCalcSmartBufferMB(t *testing.T) {
	// 亚太档
	cases := []struct {
		bw      float64
		overseas bool
		want    int
	}{
		{100, false, 8},
		{600, false, 12},
		{1500, false, 16},
		{3000, false, 24},
		{8000, false, 28},
		{20000, false, 32},
		// 美欧档
		{100, true, 16},
		{600, true, 48},
		{1500, true, 64},
	}
	for _, c := range cases {
		if got := CalcSmartBufferMB(c.bw, c.overseas, 64); got != c.want {
			t.Errorf("CalcSmartBufferMB(%v, %v, 64) = %d, want %d", c.bw, c.overseas, got, c.want)
		}
	}
	// cap 限制
	if got := CalcSmartBufferMB(20000, true, 16); got != 16 {
		t.Errorf("cap 限制应返回 16, got %d", got)
	}
	// 非法带宽回退 1000Mbps → 亚太档 16MB（原脚本 1000 落在 [1000,2000) 档）
	if got := CalcSmartBufferMB(-5, false, 64); got != 16 {
		t.Errorf("非法带宽应回退为 1000Mbps→16MB, got %d", got)
	}
}

func TestArchFilter(t *testing.T) {
	if got, ok := ArchFilter("aarch64"); !ok || got != "arm64" {
		t.Errorf("aarch64 → arm64, got %q", got)
	}
	if got, ok := ArchFilter("x86_64"); !ok || got != "x86_64" {
		t.Errorf("x86_64 → x86_64, got %q", got)
	}
	if _, ok := ArchFilter("riscv64"); ok {
		t.Error("riscv64 不应支持")
	}
}

func TestProfileLabel(t *testing.T) {
	if ProfileLabel(ProfileMax) != "BBR v3 Max（激进吞吐内核）" {
		t.Error("Max label 错误")
	}
	if ProfileLabel(ProfileStandard) != "BBR v3 标准版" {
		t.Error("standard label 错误")
	}
}

func TestExpectedInstalledVersion(t *testing.T) {
	// 用户给出的案例：linux-headers-7.2.0-minimaxflora-bbrv3-max_7.2.0-1_amd64.deb
	// （Debian 包名必须全小写，不能用显示名 MinimaxFlora 的大写）
	if got := ExpectedInstalledVersion("x86_64-7.2.0-max", ProfileMax); got != "7.2.0-minimaxflora-bbrv3-max" {
		t.Errorf("ExpectedInstalledVersion max = %q", got)
	}
	if got := ExpectedInstalledVersion("arm64-7.0.11", ProfileStandard); got != "7.0.11-minimaxflora-bbrv3" {
		t.Errorf("ExpectedInstalledVersion standard = %q", got)
	}
}

func TestTagMatchesProfile(t *testing.T) {
	if !TagMatchesProfile("x86_64-7.1.8-max", "x86_64", ProfileMax) {
		t.Error("x86_64-7.1.8-max 应匹配 max")
	}
	if TagMatchesProfile("x86_64-7.1.8", "x86_64", ProfileMax) {
		t.Error("x86_64-7.1.8 不应匹配 max")
	}
	if !TagMatchesProfile("arm64-7.0.11", "arm64", ProfileStandard) {
		t.Error("arm64-7.0.11 应匹配 standard")
	}
	if TagMatchesProfile("arm64-7.0.11-max", "arm64", ProfileStandard) {
		t.Error("arm64-7.0.11-max 不应匹配 standard")
	}
	if TagMatchesProfile("x86_64-7.0.11", "arm64", ProfileStandard) {
		t.Error("架构不匹配应拒绝")
	}
}

func TestIsDebugAsset(t *testing.T) {
	if !IsDebugAsset("https://example.com/linux-headers-7.1.8-dbg_7.1.8-1_amd64.deb") {
		t.Error("dbg 资产应识别")
	}
	if !IsDebugAsset("https://example.com/linux-image-7.1.8-dbgsym_7.1.8-1_amd64.deb") {
		t.Error("dbgsym 资产应识别")
	}
	if IsDebugAsset("https://example.com/linux-headers-7.1.8_7.1.8-1_amd64.deb") {
		t.Error("正常资产不应识别为 debug")
	}
}

func TestSortTagsByVersion(t *testing.T) {
	tags := []string{"arm64-7.0.11-max", "arm64-7.0.12-max", "arm64-7.0.9-max", "arm64-7.1.0-max"}
	SortTagsByVersion(tags)
	want := []string{"arm64-7.0.9-max", "arm64-7.0.11-max", "arm64-7.0.12-max", "arm64-7.1.0-max"}
	for i := range want {
		if tags[i] != want[i] {
			t.Fatalf("排序错误: %v", tags)
		}
	}
}

func TestVersionFromTag(t *testing.T) {
	if got := VersionFromTag("x86_64-7.1.8-max"); got != "7.1.8" {
		t.Errorf("VersionFromTag = %q", got)
	}
	if got := VersionFromTag("arm64-7.0.11"); got != "7.0.11" {
		t.Errorf("VersionFromTag = %q", got)
	}
}
