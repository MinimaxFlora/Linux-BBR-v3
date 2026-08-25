package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/netutil"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/system"

	tea "github.com/charmbracelet/bubbletea"
)

// errManualBandwidth 测速失败，需要用户手动输入带宽。
var errManualBandwidth = errors.New("speedtest failed")

// githubToken 从环境读取 GitHub token（支持 GITHUB_TOKEN / GH_TOKEN）。
func githubToken() string {
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t
	}
	return os.Getenv("GH_TOKEN")
}

// currentEnv 返回当前环境（由 main 注入）。
func currentEnv() *system.Env { return envSingleton }

func bbrProfile(i int) bbr.Profile {
	if i == 1 {
		return bbr.ProfileMax
	}
	return bbr.ProfileStandard
}

func parseNum(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// ---------- 菜单 1：安装或更新最新版 ----------

func menuInstallLatest(m Model) (Model, tea.Cmd) {
	return m.askProfile(func(p bbr.Profile) (Model, tea.Cmd) {
		label := bbr.ProfileLabel(p)
		var installed bool
		m.installProfile = p
		m, cmd := m.startTask("🚀 安装或更新 BBR v3 (最新版)", func(ctx context.Context, log execx.Logger) error {
			return installLatestFlow(ctx, log, p, &installed)
		})
		m.afterTask = func(m Model) (Model, tea.Cmd) {
			if m.taskErr != nil {
				return m.showResult("❌ 安装失败", m.taskErr.Error())
			}
			if installed {
				return m.askConfirm("需要重启系统来加载新内核。是否立即重启？(y/n)",
					func(m Model) (Model, tea.Cmd) {
						return m.startTask("系统即将重启...", func(ctx context.Context, log execx.Logger) error {
							return system.RebootSystem(ctx)
						})
					},
					func(m Model) (Model, tea.Cmd) {
						return m.showResult("操作完成", "请记得稍后手动重启 ('sudo reboot') 来应用新内核。")
					})
			}
			return m.showResult("✔ 已是最新", fmt.Sprintf("您已安装最新 %s，无需更新！", label))
		}
		return m, cmd
	})
}

// installLatestFlow 最新版安装流程（对应 install_latest_version）。
func installLatestFlow(ctx context.Context, log execx.Logger, profile bbr.Profile, installed *bool) error {
	env := currentEnv()
	if err := system.AssertSupportedKernelInstallSystem(ctx, env.OS); err != nil {
		return err
	}
	log.Logf("正在从 GitHub 获取 %s 最新版本信息...", bbr.ProfileLabel(profile))

	releases, err := fetchReleases(ctx, log)
	if err != nil {
		return err
	}
	latestTag := latestTagFor(releases, env.ArchFilter, profile)
	if latestTag == "" {
		return fmt.Errorf("未找到适合当前架构 (%s) 的 %s 最新版本。", env.Arch, bbr.ProfileLabel(profile))
	}
	log.Logf("检测到最新版本：%s", latestTag)

	installedVer := system.InstalledKernelVersion(ctx, profile)
	if installedVer == "" {
		log.Logf("当前已安装版本：未安装")
	} else {
		log.Logf("当前已安装版本：%s", installedVer)
	}
	expected := bbr.ExpectedInstalledVersion(latestTag, profile)
	if installedVer != "" && installedVer == expected {
		log.Logf("您已安装最新 %s，无需更新！", bbr.ProfileLabel(profile))
		return nil
	}

	log.Logf("发现新版本或未安装内核，准备下载...")
	if err := downloadReleaseAssets(ctx, log, releases, latestTag); err != nil {
		return err
	}
	if err := system.InstallPackages(ctx, log, "/tmp"); err != nil {
		return err
	}
	*installed = true
	return nil
}

// ---------- 菜单 2：指定版本安装 ----------

func menuInstallSpecific(m Model) (Model, tea.Cmd) {
	return m.askProfile(func(p bbr.Profile) (Model, tea.Cmd) {
		m.installProfile = p
		var tags []string
		var releases []netutil.Release
		m, cmd := m.startTask("📚 指定版本安装", func(ctx context.Context, log execx.Logger) error {
			var err error
			tags, releases, err = listMatchingTags(ctx, log, p)
			return err
		})
		m.afterTask = func(m Model) (Model, tea.Cmd) {
			if m.taskErr != nil {
				return m.showResult("❌ 获取版本失败", m.taskErr.Error())
			}
			return m.askVersion(tags, func(tag string) (Model, tea.Cmd) {
				var installed bool
				m, cmd := m.startTask("📚 安装 "+tag, func(ctx context.Context, log execx.Logger) error {
					return installTagFlow(ctx, log, releases, tag, &installed)
				})
				m.afterTask = func(m Model) (Model, tea.Cmd) {
					if m.taskErr != nil {
						return m.showResult("❌ 安装失败", m.taskErr.Error())
					}
					if installed {
						return m.askConfirm("需要重启系统来加载新内核。是否立即重启？(y/n)",
							func(m Model) (Model, tea.Cmd) {
								return m.startTask("系统即将重启...", func(ctx context.Context, log execx.Logger) error {
									return system.RebootSystem(ctx)
								})
							},
							func(m Model) (Model, tea.Cmd) {
								return m.showResult("操作完成", "请记得稍后手动重启 ('sudo reboot') 来应用新内核。")
							})
					}
					return m.showResult("✔ 安装完成", "内核安装并配置完成！")
				}
				return m, cmd
			})
		}
		return m, cmd
	})
}

// listMatchingTags 获取并筛选当前架构匹配的 release tag 列表（升序）。
func listMatchingTags(ctx context.Context, log execx.Logger, profile bbr.Profile) ([]string, []netutil.Release, error) {
	env := currentEnv()
	if err := system.AssertSupportedKernelInstallSystem(ctx, env.OS); err != nil {
		return nil, nil, err
	}
	log.Logf("正在从 GitHub 获取 %s 版本信息...", bbr.ProfileLabel(profile))
	releases, err := fetchReleases(ctx, log)
	if err != nil {
		return nil, nil, err
	}
	var tags []string
	for _, r := range releases {
		if bbr.TagMatchesProfile(r.TagName, env.ArchFilter, profile) {
			tags = append(tags, r.TagName)
		}
	}
	if len(tags) == 0 {
		return nil, nil, fmt.Errorf("未找到适合当前架构的 %s 版本。", bbr.ProfileLabel(profile))
	}
	bbr.SortTagsByVersion(tags)
	return tags, releases, nil
}

// installTagFlow 下载并安装指定 tag（对应 install_specific_version 后半段）。
func installTagFlow(ctx context.Context, log execx.Logger, releases []netutil.Release, tag string, installed *bool) error {
	log.Logf("已选择版本：%s", tag)
	if err := downloadReleaseAssets(ctx, log, releases, tag); err != nil {
		return err
	}
	if err := system.InstallPackages(ctx, log, "/tmp"); err != nil {
		return err
	}
	*installed = true
	return nil
}

// ---------- 菜单 3：检查状态 ----------

func menuCheckStatus(m Model) (Model, tea.Cmd) {
	m, cmd := m.startTask("🔍 检查 BBR v3 状态", checkStatusFlow)
	m.afterTask = func(m Model) (Model, tea.Cmd) {
		if m.taskErr != nil {
			return m.showResult("❌ 检查失败", m.taskErr.Error())
		}
		return m.showResult("✔ 检查完成", "按 Enter 返回主菜单")
	}
	return m, cmd
}

// checkStatusFlow 检查 BBR v3 状态（对应菜单 3 的检查逻辑）。
func checkStatusFlow(ctx context.Context, log execx.Logger) error {
	env := currentEnv()

	// 1. tcp_bbr 模块
	modInfo := execx.TryOutput(ctx, "modinfo", "tcp_bbr")
	if modInfo == "" {
		log.Logf("正在刷新模块依赖...")
		_ = execx.RunOK(ctx, "depmod", "-a")
		modInfo = execx.TryOutput(ctx, "modinfo", "tcp_bbr")
	}
	if modInfo == "" {
		return errors.New("未加载 tcp_bbr 模块，无法检查版本。请先安装内核并重启。")
	}

	bbrVersion := parseModinfoVersion(modInfo)
	if bbrVersion == "3" {
		log.Logf("✔ BBR 模块版本：%s (v3)", bbrVersion)
	} else {
		log.Logf("检测到 BBR 模块，但版本是：%s，不是 v3！", bbrVersion)
	}

	// 2. 拥塞控制算法
	algo := env.CurrentAlgo
	if algo == "bbr" {
		log.Logf("✔ TCP 拥塞控制算法：%s", algo)
	} else {
		log.Logf("当前算法不是 bbr，而是：%s", algo)
	}

	// 3. 综合判定
	if bbrVersion == "3" && algo == "bbr" {
		log.Logf("ヽ(✿ﾟ▽ﾟ)ノ 检测完成，BBR v3 已正确安装并生效！")
	} else {
		log.Logf("BBR v3 未完全生效。请确保已安装内核并重启，然后使用选项 4-7 启用。")
	}

	// 4. Dirty Frag 缓解
	if system.DirtyFragMitigated(ctx) {
		log.Logf("✔ Dirty Frag 缓解状态：已启用（esp4/esp6/rxrpc 已黑名单）")
	} else {
		log.Logf("✘ Dirty Frag 缓解状态：未启用")
		log.Logf("  建议重新运行本程序，或手动写入 %s", bbr.SecurityModprobeConfPath)
	}
	return nil
}

// parseModinfoVersion 从 modinfo 输出解析 version 字段。
func parseModinfoVersion(modInfo string) string {
	for _, line := range strings.Split(modInfo, "\n") {
		if strings.HasPrefix(line, "version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "version:"))
		}
	}
	return ""
}

// ---------- 菜单 4-7：加速模式 ----------

func menuQdisc(qdisc string) func(Model) (Model, tea.Cmd) {
	upper := strings.ToUpper(qdisc)
	return func(m Model) (Model, tea.Cmd) {
		m, cmd := m.startTask("⚡ 启用 BBR + "+upper, func(ctx context.Context, log execx.Logger) error {
			return applyQdiscFlow(ctx, log, qdisc)
		})
		m.afterTask = func(m Model) (Model, tea.Cmd) {
			if m.taskErr != nil {
				return m.showResult("❌ 配置应用失败", m.taskErr.Error())
			}
			return m.askConfirm(fmt.Sprintf("要将这些配置永久保存到 %s 吗？(y/n)", bbr.SysctlConfPath),
				func(m Model) (Model, tea.Cmd) {
					m, cmd := m.startTask("永久保存配置", func(ctx context.Context, log execx.Logger) error {
						return saveQdiscFlow(ctx, log, qdisc)
					})
					m.afterTask = func(m Model) (Model, tea.Cmd) {
						if m.taskErr != nil {
							return m.showResult("❌ 保存失败", m.taskErr.Error())
						}
						return m.showResult("✔ 已永久保存", fmt.Sprintf("配置已写入 %s，重启后依然生效。", bbr.SysctlConfPath))
					}
					return m, cmd
				},
				func(m Model) (Model, tea.Cmd) {
					return m.showResult("未永久保存", "没有永久保存，重启后会恢复原设置呢~")
				})
		}
		return m, cmd
	}
}

// applyQdiscFlow 立即应用 bbr + qdisc（对应 ask_to_save 前半段）。
func applyQdiscFlow(ctx context.Context, log execx.Logger, qdisc string) error {
	if ok, msg := system.LoadQdiscModule(ctx, log, qdisc); !ok {
		log.Logf("⚠ %s", msg)
	}
	log.Logf("正在应用配置...")
	if err := system.SysctlSet(ctx, log, "net.core.default_qdisc", qdisc); err != nil {
		return fmt.Errorf("设置 default_qdisc 失败: %w", err)
	}
	if err := system.SysctlSet(ctx, log, "net.ipv4.tcp_congestion_control", "bbr"); err != nil {
		return fmt.Errorf("设置 tcp_congestion_control 失败: %w", err)
	}
	system.ApplyQdiscToActiveInterfaces(ctx, log, qdisc)

	// 验证是否生效
	newQdisc := system.SysctlGet(ctx, "net.core.default_qdisc")
	newAlgo := system.SysctlGet(ctx, "net.ipv4.tcp_congestion_control")
	if newQdisc == qdisc && newAlgo == "bbr" {
		log.Logf("✔ 配置已立即生效！")
		log.Logf("  当前队列算法：%s", newQdisc)
		log.Logf("  当前拥塞控制：%s", newAlgo)
		return nil
	}
	return fmt.Errorf("配置应用失败！队列算法期望：%s，实际：%s；拥塞控制期望：bbr，实际：%s。可能原因：当前内核不支持 %s 队列算法", qdisc, newQdisc, newAlgo, qdisc)
}

// saveQdiscFlow 永久保存配置（对应 ask_to_save 后半段）。
func saveQdiscFlow(ctx context.Context, log execx.Logger, qdisc string) error {
	system.CleanSysctlConf(ctx, log)
	if err := system.AppendSysctlConf(ctx, log,
		"net.core.default_qdisc="+qdisc,
		"net.ipv4.tcp_congestion_control=bbr",
	); err != nil {
		return err
	}
	system.ReloadSysctl(ctx)
	system.PersistQdiscModule(ctx, log, qdisc)
	return nil
}

// ---------- 菜单 8：亚太机器 TCP 调优 ----------

func menuAPACTuning(m Model) (Model, tea.Cmd) {
	m, cmd := m.startTask("🌏 亚太机器 TCP 调优", system.ApplyAPACTuning)
	m.afterTask = func(m Model) (Model, tea.Cmd) {
		if m.taskErr != nil {
			return m.showResult("❌ 调优失败", m.taskErr.Error())
		}
		return m.showResult("✔ 亚太调优完成", "按 Enter 返回主菜单")
	}
	return m, cmd
}

// ---------- 菜单 9：卸载内核 ----------

func menuUninstall(m Model) (Model, tea.Cmd) {
	return m.askConfirm("确定要卸载所有由本程序安装的 MinimaxFlora 内核包吗？(y/n)",
		func(m Model) (Model, tea.Cmd) {
			m, cmd := m.startTask("🗑️ 卸载 BBR 内核", func(ctx context.Context, log execx.Logger) error {
				_, err := system.RemoveInstalledKernels(ctx, log)
				return err
			})
			m.afterTask = func(m Model) (Model, tea.Cmd) {
				if m.taskErr != nil {
					return m.showResult("❌ 卸载失败", m.taskErr.Error())
				}
				return m.showResult("✔ 卸载完成", "内核包已卸载。请记得重启系统。")
			}
			return m, cmd
		},
		func(m Model) (Model, tea.Cmd) {
			return m.showResult("已取消", "未卸载任何内核包。")
		})
}

// ---------- 菜单 10：智能带宽优化 ----------

func menuSmartTuning(m Model) (Model, tea.Cmd) {
	var upload, download int
	m, cmd := m.startTask("🧠 BBR v3 智能带宽优化", func(ctx context.Context, log execx.Logger) error {
		var err error
		upload, download, err = smartTuningFlow(ctx, log)
		return err
	})
	m.afterTask = func(m Model) (Model, tea.Cmd) {
		if m.taskErr != nil {
			if errors.Is(m.taskErr, errManualBandwidth) {
				return m.askInput("请输入上传带宽(Mbit/s，默认 1000): ", func(v string) (Model, tea.Cmd) {
					if v == "" {
						v = "1000"
					}
					if !bbr.IsPositiveNumber(v) {
						m.inputErr = "请输入有效的正数。"
						return m, nil
					}
					f, _ := strconv.ParseFloat(v, 64)
					m.smartUpload = int(f)
					m.smartDownload = m.smartUpload
					return m.askRTTMode()
				})
			}
			return m.showResult("❌ 智能优化失败", m.taskErr.Error())
		}
		m.smartUpload = upload
		m.smartDownload = download
		return m.askRTTMode()
	}
	return m, cmd
}

// smartTuningFlow 智能优化前半段：启用 BBR+FQ 并测速（对应 apply_smart_bandwidth_tuning 前半段）。
func smartTuningFlow(ctx context.Context, log execx.Logger) (upload, download int, err error) {
	env := currentEnv()
	if err := system.EnableBBRFQ(ctx, log); err != nil {
		return 0, 0, err
	}
	log.Logf("正在运行 Ookla Speedtest 测速，请稍候...")
	log.Logf("测速只用于估算带宽；测速节点延迟不会显示，也不会用于 RTT 计算。")

	bin, err := netutil.EnsureOoklaSpeedtest(ctx, log, env.Arch)
	if err != nil {
		return 0, 0, errManualBandwidth
	}
	res, err := netutil.RunSpeedtest(ctx, log, bin)
	if err != nil {
		log.Logf("⚠ Speedtest 输出解析失败，正在清理 speedtest-cli 并重装 Ookla 官方版本后重试...")
		netutil.RemoveSpeedtestCLI(ctx, log)
		if bin2, err2 := netutil.EnsureOoklaSpeedtest(ctx, log, env.Arch); err2 == nil {
			res, err = netutil.RunSpeedtest(ctx, log, bin2)
		}
	}
	if err != nil {
		log.Logf("⚠ Speedtest 输出解析失败，将改为手动输入带宽。")
		return 0, 0, errManualBandwidth
	}
	log.Logf("  Download: %s Mbit/s", strconv.FormatFloat(res.Download, 'f', 0, 64))
	log.Logf("  Upload:   %s Mbit/s", strconv.FormatFloat(res.Upload, 'f', 0, 64))
	return int(res.Upload), int(res.Download), nil
}

// askRTTMode 选择 buffer 档位模式（对应 select_tuning_rtt）。
func (m Model) askRTTMode() (Model, tea.Cmd) {
	return m.askInput("请选择 buffer 档位模式：\n 1. 亚太档位（通常 RTT < 100ms）\n 2. 美欧档位（通常 RTT 150-300ms）\n 3. 手动 RTT + 手动档位\n请选择 (1-3): ", func(v string) (Model, tea.Cmd) {
		switch v {
		case "1":
			return m.askRTT("亚太档位（通常 RTT < 100ms）", "asia", "亚太")
		case "2":
			return m.askRTT("美欧档位（通常 RTT 150-300ms）", "overseas", "美欧")
		case "3":
			return m.askRTT("手动 RTT + 手动档位", "", "")
		default:
			m.inputErr = "请输入 1、2 或 3 选择线路模式。"
			return m, nil
		}
	})
}

// askRTT 输入真实链接延迟。
func (m Model) askRTT(prompt, regionCode, regionLabel string) (Model, tea.Cmd) {
	return m.askInput("请输入真实链接延迟(ms，v2rayN 测出来的即可): ", func(v string) (Model, tea.Cmd) {
		if !bbr.IsPositiveNumber(v) {
			m.inputErr = "请输入有效的正数，不能留空。"
			return m, nil
		}
		m.smartRTTMS = v
		if regionCode != "" {
			m.smartRegionCode = regionCode
			m.smartRegionLabel = regionLabel
			return m.smartApply()
		}
		// 模式 3：手动选择档位
		return m.askInput("请选择 buffer 档位模式：\n 1. 亚太档位\n 2. 美欧档位\n请选择 (1-2): ", func(bv string) (Model, tea.Cmd) {
			switch bv {
			case "1":
				m.smartRegionCode = "asia"
				m.smartRegionLabel = "手动 RTT / 亚太档"
			case "2":
				m.smartRegionCode = "overseas"
				m.smartRegionLabel = "手动 RTT / 美欧档"
			default:
				m.inputErr = "请输入 1 或 2 选择 buffer 档位。"
				return m, nil
			}
			return m.smartApply()
		})
	})
}

// smartApply 应用智能 buffer 优化（对应 apply_smart_bandwidth_tuning 后半段）。
func (m Model) smartApply() (Model, tea.Cmd) {
	m, cmd := m.startTask("应用 BBR v3 智能带宽优化", func(ctx context.Context, log execx.Logger) error {
		_, err := system.SmartApplyBuffers(ctx, log, m.smartUpload, m.smartDownload, m.smartRegionCode, m.smartRegionLabel, m.smartRTTMS)
		return err
	})
	m.afterTask = func(m Model) (Model, tea.Cmd) {
		if m.taskErr != nil {
			return m.showResult("❌ 智能优化失败", m.taskErr.Error())
		}
		return m.showResult("✔ 智能优化完成", "按 Enter 返回主菜单")
	}
	return m, cmd
}

// ---------- 菜单 11：清空网络优化配置 ----------

func menuClearOptimizations(m Model) (Model, tea.Cmd) {
	return m.askConfirm("确定要清空本程序写入的网络优化配置吗？(y/n)",
		func(m Model) (Model, tea.Cmd) {
			m, cmd := m.startTask("🧹 清空网络优化配置", system.ClearNetworkOptimizations)
			m.afterTask = func(m Model) (Model, tea.Cmd) {
				if m.taskErr != nil {
					return m.showResult("❌ 清空失败", m.taskErr.Error())
				}
				return m.showResult("✔ 已清空", "网络优化持久配置已清空，运行态参数重启后恢复默认。")
			}
			return m, cmd
		},
		func(m Model) (Model, tea.Cmd) {
			return m.showResult("已取消", "未清空任何配置。")
		})
}

// ---------- 菜单 12：疯批模式 ----------

func menuExtremeMode(m Model) (Model, tea.Cmd) {
	return m.askConfirm("⚠ BBR v3 疯批模式（极限测速挑战）\n该模式只适合自有链路极限测速，不适合日常使用。\n它会优先压榨吞吐，可能显著增加重传、抖动、排队延迟和内存占用。\n确定继续吗？(y/n)",
		func(m Model) (Model, tea.Cmd) {
			m, cmd := m.startTask("🧨 BBR v3 疯批模式", system.ApplyExtremeTuning)
			m.afterTask = func(m Model) (Model, tea.Cmd) {
				if m.taskErr != nil {
					return m.showResult("❌ 疯批模式失败", m.taskErr.Error())
				}
				return m.showResult("✔ 疯批模式已应用", "极限测速参数已生效，请勿长期使用。")
			}
			return m, cmd
		},
		func(m Model) (Model, tea.Cmd) {
			return m.showResult("已取消", "未应用疯批模式。")
		})
}

// ---------- 公共辅助 ----------

// fetchReleases 获取 releases 并处理 API 错误（rate limit 提示）。
func fetchReleases(ctx context.Context, log execx.Logger) ([]netutil.Release, error) {
	releases, err := netutil.FetchReleases(ctx, githubToken())
	if err != nil {
		var apiErr *netutil.GitHubAPIError
		if errors.As(err, &apiErr) {
			log.Logf("GitHub API 返回错误：%s", apiErr.Message)
			if apiErr.IsRateLimit() {
				log.Logf("提示：可先执行 export GITHUB_TOKEN=你的令牌，再重新运行本程序。")
			}
			return nil, err
		}
		log.Logf("从 GitHub 获取版本信息失败。请检查网络连接或 API 状态。")
		return nil, err
	}
	return releases, nil
}

// latestTagFor 按发布时间取最新匹配 tag。
func latestTagFor(releases []netutil.Release, arch string, profile bbr.Profile) string {
	var latest string
	var latestTime string
	for _, r := range releases {
		if !bbr.TagMatchesProfile(r.TagName, arch, profile) {
			continue
		}
		if latest == "" || r.PublishedAt > latestTime {
			latest = r.TagName
			latestTime = r.PublishedAt
		}
	}
	return latest
}

// downloadReleaseAssets 下载 release 中非 debug 资产到 /tmp（带进度日志）。
func downloadReleaseAssets(ctx context.Context, log execx.Logger, releases []netutil.Release, tag string) error {
	var assets []netutil.Asset
	for _, r := range releases {
		if r.TagName == tag {
			assets = r.NonDebugAssets()
			break
		}
	}
	if len(assets) == 0 {
		return fmt.Errorf("release %s 无可用资产", tag)
	}

	_ = execx.RunOK(ctx, "rm", "-f", "/tmp/linux-*.deb")
	for _, a := range assets {
		log.Logf("正在下载文件：%s", a.BrowserDownloadURL)
		dest := filepath.Join("/tmp", filepath.Base(a.BrowserDownloadURL))
		lastPct := 0
		err := netutil.Download(ctx, a.BrowserDownloadURL, dest, func(downloaded, total int64) {
			if total <= 0 {
				return
			}
			pct := int(downloaded * 100 / total)
			if pct >= lastPct+5 || pct == 100 {
				log.Logf("  下载进度：%d%%", pct)
				lastPct = pct
			}
		})
		if err != nil {
			return fmt.Errorf("下载失败：%s (%v)", a.BrowserDownloadURL, err)
		}
	}
	log.Logf("✔ 全部内核包下载完成")
	return nil
}

// envSingleton 由 main 注入的系统环境信息。
var envSingleton *system.Env

// SetEnv 注入系统环境信息（main 调用）。
func SetEnv(e *system.Env) { envSingleton = e }
