# Tailcat-QUIC

[English](README.md) · [简体中文](README.zh-CN.md) · **日本語**

接続アドレスを共有するだけで、2 台のマシン間でファイル転送、ポート転送、SSH 接続ができます。実行にアカウント、VPN 設定、管理者権限は不要です。直接接続とリレーへの切り替えは自動です。

[Tailcat](https://github.com/tailscale/tailcat) の独立フォークで、認証付き QUIC/HTTP/3 と BBRv3 を使用します。コマンド名は `tailcat` のままで、普段の操作は公式版と同じです。**両端に Tailcat-QUIC が必要です**。本版の `tch3…` アドレスと公式版の `tc…` アドレスには互換性がありません。

[最新リリース: v0.7.0-quic.4](https://github.com/LiuTangLei/tailcat-quic/releases/tag/v0.7.0-quic.4)

## インストール

Linux / macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.ps1 | iex
```

インストーラは対象プラットフォームを選び、ダウンロードを検証します。アーカイブ、DEB/RPM、Homebrew、Scoop、Nix、コンテナは[インストールガイド](INSTALL.md)を参照してください。

## クイックスタート

### テキストやファイルを送る

受信側:

```sh
tailcat > received.txt
```

表示された完全な `tch3…` アドレスを送信側へ渡します:

```sh
echo hello | tailcat 'tch3…'
# ファイルを送る場合:
tailcat 'tch3…' < document.pdf
```

次の転送では受信側を再起動します。鍵は自動生成されるため、事前の設定は不要です。アドレスはサービスへのアクセスを許可する情報なので、相手にだけ共有してください。

### ローカルサービスを転送する

サービスが動いているマシン:

```sh
tailcat serve 8080
```

接続するマシン:

```sh
tailcat forward 'tch3…' 18080:8080
```

`http://127.0.0.1:18080` を開きます。Web サービスなら `tailcat browse 'tch3…'` で直接開くこともできます。

### ファイル名を指定してコピーする

```sh
# 受信側
tailcat recv

# 送信側
tailcat cp document.pdf 'tch3…:'
```

受信したファイルは現在のディレクトリに一意の名前で保存されます。既存ファイルを共有する場合は `tailcat serve --files=./shared files` を実行します。デフォルトは読み取り専用です。相手は `tailcat ls 'tch3…:'` で一覧を表示し、`tailcat cp 'tch3…:document.pdf' .` で取得できます。

### SSH

既存の SSH 公開鍵を使って許可する場合:

```sh
tailcat serve --ssh-authorized-keys ~/.ssh/authorized_keys ssh
```

相手は `tailcat ssh 'tch3…'` で接続します。既存の SSH サーバーは `tailcat serve 22` で公開できます。

## その他の操作

| 操作 | コマンド |
| --- | --- |
| 接続経路を確認 | `tailcat ping 'tch3…'` |
| 複数ポートを共有 | `tailcat serve 8080,8443` |
| 接続ごとにコマンドを実行 | `tailcat serve exec -- /path/to/program arg1` |
| SOCKS 経由でアクセス | `tailcat socks 'tch3…' curl http://server.tailcat:8080/` |
| 再起動後も同じアドレスを使う | `tailcat genkey --key=default` |
| ヘルプ | `tailcat --help` または `tailcat <command> --help` |

鍵の保存は任意です。`default` というサーバー鍵を保存すると次回から再利用されます。新しいアドレスには `--key=new` を使います。アクセス制限は[セキュリティ](SECURITY.md)を参照してください。

## 性能と互換性

TCP は信頼性のある QUIC ストリーム、UDP は QUIC DATAGRAM を使用します。暗号化、ノード認証、接続シークレットは自動設定されます。公共リレーには帯域制限がある場合があり、速度は経路とマシンに依存します。

9 月 27 日の公式 Tailcat userspace WireGuard との比較では、日本ノード間の一方向で QUIC の 4 ストリームが **556 / 506 Mbps** でした。単一ストリームや他の経路はこれより遅く、すべての環境で 500 Mbps を保証するものではありません。[全測定結果](docs/wg-quic4-acceptance-20260927.md)。

## 実験的機能

Browser/WebAssembly デモとブラウザ風 TLS フィンガープリントは[実験的機能](docs/experimental.md)にまとめています。通常の CLI 利用に追加設定は不要です。ブラウザ通信は現在リレー経由です。

## プロジェクト

[ソースビルド](INSTALL.md#build-from-source) · [変更履歴](CHANGELOG.md) · [セキュリティ](SECURITY.md) · [開発とリリース](RELEASING.md)

Tailscale の公式サポート対象ではない独立フォークです。[BSD-3-Clause](LICENSE) · [依存ライブラリの表示](THIRD_PARTY_NOTICES.md)。
