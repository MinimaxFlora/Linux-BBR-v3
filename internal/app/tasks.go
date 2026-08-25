package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/bbr"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
	"github.com/MinimaxFlora/Linux-BBR-v3/internal/i18n"
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
		m, cmd := m.startTask(i18n.T("task.installLatest"), func(ctx context.Context, log execx.Logger) error {
			return installLatestFlow(ctx, log, p, &installed)
		})
		m.afterTask = func(m Model) (Model, tea.Cmd) {
			if m.taskErr != nil {
				return m.showResult(i18n.T("install.fail"), m.taskErr.Error())
			}
			if installed {
				return m.askConfirm(i18n.T("install.rebootAsk"),
					func(m Model) (Model, tea.Cmd) {
						return m.startTask(i18n.T("task.reboot"), func(ctx context.Context, log execx.Logger) error {
							return system.RebootSystem(ctx)
						})
					},
					func(m Model) (Model, tea.Cmd) {
						return m.showResult(i18n.T("common.cancel"), i18n.T("install.rebootLater"))
					})
			}
			return m.showResult(i18n.T("install.upToDateR"), i18n.Tf("install.upToDate", label))
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
	log.Logf(i18n.Tf("install.getting", bbr.ProfileLabel(profile)))

	releases, err := fetchReleases(ctx, log)
	if err != nil {
		return err
	}
	latestTag := latestTagFor(releases, env.ArchFilter, profile)
	if latestTag == "" {
		return errors.New(i18n.Tf("install.noLatest", env.Arch, bbr.ProfileLabel(profile)))
	}
	log.Logf(i18n.Tf("install.latestVer", latestTag))

	installedVer := system.InstalledKernelVersion(ctx, profile)
	if installedVer == "" {
		log.Logf(i18n.T("install.installed"))
	} else {
		log.Logf(i18n.Tf("install.installedV", installedVer))
	}
	expected := bbr.ExpectedInstalledVersion(latestTag, profile)
	if installedVer != "" && installedVer == expected {
		log.Logf(i18n.Tf("install.upToDate", bbr.ProfileLabel(profile)))
		return nil
	}

	log.Logf(i18n.T("install.foundNew"))
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
		m, cmd := m.startTask(i18n.T("task.installVer"), func(ctx context.Context, log execx.Logger) error {
			var err error
			tags, releases, err = listMatchingTags(ctx, log, p)
			return err
		})
		m.afterTask = func(m Model) (Model, tea.Cmd) {
			if m.taskErr != nil {
				return m.showResult(i18n.T("install.getFail"), m.taskErr.Error())
			}
			return m.askVersion(tags, func(tag string) (Model, tea.Cmd) {
				var installed bool
				m, cmd := m.startTask(i18n.Tf("task.installTag", tag), func(ctx context.Context, log execx.Logger) error {
					return installTagFlow(ctx, log, releases, tag, &installed)
				})
				m.afterTask = func(m Model) (Model, tea.Cmd) {
					if m.taskErr != nil {
						return m.showResult(i18n.T("install.fail"), m.taskErr.Error())
					}
					if installed {
						return m.askConfirm(i18n.T("install.rebootAsk"),
							func(m Model) (Model, tea.Cmd) {
								return m.startTask(i18n.T("task.reboot"), func(ctx context.Context, log execx.Logger) error {
									return system.RebootSystem(ctx)
								})
							},
							func(m Model) (Model, tea.Cmd) {
								return m.showResult(i18n.T("common.cancel"), i18n.T("install.rebootLater"))
							})
					}
					return m.showResult(i18n.T("install.doneR"), i18n.T("install.done"))
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
	log.Logf(i18n.Tf("install.versionList", bbr.ProfileLabel(profile)))
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
		return nil, nil, errors.New(i18n.Tf("install.noVersion", bbr.ProfileLabel(profile)))
	}
	bbr.SortTagsByVersion(tags)
	return tags, releases, nil
}

// installTagFlow 下载并安装指定 tag（对应 install_specific_version 后半段）。
func installTagFlow(ctx context.Context, log execx.Logger, releases []netutil.Release, tag string, installed *bool) error {
	log.Logf(i18n.Tf("install.selected", tag))
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
	m, cmd := m.startTask(i18n.T("task.check"), checkStatusFlow)
	m.afterTask = func(m Model) (Model, tea.Cmd) {
		if m.taskErr != nil {
			return m.showResult(i18n.T("check.fail"), m.taskErr.Error())
		}
		return m.showResult(i18n.T("check.title"), i18n.T("result.enter"))
	}
	return m, cmd
}

// checkStatusFlow 检查 BBR v3 状态（对应菜单 3 的检查逻辑）。
func checkStatusFlow(ctx context.Context, log execx.Logger) error {
	env := currentEnv()

	modInfo := execx.TryOutput(ctx, "modinfo", "tcp_bbr")
	if modInfo == "" {
		log.Logf(i18n.T("check.depmod"))
		_ = execx.RunOK(ctx, "depmod", "-a")
		modInfo = execx.TryOutput(ctx, "modinfo", "tcp_bbr")
	}
	if modInfo == "" {
		return errors.New(i18n.T("check.noModule"))
	}

	bbrVersion := parseModinfoVersion(modInfo)
	if bbrVersion == "3" {
		log.Logf(i18n.Tf("check.bbrVer", bbrVersion))
	} else {
		log.Logf(i18n.Tf("check.bbrNot3", bbrVersion))
	}

	algo := env.CurrentAlgo
	if algo == "bbr" {
		log.Logf(i18n.Tf("check.algo", algo))
	} else {
		log.Logf(i18n.Tf("check.algoNot", algo))
	}

	if bbrVersion == "3" && algo == "bbr" {
		log.Logf(i18n.T("check.allGood"))
	} else {
		log.Logf(i18n.T("check.notActive"))
	}

	if system.DirtyFragMitigated(ctx) {
		log.Logf(i18n.T("check.fragOk"))
	} else {
		log.Logf(i18n.T("check.fragNo"))
		log.Logf(i18n.Tf("check.fragTip", bbr.SecurityModprobeConfPath))
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

// ---------- 菜单 4：加速模式（原 4-7 合并） ----------

func menuQdisc(qdisc string) func(Model) (Model, tea.Cmd) {
	upper := strings.ToUpper(qdisc)
	return func(m Model) (Model, tea.Cmd) {
		m, cmd := m.startTask(i18n.Tf("task.qdisc", upper), func(ctx context.Context, log execx.Logger) error {
			return applyQdiscFlow(ctx, log, qdisc)
		})
		m.afterTask = func(m Model) (Model, tea.Cmd) {
			if m.taskErr != nil {
				return m.showResult(i18n.T("qdisc.appFail"), m.taskErr.Error())
			}
			return m.askConfirm(i18n.Tf("qdisc.saveAsk", bbr.SysctlConfPath),
				func(m Model) (Model, tea.Cmd) {
					m, cmd := m.startTask(i18n.T("task.save"), func(ctx context.Context, log execx.Logger) error {
						return saveQdiscFlow(ctx, log, qdisc)
					})
					m.afterTask = func(m Model) (Model, tea.Cmd) {
						if m.taskErr != nil {
							return m.showResult(i18n.T("qdisc.saveFail"), m.taskErr.Error())
						}
						return m.showResult(i18n.T("qdisc.saved"), i18n.Tf("qdisc.savedMsg", bbr.SysctlConfPath))
					}
					return m, cmd
				},
				func(m Model) (Model, tea.Cmd) {
					return m.showResult(i18n.T("qdisc.notSave"), i18n.T("qdisc.notSaveMsg"))
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
	log.Logf(i18n.T("qdisc.apply"))
	if err := system.SysctlSet(ctx, log, "net.core.default_qdisc", qdisc); err != nil {
		return errors.New(i18n.Tf("sys.sysctlFail", "default_qdisc", err))
	}
	if err := system.SysctlSet(ctx, log, "net.ipv4.tcp_congestion_control", "bbr"); err != nil {
		return errors.New(i18n.Tf("sys.sysctlFail", "tcp_congestion_control", err))
	}
	system.ApplyQdiscToActiveInterfaces(ctx, log, qdisc)

	newQdisc := system.SysctlGet(ctx, "net.core.default_qdisc")
	newAlgo := system.SysctlGet(ctx, "net.ipv4.tcp_congestion_control")
	if newQdisc == qdisc && newAlgo == "bbr" {
		log.Logf(i18n.T("qdisc.applied"))
		log.Logf(i18n.Tf("qdisc.curQdisc", newQdisc))
		log.Logf(i18n.Tf("qdisc.curAlgo", newAlgo))
		return nil
	}
	return errors.New(i18n.Tf("qdisc.fail", qdisc, newQdisc, newAlgo, qdisc))
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

// ---------- 菜单 5：亚太机器 TCP 调优 ----------

func menuAPACTuning(m Model) (Model, tea.Cmd) {
	m, cmd := m.startTask(i18n.T("task.apac"), system.ApplyAPACTuning)
	m.afterTask = func(m Model) (Model, tea.Cmd) {
		if m.taskErr != nil {
			return m.showResult(i18n.T("apac.titleF"), m.taskErr.Error())
		}
		return m.showResult(i18n.T("apac.title"), i18n.T("result.enter"))
	}
	return m, cmd
}

// ---------- 菜单 6：卸载内核 ----------

func menuUninstall(m Model) (Model, tea.Cmd) {
	return m.askConfirm(i18n.T("uninstall.ask"),
		func(m Model) (Model, tea.Cmd) {
			m, cmd := m.startTask(i18n.T("task.uninstall"), func(ctx context.Context, log execx.Logger) error {
				_, err := system.RemoveInstalledKernels(ctx, log)
				return err
			})
			m.afterTask = func(m Model) (Model, tea.Cmd) {
				if m.taskErr != nil {
					return m.showResult(i18n.T("uninstall.titleF"), m.taskErr.Error())
				}
				return m.showResult(i18n.T("uninstall.title"), i18n.T("uninstall.done"))
			}
			return m, cmd
		},
		func(m Model) (Model, tea.Cmd) {
			return m.showResult(i18n.T("uninstall.cancel"), i18n.T("uninstall.cancelMsg"))
		})
}

// ---------- 菜单 7：智能带宽优化 ----------

func menuSmartTuning(m Model) (Model, tea.Cmd) {
	var upload, download int
	m, cmd := m.startTask(i18n.T("task.smart"), func(ctx context.Context, log execx.Logger) error {
		var err error
		upload, download, err = smartTuningFlow(ctx, log)
		return err
	})
	m.afterTask = func(m Model) (Model, tea.Cmd) {
		if m.taskErr != nil {
			if errors.Is(m.taskErr, errManualBandwidth) {
				return m.askInput(i18n.T("smart.bandAsk"), func(v string) (Model, tea.Cmd) {
					if v == "" {
						v = "1000"
					}
					if !bbr.IsPositiveNumber(v) {
						m.inputErr = i18n.T("smart.positive")
						return m, nil
					}
					f, _ := strconv.ParseFloat(v, 64)
					m.smartUpload = int(f)
					m.smartDownload = m.smartUpload
					return m.askRTTMode()
				})
			}
			return m.showResult(i18n.T("smart.titleF"), m.taskErr.Error())
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
	log.Logf(i18n.T("smart.speedtest"))
	log.Logf(i18n.T("smart.speedtestTip"))

	bin, err := netutil.EnsureOoklaSpeedtest(ctx, log, env.Arch)
	if err != nil {
		return 0, 0, errManualBandwidth
	}
	res, err := netutil.RunSpeedtest(ctx, log, bin)
	if err != nil {
		log.Logf(i18n.T("smart.retry"))
		netutil.RemoveSpeedtestCLI(ctx, log)
		if bin2, err2 := netutil.EnsureOoklaSpeedtest(ctx, log, env.Arch); err2 == nil {
			res, err = netutil.RunSpeedtest(ctx, log, bin2)
		}
	}
	if err != nil {
		log.Logf(i18n.T("smart.manual"))
		return 0, 0, errManualBandwidth
	}
	log.Logf(i18n.Tf("smart.dl", strconv.FormatFloat(res.Download, 'f', 0, 64)))
	log.Logf(i18n.Tf("smart.ul", strconv.FormatFloat(res.Upload, 'f', 0, 64)))
	return int(res.Upload), int(res.Download), nil
}

// askRTTMode 选择 buffer 档位模式（对应 select_tuning_rtt）。
func (m Model) askRTTMode() (Model, tea.Cmd) {
	return m.askInput(i18n.T("smart.rttMode"), func(v string) (Model, tea.Cmd) {
		switch v {
		case "1":
			return m.askRTT(i18n.T("smart.rttAsk"), "asia", "亚太")
		case "2":
			return m.askRTT(i18n.T("smart.rttAsk"), "overseas", "美欧")
		case "3":
			return m.askRTT(i18n.T("smart.rttAsk"), "", "")
		default:
			m.inputErr = i18n.T("smart.rttModeErr")
			return m, nil
		}
	})
}

// askRTT 输入真实链接延迟。
func (m Model) askRTT(prompt, regionCode, regionLabel string) (Model, tea.Cmd) {
	return m.askInput(prompt, func(v string) (Model, tea.Cmd) {
		if !bbr.IsPositiveNumber(v) {
			m.inputErr = i18n.T("smart.positiveReq")
			return m, nil
		}
		m.smartRTTMS = v
		if regionCode != "" {
			m.smartRegionCode = regionCode
			m.smartRegionLabel = regionLabel
			return m.smartApply()
		}
		// 模式 3：手动选择档位
		return m.askInput(i18n.T("smart.bufMode"), func(bv string) (Model, tea.Cmd) {
			switch bv {
			case "1":
				m.smartRegionCode = "asia"
				m.smartRegionLabel = "手动 RTT / 亚太档"
			case "2":
				m.smartRegionCode = "overseas"
				m.smartRegionLabel = "手动 RTT / 美欧档"
			default:
				m.inputErr = i18n.T("smart.bufModeErr")
				return m, nil
			}
			return m.smartApply()
		})
	})
}

// smartApply 应用智能 buffer 优化（对应 apply_smart_bandwidth_tuning 后半段）。
func (m Model) smartApply() (Model, tea.Cmd) {
	m, cmd := m.startTask(i18n.T("task.smartApply"), func(ctx context.Context, log execx.Logger) error {
		_, err := system.SmartApplyBuffers(ctx, log, m.smartUpload, m.smartDownload, m.smartRegionCode, m.smartRegionLabel, m.smartRTTMS)
		return err
	})
	m.afterTask = func(m Model) (Model, tea.Cmd) {
		if m.taskErr != nil {
			return m.showResult(i18n.T("smart.titleF"), m.taskErr.Error())
		}
		return m.showResult(i18n.T("smart.title"), i18n.T("result.enter"))
	}
	return m, cmd
}

// ---------- 菜单 8：清空网络优化配置 ----------

func menuClearOptimizations(m Model) (Model, tea.Cmd) {
	return m.askConfirm(i18n.T("clear.ask"),
		func(m Model) (Model, tea.Cmd) {
			m, cmd := m.startTask(i18n.T("task.clear"), system.ClearNetworkOptimizations)
			m.afterTask = func(m Model) (Model, tea.Cmd) {
				if m.taskErr != nil {
					return m.showResult(i18n.T("clear.titleF"), m.taskErr.Error())
				}
				return m.showResult(i18n.T("clear.title"), i18n.T("clear.done"))
			}
			return m, cmd
		},
		func(m Model) (Model, tea.Cmd) {
			return m.showResult(i18n.T("clear.cancel"), i18n.T("clear.cancelMsg"))
		})
}

// ---------- 菜单 9：疯批模式 ----------

func menuExtremeMode(m Model) (Model, tea.Cmd) {
	return m.askConfirm(i18n.T("extreme.ask"),
		func(m Model) (Model, tea.Cmd) {
			m, cmd := m.startTask(i18n.T("task.extreme"), system.ApplyExtremeTuning)
			m.afterTask = func(m Model) (Model, tea.Cmd) {
				if m.taskErr != nil {
					return m.showResult(i18n.T("extreme.titleF"), m.taskErr.Error())
				}
				return m.showResult(i18n.T("extreme.title"), i18n.T("extreme.msg"))
			}
			return m, cmd
		},
		func(m Model) (Model, tea.Cmd) {
			return m.showResult(i18n.T("extreme.cancel"), i18n.T("extreme.cancelMsg"))
		})
}

// ---------- 菜单 10：检测 TUI 更新 ----------

func menuCheckUpdate(m Model) (Model, tea.Cmd) {
	var title, extra string
	m, cmd := m.startTask(i18n.T("task.checkUpdate"), func(ctx context.Context, log execx.Logger) error {
		t, e, err := checkTUIUpdate(ctx, log)
		if err != nil {
			return err
		}
		title, extra = t, e
		return nil
	})
	m.afterTask = func(m Model) (Model, tea.Cmd) {
		if m.taskErr != nil {
			return m.showResult(i18n.T("update.titleFail"), m.taskErr.Error())
		}
		return m.showResult(title, extra)
	}
	return m, cmd
}

// checkTUIUpdate 检测 TUI 是否有新版本：对比本地注入 Commit 与远端
// bbrv3-cli release 的 target_commitish（release 固定 tag，每次 push 覆盖上传）。
func checkTUIUpdate(ctx context.Context, log execx.Logger) (title, extra string, err error) {
	log.Logf(i18n.T("update.checking"))
	releases, err := fetchReleases(ctx, log)
	if err != nil {
		return "", "", err
	}
	var remoteCommit string
	for _, r := range releases {
		if r.TagName == "bbrv3-cli" {
			remoteCommit = r.TargetCommitish
			break
		}
	}
	if remoteCommit == "" {
		return "", "", errors.New(i18n.T("update.noRelease"))
	}
	short := func(s string) string {
		if len(s) > 8 {
			return s[:8]
		}
		return s
	}
	remote := short(remoteCommit)
	local := short(Commit)
	log.Logf(i18n.Tf("update.local", local))
	log.Logf(i18n.Tf("update.remote", remote))
	if local == "dev" {
		return i18n.T("update.devTitle"), i18n.Tf("update.devMsg", remote), nil
	}
	if strings.EqualFold(local, remote) {
		log.Logf(i18n.T("update.latest"))
		return i18n.T("update.latestTitle"), i18n.Tf("update.latestMsg", remote), nil
	}
	log.Logf(i18n.T("update.newFound"))
	return i18n.T("update.newTitle"), i18n.Tf("update.newMsg", remote, local), nil
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
				log.Logf(i18n.T("install.rateLimit"))
			}
			return nil, err
		}
		log.Logf(i18n.T("install.networkErr"))
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
		return errors.New(i18n.Tf("install.noAssets", tag))
	}

	_ = execx.RunOK(ctx, "rm", "-f", "/tmp/linux-*.deb")
	for _, a := range assets {
		log.Logf(i18n.Tf("install.downloading", a.BrowserDownloadURL))
		dest := filepath.Join("/tmp", filepath.Base(a.BrowserDownloadURL))
		lastPct := 0
		err := netutil.Download(ctx, a.BrowserDownloadURL, dest, func(downloaded, total int64) {
			if total <= 0 {
				return
			}
			pct := int(downloaded * 100 / total)
			if pct >= lastPct+5 || pct == 100 {
				log.Logf(i18n.Tf("install.dlProgress", pct))
				lastPct = pct
			}
		})
		if err != nil {
			return errors.New(i18n.Tf("install.dlFail", a.BrowserDownloadURL, err))
		}
	}
	log.Logf(i18n.T("install.dlDone"))
	return nil
}

// envSingleton 由 main 注入的系统环境信息。
var envSingleton *system.Env

// SetEnv 注入系统环境信息（main 调用）。
func SetEnv(e *system.Env) { envSingleton = e }
