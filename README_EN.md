<p align="center">
  <img src="assets/logo.jpg" alt="Linux-BBR-v3 logo" width="480">
</p>

🌏 **Languages:** [简体中文](README.md) | **English** | [日本語](README_JA.md) | [한국어](README_KO.md)

A BBRv3 kernel installer and network acceleration manager for Debian/Ubuntu VPS.

This project is a **Go + [bubbletea](https://github.com/charmbracelet/bubbletea) TUI rewrite** of [byJoey/Actions-bbr-v3](https://github.com/byJoey/Actions-bbr-v3):

- The original shell `install.sh` is rewritten as a Go program with **identical features and options** (interactive menu, security mitigations, `bbr` quick command).
- The branding `joeyblog` is replaced with `MinimaxFlora` (kernel package names, config paths, uninstall matching, MODULE_DESCRIPTION).
- Kernel `.deb` packages are downloaded from this repository's GitHub Releases.

## Usage

**Install with `curl`:**

> GitHub source (recommended for overseas or proxied environments)

```sh
export url='https://raw.githubusercontent.com/MinimaxFlora/Linux-BBR-v3/master' \
  && sh -c "$(curl -fsSL $url/install.sh)"
```

> Or jsDelivr CDN source (for mainland China networks)

```sh
export url='https://testingcf.jsdelivr.net/gh/MinimaxFlora/Linux-BBR-v3@master' \
  && sh -c "$(curl -fsSL $url/install.sh)"
```

**Install with `wget`:**

> GitHub source

```sh
export url='https://raw.githubusercontent.com/MinimaxFlora/Linux-BBR-v3/master' \
  && wget -q --no-check-certificate -O /tmp/install.sh $url/install.sh \
  && sh /tmp/install.sh
```

> Or jsDelivr CDN source

```sh
export url='https://testingcf.jsdelivr.net/gh/MinimaxFlora/Linux-BBR-v3@master' \
  && wget -q --no-check-certificate -O /tmp/install.sh $url/install.sh \
  && sh /tmp/install.sh
```

> For the binary download in mainland China, pair it with a mirror: `export BBRV3_MIRROR=https://ghfast.top/` (see "China Mainland Network Support").

After the first run, the program automatically installs the `bbr` quick command. From then on you can just run:

```bash
bbr
```

`bbr` executes the locally installed version directly — no network download on every run. To update, use menu item 10 "Check for TUI updates".

When running as root, the program first:

1. Installs the `/usr/local/bin/bbr` quick command (copies the current binary locally, then runs it directly).
2. Writes Dirty Frag risk-surface mitigation rules (`/etc/modprobe.d/99-minimaxflora-security.conf`).

Then the interactive menu starts.

## China Mainland Network Support

The program defaults to **auto mode**: it tries GitHub directly, and on failure silently falls back to mainland China mirror sources in order (`gh-proxy.kejizero.xyz` / `gh-proxy.com` / `ghfast.top`).

- TUI menu 11 "Network source settings" lets you switch between **Auto** (recommended) / GitHub direct only / a fixed mirror; the choice is persisted to `/etc/bbrv3/mirror`.
- The `BBRV3_MIRROR` environment variable takes precedence: `auto`, `direct`, or a mirror URL (e.g. `https://ghfast.top/`).
- `install.sh` supports it too: `BBRV3_MIRROR=https://ghfast.top/ bash <(curl ...)`.
- Covered: install.sh first install, kernel `.deb` downloads, TUI self-update (menu 10), and version-check APIs.

## Supported Environments

| Item | Requirement |
| --- | --- |
| Minimum OS | Ubuntu 24.04+ / Debian 12+ |
| Recommended OS | Ubuntu 24.04+ / Debian 12+ |
| Package manager | `apt-get` |
| Architectures | `x86_64` / `aarch64` |
| Bootloader | GRUB recommended |
| Use case | VPS / cloud server / dedicated server |

Not recommended on devices that depend on U-Boot or vendor-customized kernel chains, such as Raspberry Pi or NanoPi.

On Debian testing/unstable without `VERSION_ID`, the program falls back to `VERSION_CODENAME` and recognizes `bookworm`, `trixie`, `forky` and `sid`. Alpine Linux is not supported yet (release artifacts are `.deb`).

The current kernel mainline for this project is Linux 7.x. When installing a kernel, environments older than the minimum supported OS are blocked to avoid kernel panics. Older systems can still use status check, network tuning, clearing optimizations and uninstall.

## Menu

```text
 1. 🚀 Install or update BBR v3 (latest)
 2. 📚 Install a specific version
 3. 🔍 Check BBR v3 status
 4. ⚡ Enable BBR acceleration mode (FQ / FQ_CODEL / FQ_PIE / CAKE submenu)
 5. 🌏 TCP tuning for Asia-Pacific servers
 6. 🗑️ Uninstall BBR kernel
 7. 🧠 BBR v3 smart bandwidth optimization
 8. 🧹 Clear network optimization settings
 9. 🧨 BBR v3 maniac mode (extreme speedtest challenge)
10. 🔄 Check for TUI updates
11. 🌐 Network source settings (CN/global)
```

Typical workflow:

1. Select `1` to install or update the BBRv3 kernel, then choose Standard or Max edition when prompted.
2. Reboot the system as prompted.
3. Run again and select `3` to check BBRv3 status.
4. Select `4` to enter the acceleration mode submenu and pick FQ / FQ_CODEL / FQ_PIE / CAKE as needed.
5. On Asia-Pacific links, select `5` to apply TCP send/receive window and idle slow-start tuning.
6. If you are unsure about link parameters, select `7` for an automatic speedtest and TCP buffer sizing by bandwidth tier.
7. For extreme speedtest challenges on your own link, select `9` to apply aggressive pacing parameters.
8. To roll back tuning, select `8` to clear the network optimization settings written by the program.
9. If GitHub is unstable from your network, select `11` to switch the network source (auto / mirror).

## Kernel & BBR Policy

```text
The BBRv3 patch is pinned; the kernel automatically follows the latest stable from kernel.org.
```

Current patch selection rules:

```text
linux-7.0.y -> patches/bbrv3-linux-7.0.patch
linux-7.1.y -> patches/bbrv3-linux-7.1.patch
linux-7.2.y -> patches/bbrv3-linux-7.2.patch
```

Kernel packages are built by GitHub Actions and published to Releases. Example package names (Debian naming rules require lowercase; `minimaxflora` is the branding suffix):

```text
linux-headers-7.2.0-minimaxflora-bbrv3_7.2.0-1_amd64.deb        (Standard)
linux-headers-7.2.0-minimaxflora-bbrv3-max_7.2.0-1_amd64.deb    (Max)
```

Release tag format:

```text
x86_64-7.1.8
arm64-7.1.8
x86_64-7.1.8-max
arm64-7.1.8-max
```

The Max edition increases the aggressiveness of the Startup, ProbeBW and cwnd strategies but keeps the BBRv3 loss, ECN, inflight and ProbeBW feedback loop intact. It is only meant for throughput tests on links you control — not recommended for daily production use.

## Check BBRv3 Status

Selecting `3` checks:

- Whether the `tcp_bbr` module version is `3`.
- Whether the current TCP congestion control algorithm is `bbr`.
- Whether the Dirty Frag-related module blacklist is in place.

## Acceleration Mode

Select `4` to enter the acceleration mode submenu:

| Option | Configuration |
| --- | --- |
| 1 | BBR + FQ (recommended, fair queue) |
| 2 | BBR + FQ_CODEL (low latency) |
| 3 | BBR + FQ_PIE (proportional integral) |
| 4 | BBR + CAKE (cake queue) |

The configuration is applied immediately and you are asked whether to persist it to `/etc/sysctl.d/99-minimaxflora.conf`.

The program writes `net.core.default_qdisc` and also swaps the root qdisc of the current default-route egress NIC to the selected algorithm right away. Queue disciplines that require module loading are written to `/etc/modules-load.d/minimaxflora-qdisc.conf`.

## TCP Tuning for Asia-Pacific Servers

Selecting `5` applies and persists:

```text
net.ipv4.tcp_wmem = 4096 16384 12582912
net.ipv4.tcp_rmem = 4096 131072 33554432
net.ipv4.tcp_limit_output_bytes = 4194304
net.ipv4.tcp_slow_start_after_idle = 0
```

## BBR v3 Smart Bandwidth Optimization

Selecting `7`:

- Installs and runs Ookla's official `speedtest 1.2.0` first, automatically tries nearby servers and measures upload/download bandwidth; a Python `speedtest-cli` is removed if detected. If the speedtest fails, you are prompted to enter the upload bandwidth manually. Test node latency is hidden and not used in RTT calculation.
- Automatically enables the `bbr` congestion control and `fq` queue discipline.
- Maps the upload bandwidth and region mode to a recommended TCP buffer tier, capped by the machine's memory.
- RTT is entered manually (the real link latency measured by v2rayN); you pick Asia-Pacific, US/EU, or a manual RTT + buffer tier.

## BBR v3 Maniac Mode

Selecting `9` force-enables `bbr` + `fq` and writes:

```text
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr
net.core.rmem_max = 1073741824
net.core.wmem_max = 1073741824
net.core.optmem_max = 1073741824
net.core.netdev_max_backlog = 1000000
net.core.somaxconn = 65535
net.ipv4.tcp_wmem = 4096 1048576 1073741824
net.ipv4.tcp_rmem = 4096 1048576 1073741824
net.ipv4.tcp_limit_output_bytes = 268435456
net.ipv4.tcp_slow_start_after_idle = 0
net.ipv4.tcp_notsent_lowat = 4294967295
net.ipv4.tcp_autocorking = 0
net.ipv4.tcp_no_metrics_save = 1
net.ipv4.tcp_mtu_probing = 1
net.ipv4.tcp_fastopen = 3
net.ipv4.tcp_window_scaling = 1
net.ipv4.tcp_moderate_rcvbuf = 1
net.ipv4.tcp_ecn = 0
```

It also raises the runtime `txqueuelen` of the current default-route egress NIC to `100000` (not persisted). Failure to apply core parameters aborts; failure of auxiliary parameters does not block.

## Clear Network Optimization Settings

Selecting `8` removes the persistent settings written by this program (default_qdisc, congestion control, the whole TCP buffer series) and deletes `/etc/modules-load.d/minimaxflora-qdisc.conf`. Only network optimization settings are cleared: the BBR kernel is not uninstalled and the Dirty Frag mitigation rules are kept. Runtime parameters may need a reboot to fully return to defaults.

## Security Mitigations

At startup, `/etc/modprobe.d/99-minimaxflora-security.conf` is written:

- `esp4` / `esp6` / `rxrpc` blacklist (`blacklist` + `install /bin/false`) to shrink the Dirty Frag attack surface. Already-loaded modules are unloaded if possible; otherwise the blacklist takes effect after a reboot.

The AEAD userspace interface related to CVE-2026-31431 is closed at the kernel config level in newly built kernels (`# CONFIG_CRYPTO_USER_API_AEAD is not set`), so no additional `algif_aead` blacklist entry is written; if an older version wrote one, it is removed automatically once the running kernel is confirmed to have it disabled.

## Uninstall

Selecting `6` uninstalls the `MinimaxFlora` kernel packages installed by this project and refreshes the boot configuration. A reboot after uninstalling is recommended.

## Build

Build locally (cross-compilation works on Windows / macOS / Linux):

```bash
# Linux amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bbrv3-linux-amd64 .
# Linux arm64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bbrv3-linux-arm64 .
```

Development preview (skips system checks so you can view the TUI on non-Linux machines):

```bash
BBRV3_DEV=1 go run .
```

Tests:

```bash
go test ./...
```

The `release-cli.yml` workflow (independent of kernel builds) builds and publishes the binary to the `bbrv3-cli` release on every push to master or manual trigger; TUI menu 10 "Check for TUI updates" pulls the latest version from it and replaces the local `bbr`. `build.yml` only handles kernel builds and releases.

## Directory Structure

```text
install.sh                      # One-shot script: detect arch → download latest binary → run
main.go                         # Entry point (auto re-executes via sudo when not root)
internal/
├── bbr/                       # Pure logic: version compare / tag parsing / buffer sizing (unit-testable)
├── execx/                     # Command runner: streams output to the TUI log
├── system/                    # sysctl / qdisc / security mitigations / network tuning / kernel install / quick command
├── netutil/                   # GitHub release API + Ookla speedtest
└── app/                       # bubbletea TUI: main menu + subpages (forms / progress log / results)
scripts/                       # Kernel build scripts (for CI, matching upstream, branding applied)
patches/                       # BBRv3 patches (pinned)
configs/x86-64.config / arm64.config   # Kernel config baselines
.github/RELEASE_NOTES_cli.md   # bbrv3-cli release notes (auto-refreshed on every push)
.github/workflows/build.yml      # Kernel auto-build + release (schedule / manual trigger)
.github/workflows/release-cli.yml # Go TUI binary published to bbrv3-cli release (push master / manual trigger)
```

## Disclaimer

Kernel upgrades involve risk. Before installing, make sure your VPS console, rescue mode or an old-kernel boot entry is available. Any boot failure, network anomaly or data loss caused by kernels built or installed with this project is at the user's own risk.
