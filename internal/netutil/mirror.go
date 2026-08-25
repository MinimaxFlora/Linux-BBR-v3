package netutil

import (
	"os"
	"strings"
)

// 网络源配置（国内/国外支持）：
// 读取优先级 环境变量 BBRV3_MIRROR > 文件 /etc/bbrv3/mirror > 默认 auto。
// 取值：
//   - auto   自动：直连 GitHub，失败自动依次尝试默认镜像（静默切换）
//   - direct 仅直连 GitHub（国外环境 / 不想走代理时）
//   - URL    固定使用该镜像（如 https://ghfast.top/）
const (
	MirrorEnv    = "BBRV3_MIRROR"
	MirrorFile   = "/etc/bbrv3/mirror"
	MirrorAuto   = "auto"
	MirrorDirect = "direct"
)

// DefaultMirrors 默认镜像源（ghproxy 风格：<镜像>/<完整 URL>，按尝试顺序）。
var DefaultMirrors = []string{
	"https://gh-proxy.kejizero.xyz/",
	"https://gh-proxy.com/",
	"https://ghfast.top/",
}

// MirrorValues 返回网络源全部可选项（TUI 展示/选择用）。
func MirrorValues() []string {
	values := []string{MirrorAuto, MirrorDirect}
	for _, m := range DefaultMirrors {
		values = append(values, strings.TrimRight(m, "/"))
	}
	return values
}

// CurrentMirror 读取当前网络源配置。
func CurrentMirror() string {
	if v := os.Getenv(MirrorEnv); v != "" {
		return v
	}
	if b, err := os.ReadFile(MirrorFile); err == nil {
		if v := strings.TrimSpace(string(b)); v != "" {
			return v
		}
	}
	return MirrorAuto
}

// SetMirror 持久化网络源配置（写 /etc/bbrv3/mirror，目录不存在则创建）。
func SetMirror(v string) error {
	v = strings.TrimSpace(v)
	if v == "" {
		v = MirrorAuto
	}
	if err := os.MkdirAll("/etc/bbrv3", 0o755); err != nil {
		return err
	}
	return os.WriteFile(MirrorFile, []byte(v), 0o644)
}

// candidateURLs 按当前配置生成尝试的 URL 列表（直连 + 镜像）。
func candidateURLs(raw string) []string {
	switch mode := CurrentMirror(); mode {
	case MirrorDirect:
		return []string{raw}
	case MirrorAuto:
		urls := make([]string, 0, 1+len(DefaultMirrors))
		urls = append(urls, raw)
		for _, m := range DefaultMirrors {
			urls = append(urls, mirrorURL(m, raw))
		}
		return urls
	default: // 固定镜像
		return []string{mirrorURL(mode, raw)}
	}
}

// mirrorURL ghproxy 风格拼接：<镜像>/<完整 URL>。
func mirrorURL(mirror, raw string) string {
	return strings.TrimRight(mirror, "/") + "/" + raw
}

// IsMirrorURL 判断 URL 是否走镜像（非 api.github.com 直连）。
// 走镜像的请求不携带 Authorization，避免把用户凭据交给第三方代理。
func IsMirrorURL(u string) bool {
	return !strings.HasPrefix(u, "https://api.github.com/")
}
