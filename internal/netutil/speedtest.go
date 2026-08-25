package netutil

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
)

// OoklaSpeedtestVersion 固定安装的 Ookla 官方 CLI 版本（与原脚本一致）。
const OoklaSpeedtestVersion = "1.2.0"

const speedtestInstallPath = "/usr/local/bin/speedtest"

var (
	downloadRe = regexp.MustCompile(`(?i)[Dd]ownload:\s*([0-9]+(?:\.[0-9]+)?)`)
	uploadRe   = regexp.MustCompile(`(?i)[Uu]pload:\s*([0-9]+(?:\.[0-9]+)?)`)
	serverIDRe = regexp.MustCompile(`^\s*([0-9]+)`)
)

// SpeedtestURL 返回当前架构的 Ookla speedtest 下载地址。
func SpeedtestURL(arch string) (string, error) {
	switch arch {
	case "x86_64":
		return fmt.Sprintf("https://install.speedtest.net/app/cli/ookla-speedtest-%s-linux-x86_64.tgz", OoklaSpeedtestVersion), nil
	case "aarch64":
		return fmt.Sprintf("https://install.speedtest.net/app/cli/ookla-speedtest-%s-linux-aarch64.tgz", OoklaSpeedtestVersion), nil
	default:
		return "", fmt.Errorf("当前架构 %s 暂无内置 Ookla speedtest 下载地址", arch)
	}
}

// IsOoklaSpeedtest 判断二进制是否为 Ookla 官方 CLI（对应 is_ookla_speedtest）。
func IsOoklaSpeedtest(binPath string) bool {
	if binPath == "" {
		return false
	}
	out, err := exec.Command(binPath, "--version").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Speedtest by Ookla "+OoklaSpeedtestVersion)
}

// RemoveSpeedtestCLI 移除 Python 版 speedtest-cli（对应 remove_speedtest_cli）。
func RemoveSpeedtestCLI(ctx context.Context, log execx.Logger) {
	// 检查 PATH 中的 speedtest 是否为非 Ookla 版本
	pathOut := execx.TryOutput(ctx, "command", "-v", "speedtest")
	if pathOut != "" && !IsOoklaSpeedtest(strings.TrimSpace(pathOut)) {
		versionOut := execx.TryOutput(ctx, strings.TrimSpace(pathOut), "--version")
		if strings.Contains(strings.ToLower(versionOut), "speedtest-cli") || strings.Contains(strings.ToLower(versionOut), "python") {
			log.Logf("检测到非 Ookla 官方 speedtest，正在移除 speedtest-cli...")
			_ = execx.RunOK(ctx, "apt-get", "remove", "--purge", "-y", "speedtest-cli")
		}
		if pathOut != speedtestInstallPath {
			_ = execx.RunOK(ctx, "rm", "-f", pathOut)
		}
	}
	// dpkg 中残留的 speedtest-cli 包
	if dpkgInstalled(ctx, "speedtest-cli") {
		log.Logf("检测到 speedtest-cli 软件包，正在移除...")
		_ = execx.RunOK(ctx, "apt-get", "remove", "--purge", "-y", "speedtest-cli")
	}
}

func dpkgInstalled(ctx context.Context, pkg string) bool {
	out := execx.TryOutput(ctx, "dpkg", "-l", pkg)
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "ii" && fields[1] == pkg {
			return true
		}
	}
	return false
}

// InstallOoklaSpeedtest 下载并安装 Ookla 官方 speedtest（对应 install_ookla_speedtest）。
func InstallOoklaSpeedtest(ctx context.Context, log execx.Logger, arch string) error {
	url, err := SpeedtestURL(arch)
	if err != nil {
		return err
	}
	log.Logf("正在安装 Ookla speedtest %s...", OoklaSpeedtestVersion)

	tmpDir, err := os.MkdirTemp("", "bbrv3-speedtest-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	tgzPath := filepath.Join(tmpDir, "speedtest.tgz")
	if err := Download(ctx, url, tgzPath, nil); err != nil {
		return fmt.Errorf("下载 speedtest 失败: %w", err)
	}
	if err := extractTGZ(tgzPath, tmpDir); err != nil {
		return fmt.Errorf("解压 speedtest 失败: %w", err)
	}

	binSrc := filepath.Join(tmpDir, "speedtest")
	if err := copyFile(binSrc, speedtestInstallPath, 0o755); err != nil {
		return err
	}
	if !IsOoklaSpeedtest(speedtestInstallPath) {
		return fmt.Errorf("Ookla speedtest 安装后校验失败")
	}
	log.Logf("✔ Ookla speedtest %s 安装完成", OoklaSpeedtestVersion)
	return nil
}

// EnsureOoklaSpeedtest 确保 Ookla 官方 speedtest 可用（对应 ensure_ookla_speedtest）。
// 返回可用二进制路径。
func EnsureOoklaSpeedtest(ctx context.Context, log execx.Logger, arch string) (string, error) {
	RemoveSpeedtestCLI(ctx, log)

	// PATH 中已有 Ookla 版本则直接使用
	if p := execx.TryOutput(ctx, "command", "-v", "speedtest"); p != "" && IsOoklaSpeedtest(strings.TrimSpace(p)) {
		return strings.TrimSpace(p), nil
	}
	if err := InstallOoklaSpeedtest(ctx, log, arch); err != nil {
		return "", err
	}
	return speedtestInstallPath, nil
}

// SpeedtestResult 一次测速的结果。
type SpeedtestResult struct {
	Download float64
	Upload   float64
}

// RunSpeedtest 执行测速并解析 Download/Upload（对应 run_speedtest_once）。
// 隐藏测速节点延迟：输出中的 Ping/Jitter 不解析、不返回。
func RunSpeedtest(ctx context.Context, log execx.Logger, binPath string) (*SpeedtestResult, error) {
	servers := speedtestServers(ctx, binPath)
	if len(servers) == 0 {
		servers = []string{"auto"}
	}

	attempts := 0
	for _, serverID := range servers {
		if attempts >= 5 {
			break
		}
		attempts++

		var output string
		if serverID == "auto" {
			output = execx.TryOutput(ctx, binPath, "--accept-license", "--accept-gdpr")
		} else {
			output = execx.TryOutput(ctx, binPath, "--accept-license", "--accept-gdpr", "--server-id="+serverID)
		}

		res := parseSpeedtestOutput(output)
		if res != nil && !strings.Contains(strings.ToLower(output), "failed") && !strings.Contains(strings.ToLower(output), "error") {
			return res, nil
		}
	}
	return nil, fmt.Errorf("Speedtest 输出解析失败")
}

// speedtestServers 获取前 10 个测速服务器 id。
func speedtestServers(ctx context.Context, binPath string) []string {
	out := execx.TryOutput(ctx, binPath, "--accept-license", "--accept-gdpr", "--servers")
	var ids []string
	for _, line := range strings.Split(out, "\n") {
		if m := serverIDRe.FindStringSubmatch(line); m != nil {
			ids = append(ids, m[1])
			if len(ids) >= 10 {
				break
			}
		}
	}
	return ids
}

func parseSpeedtestOutput(out string) *SpeedtestResult {
	d := downloadRe.FindStringSubmatch(out)
	u := uploadRe.FindStringSubmatch(out)
	if len(u) == 0 {
		return nil // 必须解析出 upload（原脚本以 upload 判定成功）
	}
	res := &SpeedtestResult{}
	if len(d) > 1 {
		if v, err := strconv.ParseFloat(d[1], 64); err == nil {
			res.Download = v
		}
	}
	if v, err := strconv.ParseFloat(u[1], 64); err == nil {
		res.Upload = v
	}
	return res
}

// ---- 辅助 ----

func extractTGZ(tgzPath, dest string) error {
	f, err := os.Open(tgzPath)
	if err != nil {
		return err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.Base(hdr.Name))
		if hdr.Typeflag == tar.TypeReg {
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}
