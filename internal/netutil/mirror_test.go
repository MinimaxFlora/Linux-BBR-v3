package netutil

import (
	"os"
	"testing"
)

func TestCandidateURLs(t *testing.T) {
	raw := "https://github.com/MinimaxFlora/Linux-BBR-v3/releases/download/bbrv3-cli/bbrv3-linux-amd64"

	t.Run("auto 默认直连+镜像", func(t *testing.T) {
		os.Unsetenv(MirrorEnv)
		urls := candidateURLs(raw)
		if len(urls) != 1+len(DefaultMirrors) {
			t.Fatalf("auto 应生成 %d 个候选, got %d", 1+len(DefaultMirrors), len(urls))
		}
		if urls[0] != raw {
			t.Errorf("首个候选应为直连, got %s", urls[0])
		}
		for i, m := range DefaultMirrors {
			want := mirrorURL(m, raw)
			if urls[i+1] != want {
				t.Errorf("候选 %d 应为 %s, got %s", i+1, want, urls[i+1])
			}
		}
	})

	t.Run("direct 仅直连", func(t *testing.T) {
		t.Setenv(MirrorEnv, MirrorDirect)
		urls := candidateURLs(raw)
		if len(urls) != 1 || urls[0] != raw {
			t.Fatalf("direct 应仅直连, got %v", urls)
		}
	})

	t.Run("固定镜像", func(t *testing.T) {
		t.Setenv(MirrorEnv, "https://ghfast.top/")
		urls := candidateURLs(raw)
		if len(urls) != 1 {
			t.Fatalf("固定镜像应只有 1 个候选, got %d", len(urls))
		}
		want := "https://ghfast.top/" + raw
		if urls[0] != want {
			t.Errorf("应为 %s, got %s", want, urls[0])
		}
	})
}

func TestIsMirrorURL(t *testing.T) {
	if IsMirrorURL("https://api.github.com/repos/x/y/releases") {
		t.Error("api.github.com 直连不应判为镜像")
	}
	if !IsMirrorURL("https://ghfast.top/https://api.github.com/repos/x/y/releases") {
		t.Error("带镜像前缀的 URL 应判为镜像")
	}
	if !IsMirrorURL("https://gh-proxy.com/https://github.com/x/y/releases/download/a/b") {
		t.Error("release 下载镜像 URL 应判为镜像")
	}
}
