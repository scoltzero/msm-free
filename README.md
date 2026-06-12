# msf

[English README](README.en.md)

`msf` 是一个面向 MosDNS + Mihomo 工作流的 MSM 风格管理面板重构版。项目目标是提供可自部署、可审计的 DNS 分流、透明代理、Mihomo 管理和 Unraid 插件体验。

当前发布版本：`v0.3.2`

## 功能概览

- 原版 MSM 风格 6 步初始化向导，支持管理员账号、系统参数、DNS、IPv6、Fake-IP、透明代理和组件安装配置。
- MosDNS + Mihomo 默认组合，暂不启用 sing-box。
- 按 mssb 风格生成 MosDNS + Mihomo 国内外分流链路：MosDNS `:53` 入口，Mihomo DNS `:6666`，Fake-IP `28.0.0.0/8`，TProxy `7896`，Redirect `7877`。
- 支持机场订阅，前端按 `名称|URL` 保存，后端会生成 Mihomo `proxy-providers`。
- 支持初始化阶段添加手动节点，手动节点会生成 `proxy_providers/msf_manual.yaml`，并作为 Mihomo 本地文件型供应商 `msf_manual` 使用。
- 手动节点分享链接基础解析支持 `ss`、`ssr`、`vmess`、`vless`、`trojan`、`hysteria`、`hysteria2`、`tuic`。
- 支持 MosDNS 客户端代理模式：关闭、白名单、黑名单。
- 支持 Mihomo 节点、规则、连接、日志、配置页面。
- 支持 Mihomo 自定义配置：用户配置可导入、新建、命名、保存、应用和回滚，代理分组、规则集和规则可在 WebUI 中管理。
- 支持 CodeMirror YAML 配置编辑器，提供高亮、行号、撤销重做和搜索快捷键。
- 支持组件更新检查、自动检查间隔、自动下载、更新通知和升级方式配置。
- 支持 MosDNS、Mihomo、Zashboard 本地上传安装，网络困难时可用预下载核心离线安装。
- 支持 GitHub 组件下载代理和加速前缀，HTTP、HTTPS、SOCKS5 均可配置。
- 支持按运行架构下载组件核心，ARM64 版本会下载并校验 ARM64 MosDNS/Mihomo。
- 支持普通 Linux systemd 安装包。
- 支持 Unraid 插件安装方式。
- 支持 Docker host 网络部署方式。

## 下载

GitHub Release：

```text
https://github.com/scoltzero/msf/releases/tag/v0.3.2
```

Linux x86_64 安装包：

```text
https://github.com/scoltzero/msf/releases/download/v0.3.2/msf-linux-amd64.tar.gz
```

Linux ARM64 安装包：

```text
https://github.com/scoltzero/msf/releases/download/v0.3.2/msf-linux-arm64.tar.gz
```

Unraid 插件文件：

```text
https://github.com/scoltzero/msf/releases/download/v0.3.2/msf.plg
```

## Linux 安装

根据机器架构下载并解压。x86_64 / amd64 使用：

```bash
curl -L -o msf-linux-amd64.tar.gz \
  https://github.com/scoltzero/msf/releases/download/v0.3.2/msf-linux-amd64.tar.gz

tar -xzf msf-linux-amd64.tar.gz -C /tmp
sudo /tmp/msf-0.3.2-linux-amd64/install.sh
```

ARM64 / aarch64 使用：

```bash
curl -L -o msf-linux-arm64.tar.gz \
  https://github.com/scoltzero/msf/releases/download/v0.3.2/msf-linux-arm64.tar.gz

tar -xzf msf-linux-arm64.tar.gz -C /tmp
sudo /tmp/msf-0.3.2-linux-arm64/install.sh
```

安装脚本会完成这些操作：

- 安装二进制到 `/usr/local/bin/msf`
- 注册兼容命令 `/usr/local/bin/msm`
- 初始化数据目录 `/opt/msf`
- 安装 systemd 服务 `msf.service`
- 启动 WebUI，默认监听 `7777`

打开 WebUI：

```text
http://<服务器IP>:7777
```

首次进入会显示初始化向导。完成初始化后，`msf` 会持久化运行态；后续重启时会按配置恢复 Mihomo、MosDNS 和 nftables，除非用户在 WebUI 中显式停止服务或清除 nftables。

常用 systemd 命令：

```bash
sudo systemctl status msf
sudo systemctl stop msf
sudo systemctl restart msf
sudo journalctl -u msf -f
```

常用 CLI 命令：

```bash
sudo msf status
sudo msf stop
sudo msf restart
sudo msf logs
sudo msf logs --lines 200 mosdns
sudo msf logs --lines 200 mihomo
sudo msf doctor
sudo msf update
```

`msm` 和 `msf` 指向同一套 CLI。`msf stop` 会向正在运行的管理进程发送优雅停止信号，管理进程退出前会停止它托管的 MosDNS 和 Mihomo 子进程。

需要强制停止时：

```bash
sudo msf stop --timeout 20s --force
```

卸载：

```bash
sudo msf uninstall
```

默认卸载只删除 systemd 服务和 `/usr/local/bin/msf`，会保留 `/opt/msf` 数据目录。需要连配置、数据库、日志、组件二进制一起删除时再显式执行：

```bash
sudo msf uninstall --purge
```

如果还保留着解压后的发布包，也可以在包目录内运行：

```bash
sudo ./uninstall.sh
sudo ./uninstall.sh --purge
```

## Unraid 插件安装

在 Unraid WebGUI 中打开 **Plugins / Install Plugin**，填入插件地址：

```text
https://github.com/scoltzero/msf/releases/download/v0.3.2/msf.plg
```

安装完成后打开 **Settings / MSF Free**，进入轻量插件控制页，再点击打开 WebUI。完整管理界面运行在独立 WebUI 中，不嵌入 Unraid Settings 页面。

Unraid 默认数据目录：

```text
/mnt/user/appdata/msf
```

Unraid 运行逻辑：

- 全新安装且尚未初始化时，只启动 `msf` 管理 WebUI。
- 完成初始化引导后，默认启用 Mihomo、MosDNS 和 nftables。
- Unraid 重启或插件服务重启后，`msf` 会按已保存状态恢复 Mihomo、MosDNS 和 nftables。
- 如果用户在 WebUI 中手动停止服务或清除 nftables，下次启动会尊重这个关闭状态。
- 在线安装 MosDNS、Mihomo、Zashboard 时会先校验 GitHub release asset SHA-256 digest；本地上传核心标记为 `local-upload`。

Unraid 停止服务：

```bash
/etc/rc.d/rc.msf stop
```

Unraid 卸载请在 WebGUI 的插件管理页面删除 `msf` 插件。插件卸载会停止 WebUI 服务并移除插件文件，默认保留 `/mnt/user/appdata/msf` 数据目录；如需彻底清理，需要手动删除该目录。

Unraid 上不要使用 `msf update` 或 `msf uninstall`，更新和卸载应通过 Unraid 插件管理页面完成，避免绕过 `.plg` 包状态。

## Docker 部署

Docker 镜像使用 host 网络一比一复刻普通 Linux 二进制能力，容器会写入宿主机 nftables 和策略路由。不要使用 br0、macvlan、ipvlan 或 bridge 静态 IP 作为等价部署方式。

使用 Docker Compose：

```bash
docker compose up -d
```

没有 `docker compose` 时可以使用普通 Docker：

```bash
./docker-run.sh
```

两种方式默认数据目录均为当前目录下的 `./msf-data`，WebUI 仍为：

```text
http://<服务器IP>:7777
```

详细要求、普通 `docker run` 命令、关闭清理和升级方式见 [Docker 部署文档](docs/docker.md)。

## 初始化配置说明

初始化向导会写入系统配置、生成 MosDNS/Mihomo 配置，并保存到数据库。几个关键点：

- 机场订阅：保存为 `名称|URL` 换行格式，最终进入 Mihomo `proxy-providers`。
- 手动节点：保存到 `mihomo_proxies`，生成 `configs/mihomo/proxy_providers/msf_manual.yaml`。
- `msf_manual`：在 Mihomo 中作为 `type: file` 的本地 proxy provider 注册。
- GitHub 下载代理：只影响组件下载过程，不会修改代理服务本身的运行配置。
- 自定义 Mihomo 配置：用户配置保存在 `configs/mihomo/user_configs`，应用时会覆盖运行入口 `configs/mihomo/config.yaml` 并重启 Mihomo。
- 配置文件列表：WebUI 只展示用户可管理的配置文件，不直接展示内部启动文件 `config.yaml`。
- 本地上传组件：系统设置 / 系统更新 / 组件更新中可上传本机预下载的 MosDNS、Mihomo 或 Zashboard 包。

### 服务端口分配

下表为本项目实际监听的端口（以 `internal/server` 中的诊断/健康检查清单为准）。supervisor 通过 unix socket 管理进程，不占用 TCP 端口，故未列入。

| 服务 | 端口 | 描述 |
|---|---|---|
| msf | 7777 | Web 管理界面（默认 `-p 7777`）；mosdns 亦复用它解析节点域名 |
| mosdns | 53 | DNS 服务入口 |
| mosdns | 2222 | 内部的国内 DNS 服务器 |
| mosdns | 3333 | 转发国外请求到内部带过期缓存的服务 |
| mosdns | 4444 | 带过期缓存的内部使用/外部使用的国外 DNS 服务器 |
| mosdns | 5656 | 主分流服务器 |
| mosdns | 6666 | 与 mihomo/sing-box 的 DNS 对接 |
| mosdns | 8888 | 内部 DNS（代理核心的默认上游 `default-nameserver`） |
| mosdns | 9099 | MosDNS 统计 / API 接口 |
| mihomo/sing-box | 7890 | HTTP 代理 |
| mihomo/sing-box | 7891 | SOCKS5 代理 |
| mihomo/sing-box | 7892 | 混合端口（Mixed） |
| mihomo/sing-box | 7896 | TProxy 透明代理（nftables 策略使用） |
| mihomo/sing-box | 7877 | Redirect 代理（nftables 策略使用） |
| mihomo/sing-box | 9090 | 外部控制器 / Web 界面（zashboard） |

## 路由器接入（让局域网设备走 msf）

msf 默认以**旁路由**方式工作：它本身不是主网关，由主路由把 **DNS 查询**和**需要代理的流量**引导到 msf 主机。要让局域网设备生效，主路由需要完成两步：

1. **DHCP DNS 重定向**：把客户端 DNS 指向 msf 主机（MosDNS `:53`）。
2. **FakeIP 静态路由**：把 FakeIP 网段的下一跳指向 msf 主机。

| 类型 | 目标网段（msf 默认值） | 下一跳 |
|---|---|---|
| IPv4 | `28.0.0.0/8` | msf 主机 IPv4 |
| IPv6 | `f2b0::/18` | msf 主机 IPv6 |

> 只配 DNS 不配静态路由不行：FakeIP 是虚拟地址，缺少回程路由会被丢弃或直连出去。两步缺一不可。FakeIP 网段需与初始化向导里的实际配置一致。

按主路由系统查看分步教程：

- [路由器接入总览](docs/guide/zh/router-integration.md)
- [RouterOS（MikroTik）](docs/guide/zh/routeros.md)
- [爱快 iKuai](docs/guide/zh/ikuai.md)
- [OpenWrt](docs/guide/zh/openwrt.md)
- [UniFi（Ubiquiti）](docs/guide/zh/unifi.md)

验证：客户端执行 `nslookup google.com` 结果应落在 `28.0.0.0/8`，`dig AAAA google.com` 落在 `f2b0::/18`。

## 运行目录

普通 Linux 默认数据目录：

```text
/opt/msf
```

Unraid 默认数据目录：

```text
/mnt/user/appdata/msf
```

主要目录结构：

- `configs/mosdns`
- `configs/mihomo`
- `configs/network`
- `data/binaries`
- `logs`
- `database`
- `backups`

## 从源码构建

本地运行：

```bash
go run ./cmd/msf serve -c ./data -p 7777
```

构建 Linux x86_64 压缩包：

```bash
make package VERSION=0.3.2 GOOS=linux GOARCH=amd64
```

构建 Linux ARM64 压缩包：

```bash
make package VERSION=0.3.2 GOOS=linux GOARCH=arm64
```

构建 Unraid 插件产物：

```bash
make unraid VERSION=0.3.2 UNRAID_VERSION=0.3.2 GITHUB_REPO=scoltzero/msf RELEASE_TAG=v0.3.2
```

构建产物：

- `dist/msf-linux-amd64.tar.gz`
- `dist/msf-linux-arm64.tar.gz`
- `dist/unraid/msf-0.3.2-x86_64-1.txz`
- `dist/unraid/msf.plg`
- `msf.plg`

发布时，Linux `.tar.gz`、Unraid `.txz` 和 `.plg` 上传到 GitHub Release。根目录 `msf.plg` 可用于仓库分支安装入口。

## 说明

`msf` 不包含 MSM 的闭源后端代码。项目目标是做一个非商业用途的开放重构版：外观和使用体验参考 MSM，后端行为围绕 mssb 风格的 MosDNS + Mihomo 工作流重新实现。

## 鸣谢

感谢这些项目提供参考：

- `msm9527/msm-wiki`：作为 MSM 管理体验和功能组织的公开参考。
- `baozaodetudou/mssb`：作为 MosDNS + Mihomo 后端工作流的公开参考。

本项目与 MSM、mssb 上游项目没有隶属关系。
