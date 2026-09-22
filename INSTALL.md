# 安装 tailcat-quic

本页适用于 `LiuTangLei/tailcat-quic`，不是官方使用 WireGuard 的 tailcat。两端都必须安装本 fork，使用 `tch3…` 连接码。

## 预编译包

打开 https://github.com/LiuTangLei/tailcat-quic/releases ，选择 `v0.7.0-h3.2` 对应系统和架构，同时下载 `checksums.txt`。0.7 因方向性性能回退标为预发布；`/releases/latest` 仍指向现有稳定版。不要只凭文件名判断版本，解压后执行 `tailcat version`。

Linux/macOS 可以用 `sha256sum -c checksums.txt --ignore-missing`（macOS 可用 `shasum -a 256` 对照校验值）；Windows 可以用 PowerShell `Get-FileHash -Algorithm SHA256`。校验成功后解压，把 `tailcat` 或 `tailcat.exe` 放到 PATH 中。

Linux 的 deb/rpm 包名为 `tailcat-quic`，与占用同一可执行文件名的 `tailcat` 包冲突，不应同时安装。安装升级不会自动给你的服务器开放 SSH、代理或退出节点服务；这些服务由显式 CLI 命令启动。

macOS 二进制未作 Apple 公证，首次运行可能需要在系统隐私与安全设置中批准。不要为安装而全局关闭系统安全功能。

## Android / Termux

Linux 架构对应的二进制保留上游 0.7 的 Android 运行支持：运行时检测 Android，接入系统 DNS、证书和受限网络接口回退；普通 Linux 不受这些入口影响。它仍是命令行程序，不是 Android VPN 应用。本仓库发布说明会区分交叉编译/单元测试与实际 Android 设备验证。

## 源码构建

需要 Go 1.27.1。`go.mod` 必须固定公开版本，不要把个人电脑上的绝对路径或开发用 `go.work` 带进发布。

```sh
git clone https://github.com/LiuTangLei/tailcat-quic.git
cd tailcat-quic
go build -trimpath -o tailcat ./cmd/tailcat
./tailcat --help
```

不要使用官方仓库的 `go install github.com/tailscale/tailcat/cmd/tailcat@latest` 来安装本混淆版，那会得到不同的传输实现。

具体使用、安全边界和发布验证结果见 README、SECURITY.md 及 `docs/release-validation-v0.7.0-h3.2.md`。
