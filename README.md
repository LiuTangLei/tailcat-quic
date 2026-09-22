# tailcat-quic

基于 [官方 tailcat v0.7.0](https://github.com/tailscale/tailcat/tree/v0.7.0) 的独立点对点加密隧道。默认使用 **HTTP/3、TLS 1.3 和用户态 BBRv3**；TCP 服务使用经过认证的可靠 H3 流，UDP 服务使用 QUIC DATAGRAM。复制连接码即可连接，无需账号、控制服务器或手动分发证书。

**应用仅在本仓库维护。** 共用协议实现来自固定版本的 `LiuTangLei/quic-go` 和 `LiuTangLei/tailscale`，不依赖 `tailcat-tailscale` / `tailcat-quic-go` 镜像。只使用 QUIC 数据面，不提供 WG/AWG 切换。`wireguard-go` 依赖里的 TUN/网络基础组件不代表数据又被 WireGuard 加密一次。

## 安装

从 [Releases](https://github.com/LiuTangLei/tailcat-quic/releases/latest) 下载对应平台的包，用同页的 `checksums.txt` 校验后解压。提供 Linux、macOS、Windows 的可执行文件；Linux 另有 deb/rpm。项目名是 `tailcat-quic`，命令仍为 `tailcat`（Windows 为 `tailcat.exe`）。

```sh
tailcat version
tailcat --help
```

**连接双方都需要本 fork。** 使用 `tch3…` 连接码，不能与官方 WireGuard `tc…` 连接码互通；浏览器/WASM 客户端未作为本 fork 的受支持客户端发布。安装细节见 [INSTALL.md](INSTALL.md)。

## 快速使用

服务器分享本地端口：

```sh
tailcat serve 8080
```

把服务器输出的完整 `tch3…` 连接码通过可信渠道发给客户端：

```sh
tailcat forward 'tch3…' 18080:8080
# 然后访问 http://127.0.0.1:18080
```

自动打开浏览器：

```sh
tailcat forward --open-browser 'tch3…' 18080:8080
# browse 是把远端 80 端口映射到空闲本地端口并打开浏览器的快捷命令
tailcat browse 'tch3…'
```

SSH 服务必须使用身份限制，尤其不要把无认证 shell 的连接码公开到 DNS：

```sh
tailcat serve --ssh-authorized-keys ~/.ssh/authorized_keys ssh
tailcat ssh 'tch3…'
```

`tailcat serve exec -- COMMAND` 可以为每条连接运行固定程序；SSH 服务中的 `-- COMMAND` 则作为强制命令，不开放任意 shell/SFTP。文件服务、SOCKS5、退出节点和权限参数以对应命令的 `--help` 为准。退出节点模式会允许客户端使用服务器网络访问其他目标，只应分享给可信客户端。

## 0.7 合并内容

包含上游的 UDP 退出节点转发、Windows localhost 双栈处理、旧 OpenSSH 的 SFTP 兼容、`Server.Listen` TCP/UDP API、Peer 直连/中继状态、browse、exec/SSH 强制命令和 DNS 公开地址安全提示。Linux 可执行文件也保留上游 Android/Termux 的 DNS、系统证书与网络接口兼容入口。

H3 直传路径专门适配了 `Server.Listen`：显式监听端口优先于通配回调，连接交给 `Accept` 后不会被旧回调过早关闭；关闭服务时仍能释放连接。

共用 QUIC 库已同步 0.63，保留此前的批量收发、受限队列、握手和关闭语义修复。底层 gVisor 更新到与上游 0.7 对齐的版本，恢复经上游修复后的 CUBIC/RACK。内核 TUN 批读对使用用户态网络栈的 Tailcat 不直接适用，不能把 Tailscale IP 隧道的提速数字当作 Tailcat 测速结果。

发布前性能与兼容性结果见 `docs/release-validation-v0.7.0-h3.2.md`；没有证据的方向不承诺提速。

## 安全边界

HTTP/3 是真实协议封装，不是“与浏览器完全无法区分”的保证。私有源站、端口、包长、时序及发现/中继流量仍可能形成特征。保持 TLS 校验、当前节点授权、连接密钥绑定和拥塞控制，不通过取消加密来换取吞吐。

连接码含有访问秘密，请视同凭据保管。`--psk=false` 在此 fork 中被拒绝；无需手工证书不代表未经认证。详细说明见 [SECURITY.md](SECURITY.md)。

## 从源码构建

需要 Go 1.27.1 或支持自动下载该工具链的 Go 环境。正常构建只使用 `go.mod` 中可公开下载的固定依赖，不需要本地 worktree 或 `go.work`。

```sh
git clone https://github.com/LiuTangLei/tailcat-quic.git
cd tailcat-quic
go build -trimpath -o tailcat ./cmd/tailcat
```

这是独立 fork，不是 Tailscale 官方支持的产品。原始版权及 BSD-3-Clause 许可证见 [LICENSE](LICENSE)，依赖声明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
