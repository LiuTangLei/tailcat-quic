# tailcat-quic

[English](README.md) · [简体中文](README.zh-CN.md) · **日本語**

[Tailscale Tailcat v0.7.0](https://github.com/tailscale/tailcat/tree/v0.7.0) をベースにした、独立した QUIC/HTTP/3-only フォークです。

Tailcat の「アカウント不要・コントロールプレーン不要」の P2P モデルを保ちながら、WireGuard データプレーンを認証済み HTTP/3 + QUIC に置き換えています。TCP サービスは信頼性のある HTTP/3 ストリーム、UDP/IP は QUIC DATAGRAM、QUIC セッションは TLS 1.3、輻輳制御はデフォルトで userspace BBRv3 を使います。

**最新リリース: [v0.7.0-quic.2](https://github.com/LiuTangLei/tailcat-quic/releases/tag/v0.7.0-quic.2)**

> 両端ともこのフォークを使用してください。`tch3…` アドレスは、公式 Tailcat の WireGuard `tc…` アドレスとは互換性がありません。

## インストール

### 最短の方法

Linux / macOS:

~~~sh
curl -fsSL https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.sh | sh
~~~

Windows PowerShell:

~~~powershell
irm https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.ps1 | iex
~~~

インストーラは GitHub Release から対象アーカイブを取得し、`checksums.txt` の SHA-256 と `tailcat version` を検証します。サービス起動、ファイアウォール、ルーティング設定の変更は行いません。

### パッケージマネージャ

| 方法 | コマンド |
| --- | --- |
| Homebrew | `brew tap LiuTangLei/tailcat-quic https://github.com/LiuTangLei/tailcat-quic.git && brew install LiuTangLei/tailcat-quic/tailcat-quic` |
| Scoop | `scoop install https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/bucket/tailcat-quic.json` |
| Nix | `nix profile install github:LiuTangLei/tailcat-quic/quic-v0.7#tailcat-quic` |
| Go | `go run github.com/LiuTangLei/tailcat-quic/install@v0.7.0-quic.2` |
| Container | `docker run --rm -i ghcr.io/liutanglei/tailcat-quic:latest` |

Linux 向けには amd64/arm64/armv7 の tar.gz・deb・rpm、Windows 向けには amd64/arm64 zip、さらに macOS amd64/arm64 tar.gz を提供します。

このフォークでは、外部パッケージレジストリを一つずつ複製すること自体はサポート目標にしません。上流の prebuilt 対応プラットフォームをすべて維持し、主要なデスクトップ／サーバープラットフォームには検証済みのワンラインインストーラまたは Release パッケージを提供します。詳細は [INSTALL.md](INSTALL.md) を参照してください。

## クイックスタート

サーバー側:

~~~sh
tailcat serve 8080
~~~

信頼できる経路で完全な `tch3…` アドレスを相手に渡し、クライアント側で:

~~~sh
tailcat forward 'tch3…' 18080:8080
# http://127.0.0.1:18080 を開く
~~~

SSH:

~~~sh
tailcat serve --ssh-authorized-keys ~/.ssh/authorized_keys ssh
tailcat ssh 'tch3…'
~~~

## 対応プラットフォーム

ダウンロード可能なバイナリは公式の prebuilt 対象をすべて含み、macOS も追加しています。

- Linux: amd64 / arm64 / armv7
- Windows: amd64 / arm64
- macOS: amd64 / arm64

CLI は FreeBSD/OpenBSD の amd64/arm64 でもクロスビルドできます。Browser/WebAssembly バンドルもソースからビルド可能ですが、ブラウザは relay-only であるため、実ブラウザでの統合確認は別途記録します。

## 性能

`v0.7.0-quic.2` は、上限付き 32 KiB H3 読み取りバッファを再利用し、すでに到着済みのデータだけをまとめて読み取ります。大きなバッチを作るための待ち時間は追加しません。

AU / US1420 の A/B では、4 ストリーム平均が一方向で **333.41 → 368.47 Mbps**、逆方向で **211.56 → 305.86 Mbps**。最終 Linux リリースバイナリでは 4 ストリーム双方向 **327.43 / 321.95 Mbps** を確認しました。限られた WAN サンプルであり、すべてのネットワークで同じ改善を保証するものではありません。

詳細は [validation report](docs/release-validation-v0.7.0-quic.2.md) を参照してください。

## セキュリティ

HTTP/3 は実際のプロトコルフレーミングですが、「すべてのブラウザ通信と完全に識別不能」という保証ではありません。

- TLS 検証、接続シークレット、ノード認証を維持します。
- `--psk=false` は拒否されます。
- TCP ストリームは認証済み QUIC セッション上でのみ開かれます。
- UDP/IP は認証済み CONNECT-IP / QUIC DATAGRAM を使用します。
- WireGuard/AWG データプレーンへのフォールバックはありません。

`tch3…` アドレスは資格情報として扱ってください。詳細は [SECURITY.md](SECURITY.md)。

## ソースからビルド

Go 1.27.1 が必要です。

~~~sh
git clone https://github.com/LiuTangLei/tailcat-quic.git
cd tailcat-quic
git checkout quic-v0.7
go build -trimpath -tags "$(cat build-tags.txt)" -o tailcat ./cmd/tailcat
~~~

このプロジェクトは独立フォークであり、Tailscale の公式サポート対象ではありません。
