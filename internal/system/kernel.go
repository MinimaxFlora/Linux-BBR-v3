package system

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
)

// InstalledKernelVersion 获取已安装的本项目内核版本（对应 get_installed_version）。
// profile: standard / max / any。返回内核版本字符串（如 7.1.8-MinimaxFlora-bbrv3-max），未安装返回空。
func InstalledKernelVersion(ctx context.Context, profile bbr.Profile) string {
	out := execx.TryOutput(ctx, "dpkg", "-l")
	var versions []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "ii" {
			continue
		}
		name := fields[1]
		if !strings.HasPrefix(name, "linux-image-") || !strings.Contains(name, bbr.Brand) {
			continue
		}
		versions = append(versions, strings.TrimPrefix(name, "linux-image-"))
	}
	if len(versions) == 0 {
		return ""
	}

	var filtered []string
	switch profile {
	case bbr.ProfileStandard:
		for _, v := range versions {
			if strings.HasSuffix(v, bbr.StandardVersionSuffix) {
				filtered = append(filtered, v)
			}
		}
	case bbr.ProfileMax:
		for _, v := range versions {
			if strings.HasSuffix(v, bbr.MaxVersionSuffix) {
				filtered = append(filtered, v)
			}
		}
	default:
		filtered = versions
	}
	if len(filtered) == 0 {
		return ""
	}
	sort.Slice(filtered, func(i, j int) bool {
		return bbr.VersionGE(filtered[j], filtered[i])
	})
	return filtered[0]
}

// InstalledKernelPackages 列出已安装的本项目内核包全名。
func InstalledKernelPackages(ctx context.Context) []string {
	out := execx.TryOutput(ctx, "dpkg", "-l")
	var pkgs []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "ii" {
			continue
		}
		if strings.Contains(fields[1], bbr.Brand) {
			pkgs = append(pkgs, fields[1])
		}
	}
	return pkgs
}

// UpdateBootloader 更新引导加载程序（对应 update_bootloader）。
func UpdateBootloader(ctx context.Context, log execx.Logger) error {
	log.Logf("正在更新引导加载程序...")
	if execx.HasCommand("update-grub") {
		log.Logf("检测到 GRUB，正在执行 update-grub...")
		if _, err := execx.Run(ctx, log, "update-grub"); err != nil {
			log.Logf("GRUB 更新失败！")
			return err
		}
		log.Logf("✔ GRUB 更新成功！")
		return nil
	}
	log.Logf("未找到 'update-grub'。您的系统可能使用 U-Boot 或其他引导程序。")
	log.Logf("在许多 ARM 系统上，内核安装包会自动处理引导更新，通常无需手动操作。")
	log.Logf("如果重启后新内核未生效，您可能需要手动更新引导配置，请参考您系统的文档。")
	return nil
}

// InstallPackages 安装 /tmp 下已下载的内核 deb 包（对应 install_packages）。
// 先卸载旧版本项目内核包，再 dpkg -i 安装并更新引导。
func InstallPackages(ctx context.Context, log execx.Logger, downloadDir string) error {
	debs, err := filepath.Glob(filepath.Join(downloadDir, "linux-*.deb"))
	if err != nil || len(debs) == 0 {
		log.Logf("错误：未在 %s 目录下找到内核文件，安装中止。", downloadDir)
		return fmt.Errorf("未找到内核 deb 包")
	}

	for _, deb := range debs {
		if !execx.RunOK(ctx, "dpkg-deb", "-I", deb) {
			log.Logf("当前系统无法读取安装包：%s", deb)
			log.Logf("可能原因：dpkg 版本过旧，不支持该压缩格式。建议升级 dpkg 后重试。")
			return fmt.Errorf("无法读取安装包 %s", deb)
		}
	}

	// 卸载旧版内核
	log.Logf("开始卸载旧版内核...")
	old := InstalledKernelPackages(ctx)
	if len(old) > 0 {
		args := append([]string{"remove", "--purge", "-y"}, old...)
		_, _ = execx.Run(ctx, log, "apt-get", args...)
	}

	// 安装新内核
	log.Logf("开始安装新内核...")
	installArgs := append([]string{"-i"}, debs...)
	installed, ierr := execx.Run(ctx, log, "dpkg", installArgs...)
	if ierr != nil {
		log.Logf("dpkg 输出：%s", installed)
		log.Logf("内核安装失败！系统可能处于不稳定状态。请不要重启并寻求手动修复！")
		return fmt.Errorf("dpkg 安装失败: %w", ierr)
	}
	if err := UpdateBootloader(ctx, log); err != nil {
		log.Logf("内核安装或引导更新失败！系统可能处于不稳定状态。请不要重启并寻求手动修复！")
		return err
	}
	log.Logf("✔ 内核安装并配置完成！")
	return nil
}

// RemoveInstalledKernels 卸载所有本项目安装的内核包（对应菜单 9）。
func RemoveInstalledKernels(ctx context.Context, log execx.Logger) (removed bool, err error) {
	pkgs := InstalledKernelPackages(ctx)
	if len(pkgs) == 0 {
		log.Logf("未找到由本程序安装的 '%s' 内核包。", bbr.Brand)
		return false, nil
	}
	log.Logf("将要卸载以下内核包: %s", strings.Join(pkgs, " "))
	args := append([]string{"remove", "--purge", "-y"}, pkgs...)
	if _, err := execx.Run(ctx, log, "apt-get", args...); err != nil {
		return true, err
	}
	if err := UpdateBootloader(ctx, log); err != nil {
		return true, err
	}
	log.Logf("✔ 内核包已卸载。请记得重启系统。")
	return true, nil
}

// RebootSystem 立即重启。
func RebootSystem(ctx context.Context) error {
	return exec.CommandContext(ctx, "reboot").Run()
}
