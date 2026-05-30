# msm-free

[English README](README.en.md)

`msm-free` 是一个面向 MosDNS + Mihomo 体系的 MSM 风格管理面板重构版。第一版目标是 x86_64 Linux 和 Unraid 插件。

## 当前范围

- 首版支持 x86_64 Linux。
- 代理核心只实现 Mihomo，暂不实现 sing-box。
- MosDNS、Mihomo、nftables 透明代理、初始化引导、用户管理、配置历史、日志和更新相关 API 都作为开放功能实现。
- 已按 mssb 方式生成 MosDNS + Mihomo 国内外分流链路：MosDNS `:53` 入口，Mihomo DNS `:6666`，fake-ip `28.0.0.0/8`，TProxy `7896`，redirect `7877`，并包含 `2222/3333/4444/5656/7777/8888/9099` 等 MosDNS 侧端口。
- 支持普通 Linux systemd 安装包。
- 支持 Unraid 插件安装方式。

## Linux x86_64 压缩包使用

从 GitHub Release 下载：

```bash
curl -L -o msm-free-linux-amd64.tar.gz \
  https://github.com/scoltzero/msm-free/releases/download/v0.1.5/msm-free-linux-amd64.tar.gz
```

解压并安装：

```bash
tar -xzf msm-free-linux-amd64.tar.gz -C /tmp
sudo /tmp/msm-free-*-linux-amd64/install.sh
```

安装脚本会完成这些操作：

- 安装二进制到 `/usr/local/bin/msm-free`
- 初始化数据目录 `/opt/msm-free`
- 安装 systemd 服务 `msm-free.service`
- 启动 WebUI，默认监听 `7777`

安装完成后打开：

```text
http://<服务器IP>:7777
```

首次进入会显示初始化引导。完成初始化后，`msm-free` 会持久化运行态；后续重启时会按配置恢复 Mihomo、MosDNS 和 nftables，除非用户在 WebUI 中显式停止服务或清除 nftables。

常用命令：

```bash
sudo systemctl status msm-free
sudo systemctl restart msm-free
sudo journalctl -u msm-free -f
```

## Unraid 插件使用

在 Unraid WebGUI 中打开 **Plugins / Install Plugin**，填入插件地址：

```text
https://raw.githubusercontent.com/scoltzero/msm-free/main/msm-free.plg
```

安装完成后打开 **Settings / MSM Free**，进入插件设置页，再点击打开 WebUI。

Unraid 默认数据目录：

```text
/mnt/user/appdata/msm-free
```

Unraid 运行逻辑：

- 全新安装且尚未初始化时，只启动 `msm-free` 管理 WebUI。
- 完成初始化引导后，默认启用 Mihomo、MosDNS 和 nftables。
- Unraid 重启或插件服务重启后，`msm-free` 会按已保存状态恢复 Mihomo、MosDNS 和 nftables。
- 如果用户在 WebUI 中手动停止服务或清除 nftables，下次启动会尊重这个关闭状态。

## 从源码构建

本地运行：

```bash
go run ./cmd/msm-free serve -c ./data -p 7777
```

构建 Linux x86_64 压缩包：

```bash
make build VERSION=0.1.5
```

构建 Unraid 插件产物：

```bash
make unraid VERSION=0.1.5 UNRAID_VERSION=0.1.5 GITHUB_REPO=scoltzero/msm-free RELEASE_TAG=v0.1.5
```

构建产物：

- `dist/msm-free-linux-amd64.tar.gz`
- `dist/unraid/msm-free-0.1.5-x86_64-1.txz`
- `msm-free.plg`

发布时，`.txz` 和 Linux `.tar.gz` 上传到 GitHub Release，`msm-free.plg` 保留在仓库根目录供 Unraid 安装器读取。

## 运行目录

普通 Linux 默认数据目录：

```text
/opt/msm-free
```

Unraid 默认数据目录：

```text
/mnt/user/appdata/msm-free
```

主要目录结构：

- `configs/mosdns`
- `configs/mihomo`
- `configs/network`
- `data/binaries`
- `logs`
- `database`
- `backups`

## 说明

`msm-free` 不包含 MSM 的闭源后端代码。项目目标是做一个非商业用途的开放重构版：外观和使用体验参考 MSM，后端行为围绕 mssb 风格的 MosDNS + Mihomo 工作流重新实现。

## 鸣谢

感谢这些项目提供参考：

- `msm9527/msm-wiki`：作为 MSM 管理体验和功能组织的公开参考。
- `baozaodetudou/mssb`：作为 MosDNS + Mihomo 后端工作流的公开参考。

本项目与 MSM、mssb 上游项目没有隶属关系。
