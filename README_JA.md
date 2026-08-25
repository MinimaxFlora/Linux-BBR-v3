# Linux-BBR-v3

🌏 **言語:** [简体中文](README.md) | [English](README_EN.md) | **日本語** | [한국어](README_KO.md)

Debian/Ubuntu VPS 向けの BBRv3 カーネル導入 & ネットワーク加速管理プログラムです。

本プロジェクトは [byJoey/Actions-bbr-v3](https://github.com/byJoey/Actions-bbr-v3) の **Go + [bubbletea](https://github.com/charmbracelet/bubbletea) TUI リライト版**です：

- 元の shell 版 `install.sh` を Go プログラムとして書き直し、**機能・選択肢は完全一致**（対話メニュー、セキュリティ緩和、`bbr` ショートカットコマンド）。
- ブランド識別子を `joeyblog` から `MinimaxFlora` に置き換え（カーネルパッケージ名、設定パス、アンインストール判定、MODULE_DESCRIPTION）。
- カーネル `.deb` パッケージは本リポジトリの GitHub Releases からダウンロードします。

## 使い方

```bash
# ワンライナーでインストール & 実行（アーキテクチャを自動判定し、最新バイナリをダウンロードして起動）
bash <(curl -fsSL https://raw.githubusercontent.com/MinimaxFlora/Linux-BBR-v3/master/install.sh)
```

初回実行時に `bbr` ショートカットコマンドが自動インストールされます。以降は直接実行できます：

```bash
bbr
```

`bbr` はローカルにインストール済みのバージョンを直接実行し、毎回ネットワークからダウンロードしません。更新する場合は TUI メニュー 10「TUI の更新確認」を使います。

root で実行すると、まず以下を行います：

1. `/usr/local/bin/bbr` ショートカットコマンドをインストール（現在のバイナリをローカルにコピーして直接実行）。
2. Dirty Frag リスク面収束ルールを書き込み（`/etc/modprobe.d/99-minimaxflora-security.conf`）。

その後、対話メニューに入ります。

## 対応環境

| 項目 | 要件 |
| --- | --- |
| 最低対応 OS | Ubuntu 24.04+ / Debian 12+ |
| 推奨 OS | Ubuntu 24.04+ / Debian 12+ |
| パッケージマネージャ | `apt-get` |
| アーキテクチャ | `x86_64` / `aarch64` |
| ブートローダ | GRUB 推奨 |
| 利用シーン | VPS / クラウドサーバー / 専用サーバー |

Raspberry Pi や NanoPi など、U-Boot やベンダー製カスタムカーネルチェーンに依存するデバイスでの使用は推奨しません。

Debian testing/unstable で `VERSION_ID` がない場合、`VERSION_CODENAME` から `bookworm`・`trixie`・`forky`・`sid` を判定します。Alpine Linux は現在未対応です（リリース成果物が `.deb` のため）。

本プロジェクトの現在のカーネルメインラインは Linux 7.x です。カーネル導入時は最低対応 OS より古い環境をブロックし、kernel panic を回避します。古い OS でも状態確認・ネットワークチューニング・最適化のクリア・アンインストールは利用できます。

## メニュー

```text
 1. 🚀 BBR v3 のインストールまたは更新（最新版）
 2. 📚 指定バージョンのインストール
 3. 🔍 BBR v3 の状態確認
 4. ⚡ BBR 加速モードを有効化（FQ / FQ_CODEL / FQ_PIE / CAKE サブメニュー）
 5. 🌏 アジア太平洋向け TCP チューニング
 6. 🗑️ BBR カーネルのアンインストール
 7. 🧠 BBR v3 スマート帯域幅最適化
 8. 🧹 ネットワーク最適化設定のクリア
 9. 🧨 BBR v3 マニアックモード（極限スピードテストチャレンジ）
10. 🔄 TUI の更新確認
```

よく使う流れ：

1. `1` を選択して BBRv3 カーネルをインストールまたは更新。プロンプトに従って標準版または Max 極限版を選択します。
2. 案内に従ってシステムを再起動します。
3. 再度実行し、`3` を選択して BBRv3 の状態を確認します。
4. `4` で加速モードのサブメニューに入り、FQ / FQ_CODEL / FQ_PIE / CAKE から選択します。
5. アジア太平洋回線のマシンは `5` で TCP 送受信ウィンドウとアイドル時スロー スタートのチューニングを書き込みます。
6. 回線パラメータが不明な場合は `7` で自動スピードテストを行い、帯域幅に応じた TCP バッファを計算します。
7. 自社回線の極限スピードテストチャレンジには `9` でアグレッシブな送信レートパラメータを書き込みます。
8. チューニングを取り消す場合は `8` でプログラムが書き込んだネットワーク最適化設定をクリアします。

## カーネル & BBR ポリシー

```text
BBRv3 パッチは固定。カーネルは kernel.org の最新 stable に自動追従します。
```

現在のパッチ選択ルール：

```text
linux-7.0.y -> patches/bbrv3-linux-7.0.patch
linux-7.1.y -> patches/bbrv3-linux-7.1.patch
linux-7.2.y -> patches/bbrv3-linux-7.2.patch
```

カーネルパッケージは GitHub Actions でビルドされ Releases に公開されます。パッケージ名の例（Debian の命名規則上すべて小文字、`minimaxflora` はブランド接尾辞）：

```text
linux-headers-7.2.0-minimaxflora-bbrv3_7.2.0-1_amd64.deb        （標準版）
linux-headers-7.2.0-minimaxflora-bbrv3-max_7.2.0-1_amd64.deb    （Max 版）
```

release tag の形式：

```text
x86_64-7.1.8
arm64-7.1.8
x86_64-7.1.8-max
arm64-7.1.8-max
```

Max 版は Startup・ProbeBW・cwnd の戦略をより攻撃的にしますが、BBRv3 の loss・ECN・inflight・ProbeBW フィードバックループは維持されます。自社回線のスループットテスト専用で、日常の本番利用には推奨しません。

## BBRv3 の状態確認

`3` を選択すると以下を確認します：

- `tcp_bbr` モジュールのバージョンが `3` かどうか。
- 現在の TCP 輻輳制御アルゴリズムが `bbr` かどうか。
- Dirty Frag 関連モジュールのブラックリストが書き込まれているかどうか。

## 加速モード

`4` を選択すると加速モードのサブメニューに入ります：

| オプション | 構成 |
| --- | --- |
| 1 | BBR + FQ（推奨、公平キュー） |
| 2 | BBR + FQ_CODEL（低遅延） |
| 3 | BBR + FQ_PIE（比例積分） |
| 4 | BBR + CAKE（ケーキキュー） |

選択後すぐに適用を試み、`/etc/sysctl.d/99-minimaxflora.conf` への恒久書き込みを確認します。

プログラムは `net.core.default_qdisc` を書き込むだけでなく、現在のデフォルトルート出口 NIC の root qdisc を即座に選択したアルゴリズムへ置き換えます。モジュールのロードが必要なキューアルゴリズムは `/etc/modules-load.d/minimaxflora-qdisc.conf` に書き込まれます。

## アジア太平洋向け TCP チューニング

`5` を選択すると即座に適用され、恒久的に書き込まれます：

```text
net.ipv4.tcp_wmem = 4096 16384 12582912
net.ipv4.tcp_rmem = 4096 131072 33554432
net.ipv4.tcp_limit_output_bytes = 4194304
net.ipv4.tcp_slow_start_after_idle = 0
```

## BBR v3 スマート帯域幅最適化

`7` を選択すると：

- まず Ookla 公式 `speedtest 1.2.0` をインストールして実行し、近隣のテストサーバーを自動選択して上り/下り帯域幅を取得します。Python 版 `speedtest-cli` を検出した場合は先に削除します。テスト失敗時は上り帯域幅の手動入力を促します。テストノードの遅延は隠され、RTT 計算には使いません。
- `bbr` 輻輳制御と `fq` キューアルゴリズムを自動で有効化します。
- 上り帯域幅と地域モードから推奨 TCP バッファ段階をマッピングし、マシンのメモリに応じて上限を設定します。
- RTT はユーザーが手動入力します（v2rayN で測定した実際のリンク遅延）。アジア太平洋・米欧の自動選択、または手動 RTT + バッファ段階を選択できます。

## BBR v3 マニアックモード

`9` を選択すると `bbr` + `fq` を強制有効化し、以下を書き込みます：

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

同時に、現在のデフォルトルート出口 NIC の実行時 `txqueuelen` を `100000` に引き上げます（恒久設定には書き込みません）。コアパラメータの適用失敗は中断、付加パラメータの失敗はブロックしません。

## ネットワーク最適化設定のクリア

`8` を選択すると、本プログラムが書き込んだ恒久設定（default_qdisc、輻輳制御、TCP バッファ全般）を削除し、`/etc/modules-load.d/minimaxflora-qdisc.conf` を削除します。ネットワーク最適化設定のみクリアし、BBR カーネルのアンインストールや Dirty Frag 緩和ルールの削除は行いません。実行時パラメータは再起動後に完全にデフォルトへ戻る場合があります。

## セキュリティ緩和

起動時に `/etc/modprobe.d/99-minimaxflora-security.conf` を書き込みます：

- `esp4` / `esp6` / `rxrpc` のブラックリスト（`blacklist` + `install /bin/false`）で Dirty Frag のリスク面を収束。ロード済みモジュールは可能ならアンロードします。使用中の場合は再起動後に有効になります。

CVE-2026-31431 に対応する AEAD userspace インターフェースは、新規ビルドカーネルではカーネル設定側で収束済み（`# CONFIG_CRYPTO_USER_API_AEAD is not set`）のため、追加の `algif_aead` ブラックリストは書き込みません。旧バージョンで書き込まれている場合、実行中カーネルで無効化を確認後に自動削除します。

## アンインストール

`6` を選択すると、本プロジェクトがインストールした `MinimaxFlora` カーネルパッケージをアンインストールし、ブート設定を更新します。アンインストール後の再起動を推奨します。

## ビルド

ローカルビルド（Windows / macOS / Linux いずれでもクロスコンパイル可能）：

```bash
# Linux amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bbrv3-linux-amd64 .
# Linux arm64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bbrv3-linux-arm64 .
```

開発プレビュー（システムチェックをスキップし、非 Linux マシンでも TUI を確認可能）：

```bash
BBRV3_DEV=1 go run .
```

テスト：

```bash
go test ./...
```

GitHub Actions の `release-cli.yml` workflow（カーネルビルドとは独立）は、master への push または手動トリガー時にバイナリをビルドして `bbrv3-cli` release に公開します。TUI メニュー 10「TUI の更新確認」がそこから最新版を取得してローカルの `bbr` を置き換えます。`build.yml` はカーネルのビルドと公開のみを担当します。

## ディレクトリ構成

```text
install.sh                      # ワンライナースクリプト：アーキテクチャ判定 → 最新バイナリのダウンロード → 実行
main.go                         # エントリーポイント（非 root 時は sudo で自動再実行）
internal/
├── bbr/                       # 純粋ロジック：バージョン比較 / tag 解析 / バッファ計算（単体テスト可能）
├── execx/                     # コマンド実行器：出力をストリームで TUI ログへ転送
├── system/                    # sysctl / qdisc / セキュリティ緩和 / ネットワークチューニング / カーネル導入 / ショートカットコマンド
├── netutil/                   # GitHub release API + Ookla speedtest
└── app/                       # bubbletea TUI：メインメニュー + サブページ（フォーム / 進捗ログ / 結果）
scripts/                       # カーネルビルドスクリプト（CI 用、上流と同一、パッケージ名はブランド化済み）
patches/                       # BBRv3 パッチ（固定）
x86-64.config / arm64.config   # カーネル設定ベースライン
.github/RELEASE_NOTES_cli.md   # bbrv3-cli release の説明（push のたびに自動更新）
.github/workflows/build.yml      # カーネル自動ビルド + 公開（schedule / 手動トリガー）
.github/workflows/release-cli.yml # Go TUI バイナリを bbrv3-cli release に公開（push master / 手動トリガー）
```

## 免責事項

カーネルアップグレードにはリスクが伴います。導入前に VPS コンソール、リカバリーモード、または旧カーネルの起動項目が利用可能であることを確認してください。本プロジェクトでビルドまたは導入したカーネルによる起動失敗、ネットワーク異常、データ損失は、利用者の責任で負担いただきます。
