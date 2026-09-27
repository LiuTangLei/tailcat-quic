# Tailcat-QUIC

[English](README.md) · **简体中文** · [日本語](README.ja.md)

用一个连接地址，在两台机器之间传文件、转发端口或连接 SSH。运行时无需账号、VPN 配置或管理员权限，直连和中继回退自动完成。

这是 [Tailcat](https://github.com/tailscale/tailcat) 的独立分支，使用经过认证的 QUIC/HTTP/3 和 BBRv3，命令仍叫 `tailcat`，日常用法与官方版一致。**两端都需要安装 Tailcat-QUIC**：本版的 `tch3…` 地址与官方版的 `tc…` 地址不兼容。

[最新版本：v0.7.0-quic.4](https://github.com/LiuTangLei/tailcat-quic/releases/tag/v0.7.0-quic.4)

## 安装

Linux / macOS：

```sh
curl -fsSL https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.sh | sh
```

Windows PowerShell：

```powershell
irm https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.ps1 | iex
```

安装器会自动选择平台并校验下载。压缩包、DEB/RPM、Homebrew、Scoop、Nix 和容器的用法见[安装说明](INSTALL.md)。

## 快速使用

### 发送文字或文件

接收端运行：

```sh
tailcat > received.txt
```

屏幕上会显示一个 `tch3…` 地址。把完整地址复制到发送端：

```sh
echo hello | tailcat 'tch3…'
# 也可以发送文件：
tailcat 'tch3…' < document.pdf
```

下次传输重新启动接收端即可。密钥自动生成，无需手动配置。请私下分享连接地址，持有者可以访问你正在提供的服务。

### 转发本地服务

服务所在的机器运行：

```sh
tailcat serve 8080
```

另一台机器运行：

```sh
tailcat forward 'tch3…' 18080:8080
```

然后访问 `http://127.0.0.1:18080`。如果是网页服务，也可以直接运行 `tailcat browse 'tch3…'` 打开。

### 按文件名传输

```sh
# 接收端
tailcat recv

# 发送端
tailcat cp document.pdf 'tch3…:'
```

接收端默认把上传文件保存到当前目录，并生成唯一文件名。需要让对方下载已有文件时，运行 `tailcat serve --files=./shared files`，默认只读。对方可以用 `tailcat ls 'tch3…:'` 查看，用 `tailcat cp 'tch3…:document.pdf' .` 下载。

### SSH

使用已有的 SSH 授权密钥启动服务：

```sh
tailcat serve --ssh-authorized-keys ~/.ssh/authorized_keys ssh
```

另一端运行 `tailcat ssh 'tch3…'`。如果机器已经运行 SSH 服务，也可以直接用 `tailcat serve 22` 转发。

## 更多用法

| 操作 | 命令 |
| --- | --- |
| 检查连接路径 | `tailcat ping 'tch3…'` |
| 共享多个端口 | `tailcat serve 8080,8443` |
| 每次连接运行一个程序 | `tailcat serve exec -- /path/to/program arg1` |
| 通过 SOCKS 代理访问 | `tailcat socks 'tch3…' curl http://server.tailcat:8080/` |
| 重启后保留服务器地址 | `tailcat genkey --key=default` |
| 查看参数 | `tailcat --help` 或 `tailcat <command> --help` |

保存密钥是可选操作。保存名为 `default` 的服务器密钥后，后续启动会复用它；使用 `--key=new` 可生成新地址。客户端白名单和访问控制见[安全说明](SECURITY.md)。

## 性能与兼容性

TCP 服务使用可靠 QUIC 流，UDP 使用 QUIC DATAGRAM。加密、节点认证和连接秘密自动配置。公共中继可能限速，实际速度取决于线路和机器。

9 月 27 日与官方 Tailcat 用户态 WireGuard 的对比中，日本节点一个方向的 QUIC 四流两轮测得 **556 / 506 Mbps**。单流及其他线路更慢，并非所有场景都能达到 500 Mbps。[完整测速和验收范围](docs/wg-quic4-acceptance-20260927.md)。

## 实验性功能

浏览器/WebAssembly 演示和浏览器风格 TLS 指纹的说明放在[实验性功能](docs/experimental.md)。正常使用命令行无需配置这些功能。浏览器目前通过中继传输。

## 项目

[源码构建](INSTALL.md#build-from-source) · [更新记录](CHANGELOG.md) · [安全说明](SECURITY.md) · [贡献与发布](RELEASING.md)

本项目是独立分支，不由 Tailscale 官方支持。[BSD-3-Clause 许可证](LICENSE) · [第三方声明](THIRD_PARTY_NOTICES.md)。
