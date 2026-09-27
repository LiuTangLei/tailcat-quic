# Experimental features

[English](#english) · [简体中文](#简体中文) · [日本語](#日本語)

## English

Normal CLI commands work with their defaults. These features do not require an extra setup step for file transfers, port forwarding or SSH.

### Browser / WebAssembly

The `web` package is a browser demo for file and text transfers using the same authenticated QUIC fork. Browser traffic currently travels through DERP/WebSocket relays, without direct UDP. Use a demo built from this fork; the official WireGuard browser demo cannot connect to `tch3…` addresses.

Developers can build the WASM module with `GOOS=js GOARCH=wasm go build -o main.wasm ./web`. A WASM module alone is not a hosted demo: the page also needs the matching JavaScript and Go WASM runtime. See the real-browser tests in `web` for the development harness.

### Browser-inspired TLS fingerprint

The shared QUIC implementation includes a Chromium-inspired `chromium-h3` ClientHello profile. The authenticated transport selects it automatically for eligible outgoing client connections; it is not a user-facing CLI switch. Other connection roles can use the standard TLS profile.

This is experimental protocol behavior, not a claim to reproduce a particular browser version or hide every traffic characteristic. It does not disable certificate/node verification or the mandatory connection secret. There is no need to choose a fingerprint, provide a browser identity or alter encryption settings for normal use. See [Security](../SECURITY.md).

## 简体中文

正常的文件传输、端口转发和 SSH 使用默认命令即可，无需额外设置。

### 浏览器 / WebAssembly

`web` 是本分支的浏览器文件、文字传输演示，使用同一套经过认证的 QUIC 传输。目前浏览器通过 DERP/WebSocket 中继连接，不进行 UDP 直连。需要使用本分支构建的页面；官方 WireGuard 浏览器演示不能连接 `tch3…` 地址。

开发者可以运行 `GOOS=js GOARCH=wasm go build -o main.wasm ./web` 构建模块。单个 WASM 文件并不是可直接访问的网站，还需要匹配的页面 JavaScript 和 Go WASM 运行时；开发测试方法可参考 `web` 中的真实浏览器测试。

### 浏览器风格 TLS 指纹

共用 QUIC 实现包含参考 Chromium 的 `chromium-h3` ClientHello。在满足条件的客户端出站连接中，传输层会自动选择它；目前没有需要用户设置的 CLI 指纹开关，其他连接角色可能使用标准 TLS 配置。

这属于实验性协议行为，不保证复现某个浏览器版本，也不保证所有流量特征都无法识别。它不会关闭证书与节点验证，也不会取消必需的连接秘密。日常使用无需选择指纹、提供浏览器身份或修改加密设置。详见[安全说明](../SECURITY.md)。

## 日本語

ファイル転送、ポート転送、SSH はデフォルトのコマンドで利用でき、追加設定は不要です。

### Browser / WebAssembly

`web` は認証付き QUIC を使うファイル・テキスト転送デモです。ブラウザは現在 DERP/WebSocket リレー経由で接続し、直接 UDP は使用しません。本フォークからビルドしたページを使ってください。公式の WireGuard デモは `tch3…` に接続できません。

開発用 WASM は `GOOS=js GOARCH=wasm go build -o main.wasm ./web` でビルドできます。ページには対応する JavaScript と Go WASM ランタイムも必要です。開発手順は `web` の実ブラウザテストを参照してください。

### ブラウザ風 TLS フィンガープリント

共有 QUIC 実装には Chromium を参考にした `chromium-h3` ClientHello があります。条件を満たすクライアントの発信接続で自動選択され、ユーザーが設定する CLI スイッチではありません。他の接続では標準 TLS プロファイルを使う場合があります。

特定のブラウザの完全再現や通信の識別不能性を保証するものではありません。証明書・ノード認証と必須の接続シークレットは維持されます。通常利用にフィンガープリント設定や暗号設定の変更は不要です。[セキュリティ](../SECURITY.md)も参照してください。
