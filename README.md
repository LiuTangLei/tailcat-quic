# tailcat-quic

基于 [官方 tailcat v0.6.0](https://github.com/tailscale/tailcat/tree/v0.6.0) 的点对点 QUIC 加密隧道。**默认使用 HTTP/3 流量特征和 BBRv3 拥塞控制**，复制一段连接码即可连接两台机器，无需账号、控制服务器或手动配置证书。

支持端口转发、字节流传输、文件传输和 SSH。只使用 QUIC 数据面，不提供 WireGuard/AWG 模式；需要标准 WireGuard 请使用官方 tailcat。

## 安装

从 [Releases](https://github.com/LiuTangLei/tailcat-quic/releases/latest) 下载对应平台的包，用同页的 `checksums.txt` 校验后解压。支持 Linux、macOS 和 Windows。

**项目名为 `tailcat-quic`，命令仍是 `tailcat`（Windows 为 `tailcat.exe`）。两端均需安装本项目版本。**

```sh
tailcat version
tailcat --help
```

## 快速使用

在提供服务的机器上，将本机 TCP 8080 端口开放给持有连接码的客户端：

```sh
tailcat serve 8080
```

复制输出的完整 `tch3…` 连接码，在另一台机器执行：

```sh
tailcat forward 'tch3…' 18080:8080
```

随后访问客户端的 `http://127.0.0.1:18080`。把示例里的 `tch3…` 替换为实际连接码；本地转发端口默认只监听回环地址。`serve PORT` 和 `forward` 是 TCP 命令，不会自动开放同端口的 UDP。

其他常用方式：

```sh
# 多端口转发：服务端与客户端分别执行
tailcat serve 8080,3306
tailcat forward 'tch3…' 18080:8080 13306:3306

# 只读共享目录：服务端执行
tailcat serve --files=./shared:ro
# 客户端列出或下载文件
tailcat ls 'tch3…'
tailcat cp 'tch3…:example.txt' ./example.txt

# 检查直连是否建立
tailcat ping --until-direct --timeout=30s 'tch3…'
```

连接码是接入凭据，请私下传递，不要放进公开 issue、DNS 记录或日志截图。更多选项见 `tailcat serve --help`、`tailcat forward --help` 和 `tailcat ssh --help`。

## 默认流量特征

| 项目 | 实际行为 |
| --- | --- |
| 直连传输 | UDP 上的 QUIC，使用真正的 HTTP/3，TLS ALPN 为 `h3`；不发送 WireGuard 握手或加密数据包。 |
| TCP 业务 | 每条业务连接对应独立的 HTTP/3 CONNECT 可靠流，多个流共享 QUIC 连接。 |
| UDP/IP 数据 | 通过 CONNECT-IP / QUIC DATAGRAM 传输，保留不可靠数据报语义；丢失的 DATAGRAM 不由 QUIC 重传。 |
| 连接发现 | 复用 magicsock 的端点发现、NAT 打洞和 DERP；无需登录 Tailscale。 |
| 中继 | 不能直连时可经 DERP 承载加密数据；公网看到的外层是 DERP 连接，不能把它描述成直连 HTTP/3 流量。 |
| 拥塞控制 | 两端默认使用用户态 BBRv3，不需要修改系统 TCP 参数或手动指定带宽。 |

这里的“混淆”是用 QUIC/HTTP/3 替代 WireGuard 数据面的协议特征，**不是保证与普通浏览器访问完全不可区分**。发现流量、握手特征、地址、包长和时序仍可能被观察；只允许 TCP 或封锁 QUIC/DERP 的网络仍可能无法连接。

协议参考：[HTTP/3（RFC 9114）](https://www.rfc-editor.org/rfc/rfc9114.html)、[QUIC DATAGRAM（RFC 9221）](https://www.rfc-editor.org/rfc/rfc9221.html)、[CONNECT-IP（RFC 9484）](https://www.rfc-editor.org/rfc/rfc9484.html)。

## 加密与身份认证

**传输加密：** 使用 QUIC 的 TLS 1.3 握手和认证加密保护业务数据及完整性；AES-GCM 或 ChaCha20-Poly1305 等具体套件由 TLS 协商，并非固定强制一种算法。不是明文 HTTP，也不是在 QUIC 内再运行 WireGuard 加密层。详见 [RFC 9001](https://www.rfc-editor.org/rfc/rfc9001.html)。

**双向认证：** 连接码包含服务端节点公钥和随机接入密钥。节点认证使用 `Noise_IK_25519_ChaChaPoly_BLAKE2s`，并将双方身份和接入密钥绑定到当前 TLS 会话；只知道节点公钥或只建立 TLS 连接，不能获得转发权限。可额外使用 `--allow` 限定客户端节点。

**证书与密钥：** TLS 证书自动在本地生成，无需购买域名证书或手动互换证书；信任由连接码和节点证明建立，而非公开网站 CA。长期连接使用 QUIC Key Update，并继续检查节点撤销，不为定时重新握手而中断 SSH 或文件流；这不等同于 WireGuard 的完整重握手周期。

连接码、私钥和被开放的服务共同决定访问范围。仅开放必要端口，详细边界见 [SECURITY.md](SECURITY.md)。吞吐和延迟取决于链路，不承诺所有方向都优于 WireGuard；已有测量与发行验证保留在 [docs](docs/)。

## 源码构建

使用 `go.mod` 指定的 Go 版本，从本仓库构建：

```sh
git clone https://github.com/LiuTangLei/tailcat-quic.git
cd tailcat-quic
go build -trimpath -o tailcat ./cmd/tailcat
```

本项目是独立 fork，不是 Tailscale 官方支持的产品；官方客户端及浏览器/WASM 客户端不能直接使用本项目的连接码。原始版权与 BSD-3-Clause 许可证见 [LICENSE](LICENSE)，依赖声明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
