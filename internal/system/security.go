package system

import (
	"bufio"
	"compress/gzip"
	"context"
	"os"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"
)

// SecurityRules Dirty Frag 风险面收敛规则（原 6 条：blacklist + install /bin/false）。
var SecurityRules = []string{
	"blacklist esp4",
	"install esp4 /bin/false",
	"blacklist esp6",
	"install esp6 /bin/false",
	"blacklist rxrpc",
	"install rxrpc /bin/false",
}

// 安全模块列表（尝试卸载已加载实例）。
var securityModules = []string{"esp4", "esp6", "rxrpc"}

// EnsureSecurityRule 确保规则存在于配置文件中（不存在则追加），返回是否新增。
func EnsureSecurityRule(ctx context.Context, rule string) (changed bool, err error) {
	content, err := os.ReadFile(bbr.SecurityModprobeConfPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == rule {
			return false, nil
		}
	}
	f, err := os.OpenFile(bbr.SecurityModprobeConfPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	if _, err := f.WriteString(rule + "\n"); err != nil {
		return false, err
	}
	return true, nil
}

// KernelDisablesAEAD 检查当前运行内核是否在配置侧关闭 CONFIG_CRYPTO_USER_API_AEAD。
func KernelDisablesAEAD(ctx context.Context) bool {
	release := execx.TryOutput(ctx, "uname", "-r")
	bootConfig := "/boot/config-" + release
	if f, err := os.Open(bootConfig); err == nil {
		defer f.Close()
		return configHasDisabledAEAD(f)
	}
	if f, err := os.Open("/proc/config.gz"); err == nil {
		defer f.Close()
		zr, gzErr := gzip.NewReader(f)
		if gzErr == nil {
			defer zr.Close()
			return configHasDisabledAEAD(zr)
		}
	}
	return false
}

func configHasDisabledAEAD(r interface{ Read([]byte) (int, error) }) bool {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "# CONFIG_CRYPTO_USER_API_AEAD is not set" {
			return true
		}
		if strings.HasPrefix(line, "CONFIG_CRYPTO_USER_API_AEAD=") {
			return false
		}
	}
	return false
}

// ApplySecurityMitigations 应用 Dirty Frag 风险面收敛（原 apply_security_mitigations）。
func ApplySecurityMitigations(ctx context.Context, log execx.Logger) error {
	changed := 0
	ensureFileExists(ctx, bbr.SecurityModprobeConfPath)

	// 管理标记
	marker := "# Managed by Linux-BBR-v3"
	if !fileContains(ctx, bbr.SecurityModprobeConfPath, marker) {
		_ = appendFile(ctx, log, bbr.SecurityModprobeConfPath, marker+"\n")
		changed++
	}

	// 旧版 algif_aead 黑名单迁移：当前内核已配置侧收敛时移除
	if hasLegacyAEADBlacklist(ctx, bbr.SecurityModprobeConfPath) {
		if KernelDisablesAEAD(ctx) {
			removeLineFromFile(ctx, bbr.SecurityModprobeConfPath, "blacklist algif_aead")
			removeLineFromFile(ctx, bbr.SecurityModprobeConfPath, "install algif_aead /bin/false")
			changed++
			log.Logf(i18n.T("sec.removedAead"))
		} else {
			log.Logf(i18n.T("sec.keepAead"))
		}
	}

	// Dirty Frag 缓解规则
	for _, rule := range SecurityRules {
		c, err := EnsureSecurityRule(ctx, rule)
		if err == nil && c {
			changed++
		}
	}

	// 卸载已加载的受影响模块
	for _, mod := range securityModules {
		if moduleLoaded(ctx, mod) {
			if execx.RunOK(ctx, "modprobe", "-r", mod) {
				log.Logf(i18n.Tf("sec.modUnloaded", mod))
			} else {
				log.Logf(i18n.Tf("sec.modBusy", mod))
			}
		}
	}

	if changed > 0 {
		log.Logf(i18n.Tf("sec.written", bbr.SecurityModprobeConfPath))
	}
	return nil
}

// DirtyFragMitigated 检查 esp4/esp6/rxrpc 黑名单是否已全部写入。
func DirtyFragMitigated(ctx context.Context) bool {
	for _, rule := range []string{"blacklist esp4", "blacklist esp6", "blacklist rxrpc"} {
		if !fileContains(ctx, bbr.SecurityModprobeConfPath, rule) {
			return false
		}
	}
	return true
}

func fileContains(ctx context.Context, path, needle string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line == needle {
			return true
		}
	}
	return false
}

func hasLegacyAEADBlacklist(ctx context.Context, path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "blacklist algif_aead" || trimmed == "install algif_aead /bin/false" {
			return true
		}
	}
	return false
}

func removeLineFromFile(ctx context.Context, path, exact string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	var kept []string
	for _, l := range lines {
		if strings.TrimSpace(l) != exact {
			kept = append(kept, l)
		}
	}
	_ = os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o644)
}
