# tailcat-quic

[English](README.md) · **简体中文** · [日本語](README.ja.md)

这是基于 [Tailscale 官方 Tailcat v0.7.0](https://github.com/tailscale/tailcat/tree/v0.7.0) 的独立 QUIC/HTTP/3-only 分支。

Tailcat-QUIC 保留 Tailcat 无账号、无控制服务器的点对点使用方式，但把 WireGuard 数据面替换为经过认证的 HTTP/3 + QUIC：TCP 服务走可靠 HTTP/3 流，UDP/IP 走 QUIC DATAGRAM，QUIC 会话使用 TLS 1.3，默认拥塞控制为用户态 BBRv3。

**当前版本：[v0.7.0-quic.2](https://github.com/LiuTangLei/tailcat-quic/releases/tag/v0.7.0-quic.2)**

此分支正在准备 **v0.7.0-quic.3**，已固定公共 QUIC 0.63 恢复修复并包含经过认证的浏览器传输。最终真实节点验证尚缺，版本保留为发布草稿；下方稳定安装命令仍选择已发布版本。详见[候选验证记录](docs/release-validation-v0.7.0-quic.3.md)。

> 两端都必须使用本分支。`tch3…` 地址与官方 Tailcat 的 WireGuard `tc…` 地址不兼容。

## 为什么做这个分支

- 数据面是真实 QUIC + HTTP/3，而不是 WG/AWG。
- TLS 1.3 + 连接秘密 + 节点认证。
- TCP 使用可靠 HTTP/3 流，UDP 使用 QUIC DATAGRAM。
- 默认用户态 BBRv3。
- 能直连时使用点对点 UDP NAT 穿透，否则回退 DERP。
- 合并 Tailcat 0.7 的 TCP/UDP Listener、退出节点 UDP 转发、browse、exec/SSH 强制命令、SFTP 兼容、Windows localhost 修复、Android/Termux 运行支持与 Peer 路径状态。
- 对 H3 流增加有界缓冲区复用和“只合并已经就绪的数据”，减少内存分配和小块交接，不增加凑包等待。

## 安装

### 最简单的一键安装

Linux / macOS：

~~~sh
curl -fsSL https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.sh | sh
~~~

Windows PowerShell：

~~~powershell
irm https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.ps1 | iex
~~~

两个安装器都会下载对应 GitHub Release 包、按 `checksums.txt` 校验 SHA-256、再检查 `tailcat version`。**不会**自动启动服务，也不会修改防火墙、路由或现有 Tailcat 身份。

### 包管理器 / 一键命令

| 方式 | 命令 | 状态 |
| --- | --- | --- |
| Release 压缩包 | [GitHub Releases](https://github.com/LiuTangLei/tailcat-quic/releases) | Linux amd64/arm64/armv7、macOS amd64/arm64、Windows amd64/arm64 |
| Debian / Ubuntu | 从 Release 下载对应 `.deb` | amd64/arm64/armv7 |
| Fedora / RHEL | 从 Release 下载对应 `.rpm` | amd64/arm64/armv7 |
| Homebrew | `brew tap LiuTangLei/tailcat-quic https://github.com/LiuTangLei/tailcat-quic.git && brew install LiuTangLei/tailcat-quic/tailcat-quic` | 仓库内 Formula |
| Scoop | `scoop install https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/bucket/tailcat-quic.json` | 直接 manifest |
| Nix | `nix profile install github:LiuTangLei/tailcat-quic/quic-v0.7#tailcat-quic` | 仓库 flake |
| Go 工具链 | `go run github.com/LiuTangLei/tailcat-quic/install@v0.7.0-quic.2` | 校验源代码后构建安装 |
| Container | `docker run --rm -i ghcr.io/liutanglei/tailcat-quic:latest` | Linux amd64/arm64 GHCR 镜像 |

使用容器转发 stdin/stdout 时不要加 `-t`；PTY 会把 stderr 状态输出和 stdout 隧道数据混到同一字节流。

本分支不追求逐个复制第三方软件仓库渠道，目标是平台覆盖不能少于官方：官方预编译支持的平台全部保留，并保证常用桌面/服务器平台都有经过校验的一键脚本或 Release 安装包。完整矩阵见 [INSTALL.md](INSTALL.md)。

### 平台覆盖

下载包覆盖官方所有预编译平台，并额外提供 macOS：

- Linux：amd64、arm64、armv7 —— tar.gz / deb / rpm。
- Windows：amd64、arm64 —— zip。
- macOS：amd64、arm64 —— tar.gz。

CLI 也已通过 FreeBSD/OpenBSD 的 amd64、arm64 交叉编译。浏览器/WebAssembly 包可以从源码构建；浏览器为 relay-only 场景，运行时互通仍单独记录真实浏览器集成测试结果。

## 快速使用

服务端共享本地端口：

~~~sh
tailcat serve 8080
~~~

把完整的 `tch3…` 地址通过可信渠道发给另一端，再建立本地转发：

~~~sh
tailcat forward 'tch3…' 18080:8080
# 访问 http://127.0.0.1:18080
~~~

直接打开远端 HTTP 服务：

~~~sh
tailcat browse 'tch3…'
~~~

SSH 建议显式限制身份：

~~~sh
tailcat serve --ssh-authorized-keys ~/.ssh/authorized_keys ssh
tailcat ssh 'tch3…'
~~~

每条连接固定运行一个程序：

~~~sh
tailcat serve exec -- /path/to/program arg1 arg2
~~~

文件传输、SOCKS5、退出节点和更细的授权参数请查看 `tailcat --help` 与对应子命令的 `--help`。

## 性能

`v0.7.0-quic.2` 的读取优化复用有上限的 32 KiB H3 流缓冲区，只合并已经就绪的数据，不会等待凑出更大的批次。

AU / US1420 发布前 A/B 中，四连接均值一个方向从 **333.41 → 368.47 Mbps**，另一方向从 **211.56 → 305.86 Mbps**；单连接均值分别为 **299.36 → 299.41 Mbps** 与 **231.34 → 262.86 Mbps**。这些只是有限 WAN 样本，不代表所有线路都同比提升。

最终使用公开固定依赖构建的 Linux 发布程序，四连接双向另测得 **327.43 / 321.95 Mbps**。详细方法、低速样本、负载延迟和限制见 [完整验证报告](docs/release-validation-v0.7.0-quic.2.md)。

## 安全边界

HTTP/3 是真实协议封装，但不代表“和所有浏览器完全无法区分”。私有主机名、端口、包长、时序、发现流量和中继行为仍可能形成可见特征。

- TLS 校验、连接秘密和节点授权默认保留。
- 本分支拒绝 `--psk=false`。
- TCP 流只能建立在已经完成认证的 QUIC 会话上，并再次检查当前 Peer / 服务权限。
- UDP/IP 继续使用经过认证的 CONNECT-IP / QUIC DATAGRAM。
- 不提供 WireGuard/AWG 数据面回退。
- 依赖 `wireguard-go` 的 TUN/网络基础组件，不代表又叠加一层 WireGuard 加密。

`tch3…` 连接地址本身包含访问秘密，应视为凭据。详见 [SECURITY.md](SECURITY.md)。

## 源码构建

需要 Go 1.27.1：

~~~sh
git clone https://github.com/LiuTangLei/tailcat-quic.git
cd tailcat-quic
git checkout quic-v0.7
go build -trimpath -tags "$(cat build-tags.txt)" -o tailcat ./cmd/tailcat
./tailcat version
~~~

正式发布只使用公开、不可变的依赖版本。应用本身只维护在这个仓库；共用 QUIC 协议库在 `LiuTangLei/quic-go`，H3 集成在 `LiuTangLei/tailscale`。

## 发布与打包

发布包在公开前会先做本地验证。公开仓库可以使用 GitHub 免费的标准 runner 做有限的平台/容器任务；不使用 Larger/GPU runner，也不让每次普通提交都触发重型构建。共用 `quic-go` 仓库继续不启用自动构建。

安装细节见 [INSTALL.md](INSTALL.md)，维护者发布流程见 [docs/manual-release.md](docs/manual-release.md)。

这是独立 fork，不由 Tailscale 官方支持。原始版权与 BSD-3-Clause 许可证见 [LICENSE](LICENSE)，依赖声明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
