# 更新日志

## v0.3.2 - 2026-06-12

### 中文

#### 说明

- 这是一次小型修复发布，基于 v0.3.1 的 CA 合规版本继续修正组件校验状态展示和初始化向导体验。
- 本版本发布资产继续包含 Linux amd64/arm64 tarball、Unraid `.txz`/`.plg`，以及从同步后的 `fnos-fpk` 分支构建的 fnOS x86/arm `.fpk` 包。

#### 修复

- 修复 MosDNS、Mihomo、Zashboard 在线安装校验成功后，后续更新检查可能把同一 digest 的已验证状态重置为“待安装校验”的问题。
- 修复组件更新状态在无新版本且 digest 未变化时的 `verified_digest`、`verified` 和 `verification_source` 保留逻辑。

#### 变更

- 改进初始化向导校验流程：提交时弹出需要修改的步骤提示，用户点击后再跳转到对应步骤。
- 初始化管理员密码只要求非空和两次输入一致，不再强制最少 8 位。
- 初始化密码输入框增加显示/隐藏切换。

### English

#### Notes

- This is a small fix release on top of the v0.3.1 CA compliance release, focused on component verification state display and setup wizard usability.
- Release assets continue to include Linux amd64/arm64 tarballs, Unraid `.txz`/`.plg`, and fnOS x86/arm `.fpk` packages built from the synced `fnos-fpk` branch.

#### Fixed

- Fixed MosDNS, Mihomo, and Zashboard status checks resetting a successfully installed and verified same-digest component back to a pending verification state.
- Fixed preservation of `verified_digest`, `verified`, and `verification_source` when no update is available and the GitHub asset digest is unchanged.

#### Changed

- Improved setup wizard validation: invalid submissions now show a modal with the step that needs attention, then jump there when confirmed.
- Setup administrator passwords now only require a non-empty value and matching confirmation; the previous minimum length requirement is no longer enforced.
- Added show/hide controls for setup password fields.

## v0.3.1 - 2026-06-11

### 中文

#### 说明

- 这是一次以 Unraid CA 合规为主的修复发布，同时补齐 v0.3.0 之后已经进入 `main` 的 Docker、文档、发布流程和运行时修复。
- 本版本发布资产包含 Linux amd64/arm64 tarball、Unraid `.txz`/`.plg`，以及从同步后的 `fnos-fpk` 分支构建的 fnOS x86/arm `.fpk` 包。

#### 新增

- 新增在线组件下载完整性校验：MosDNS、Mihomo、Zashboard 在线安装会读取 GitHub Release API asset `digest`，只接受合法 `sha256:<hex>`，下载后哈希匹配才会解压或覆盖现有核心。
- 新增组件校验状态字段：组件更新状态会返回并持久化 `download_digest`、`verified_digest`、`verified` 和 `verification_source`。
- 新增本地上传来源标记：手动上传核心仍保留 ELF/架构校验，但会显示为 `local-upload` 且 `verified=false`，不声明由项目验证。
- 新增 `make audit-compliance` 和 `scripts/audit-compliance.sh`，扫描源码与构建产物中的旧真实订阅、真实节点、真实 IP 和非 inert 代理 URL 样例。
- 新增 Docker host-network 部署支持，包含 `Dockerfile`、`docker-compose.yml`、`docker-run.sh`、GHCR workflow 与 Docker 部署文档。
- 新增 Docker runtime 检测与保护：容器内禁用主机级 `msf update`、`msf uninstall` 和 systemd service install/uninstall。
- 新增路由器集成文档，覆盖 OpenWrt、RouterOS、iKuai、UniFi 的中英文 DHCP / 静态路由配置说明。
- 新增手动发布 runbook，并在 README 中补充服务端口占用表。

#### 变更

- 清空默认初始化配置中的真实订阅链接、真实分享节点和 YAML 节点；输入占位与 MosDNS 规则演示数据统一改为 `example.invalid`、`198.51.100.0/24`、`203.0.113.0/24` 等 inert 示例。
- Unraid Settings 入口保留为独立轻量插件控制页，只提供启停、监听地址、端口、数据目录、状态和 `Open WebUI`；完整管理界面运行在独立 MSF WebUI。
- 更新 Unraid CA 模板、`ca_profile.xml`、README 和 `.plg` release notes，明确核心下载校验、本地上传语义和 Settings 控制页边界。
- 将项目描述统一调整为 free and open-source DNS & proxy management tool。
- 移除 GitHub Actions release workflow，发布改为手动 runbook 流程。
- 从版本库移除 `.codex`、graphify 等本地工具产物，并通过 `.gitignore` 阻止再次纳入。

#### 修复

- 修复 MosDNS 缓存统计读取与展示逻辑。
- 修复 nftables / policy routing 应用逻辑：应用前清理旧 `table inet msf` 与重复 fwmark rule，路由使用 `replace`，清理时同时处理 IPv4/IPv6 policy route。
- 修复 daemon stop/restart 与进程退出路径，统一走 `ShutdownRuntime` 清理运行时状态。
- 修复 Unraid CA submission scanner 对模板/profile XML 的识别问题。
- 修复 `component_update_info` 旧数据库缺少新增校验列时的迁移兼容。

### English

#### Notes

- This is primarily an Unraid CA compliance release, while also documenting the Docker, runtime, release-process, and documentation changes that landed after v0.3.0.
- Release assets include Linux amd64/arm64 tarballs, Unraid `.txz`/`.plg`, and fnOS x86/arm `.fpk` packages built from the synced `fnos-fpk` branch.

#### Added

- Added integrity verification for online component downloads: MosDNS, Mihomo, and Zashboard installs now read the GitHub Release API asset `digest`, require a valid `sha256:<hex>` value, and install only after the downloaded file hash matches.
- Added component verification state fields: `download_digest`, `verified_digest`, `verified`, and `verification_source` are persisted and returned by the component update APIs.
- Added explicit local-upload provenance: manually uploaded cores still go through ELF/architecture checks, but are shown as `local-upload` with `verified=false` instead of being presented as project-verified downloads.
- Added `make audit-compliance` and `scripts/audit-compliance.sh` to scan source and generated artifacts for old live subscriptions, live nodes, live IPs, and non-inert proxy URL samples.
- Added Docker host-network deployment support with `Dockerfile`, `docker-compose.yml`, `docker-run.sh`, a GHCR workflow, and Docker deployment documentation.
- Added Docker runtime detection and safeguards: host-level `msf update`, `msf uninstall`, and systemd service install/uninstall commands are disabled inside containers.
- Added bilingual router integration guides for OpenWrt, RouterOS, iKuai, and UniFi DHCP/static-route setup.
- Added a manual release runbook and documented service port allocation in the README.

#### Changed

- Removed live subscription URLs, live share nodes, and YAML node samples from the default initialization config; UI placeholders and MosDNS rule demo data now use inert examples such as `example.invalid`, `198.51.100.0/24`, and `203.0.113.0/24`.
- Kept the Unraid Settings entry as a separate lightweight plugin control page for service enablement, listen address, port, data directory, status, and `Open WebUI`; the full management interface runs in the standalone MSF WebUI.
- Updated the Unraid CA template, `ca_profile.xml`, README text, and `.plg` release notes to describe core hash verification, local-upload semantics, and the Settings page boundary.
- Standardized the project description as a free and open-source DNS & proxy management tool.
- Removed the GitHub Actions release workflow; releases now follow the manual runbook.
- Removed local tooling artifacts such as `.codex` and graphify output from version control and ignored them going forward.

#### Fixed

- Fixed MosDNS cache statistics parsing and display.
- Fixed nftables / policy routing application: old `table inet msf` and duplicate fwmark rules are cleared before apply, routes use `replace`, and cleanup now covers both IPv4 and IPv6 policy routes.
- Fixed daemon stop/restart and shutdown paths to use `ShutdownRuntime` for runtime cleanup.
- Fixed Unraid CA submission scanner detection for the template/profile XML files.
- Fixed database migration compatibility for existing `component_update_info` tables that lack the new verification columns.

## v0.3.0 - 2026-06-08

### 说明

- 项目品牌与工程标识从 `msm-free` / `MSM Free` 迁移为 `msf` / `MSF Free`，GitHub 仓库发布路径切换到 `scoltzero/msf`。
- Linux v0.2.2 用户可通过原有 WebUI 自更新入口升级：发布包继续提供 `msm-free-linux-amd64.tar.gz` 与 `msm-free-linux-arm64.tar.gz` 兼容副本，内容与新 `msf-*` 包逐字节一致。

### 新增

- 新增 `msf migrate` 一次性迁移命令，支持迁移旧数据目录、数据库文件、`update_info` 组件键、`msm_manual` Mihomo provider、旧 PID/日志文件与旧 nftables 表。
- Linux 安装脚本默认安装到 `/opt/msf`、`msf.service` 和 `/usr/local/bin/msf`，并保留 `/usr/local/bin/msm` CLI 兼容别名。
- 新增 `msf` Unraid 插件包与 CA 元数据，安装路径切换为 `/mnt/user/appdata/msf`、`rc.msf` 和 `/usr/local/emhttp/plugins/msf`。

### 修复

- 修复改名后发布链路和系统自更新资源名匹配问题，确保新旧 Linux 包名同时发布并生成校验文件。
- 修复 Unraid/fnOS 环境下网页自更新入口可能使用 Linux systemd 安装流程的问题，改为提示通过对应应用/插件管理入口升级。
- 修复前端标题、初始化向导、登录页、导航、storage key 和 API token 前缀中的旧 `MSM`/`msm` 标识残留。

## v0.2.2 - 2026-06-05

### 说明

- 这个版本意味着 `msm-free` 的主要功能已经基本稳定，初始化、MosDNS/Mihomo 管理、代理规则、更新与发布链路进入可持续迭代状态。

### 新增

- 新增 Mihomo 自定义配置能力：支持导入、新建、保存、命名、覆盖和应用用户配置；运行时仍以 `configs/mihomo/config.yaml` 作为启动文件，用户配置统一保存在 `configs/mihomo/user_configs`。
- 新增 Mihomo 代理分组、代理供应商、规则集和规则的可视化管理入口，WebUI 可读取并展示用户自定义的 `proxy-groups`、`proxy-providers`、`rule-providers` 和 `rules`。
- 新增组件本地上传安装能力：MosDNS、Mihomo 支持上传原始二进制、`.tar.gz`、`.zip`；Zashboard 支持上传 zip 包，方便网络困难时离线安装核心和 UI。
- 新增 Mihomo 配置编辑器的 CodeMirror 6 实现，支持 YAML 高亮、行号、可见光标、撤销重做和搜索快捷键。

### 修复

- 修复 ARM64 发布包下载组件时可能拉取非对应架构核心的问题；ARM64 系统会下载并校验 ARM64 版本的 MosDNS/Mihomo。
- 修复初始化 6 步完成后的核心下载流程，成功后再进入登录；失败时停留在下载页并提示可登录后到系统设置手动下载。
- 修复 Mihomo 代理节点展开后只能点击小圆圈切换的问题，现在整张节点卡片可选择，延迟按钮仍只触发测速。
- 修复代理节点列表中少量节点自适应撑满整行导致视觉不稳定的问题，节点卡片宽度保持稳定。
- 修复 MosDNS 客户端识别来源展示，ARP 与 MosDNS 查询日志来源可合并显示。
- 修复 Mihomo 配置文件列表展示逻辑，隐藏内部启动文件 `config.yaml`，只展示用户可管理的配置文件。
- 修复系统更新中 MSM 自身更新缺少“安装并重启”操作的问题，下载完成后可在 WebUI 触发安装并重启。
- 修复 MSM 自身更新下载未明确走 GitHub 加速的问题，WebUI 会展示实际加速后的下载地址。
- 修复命令行 `msm update` 不读取初始化 GitHub 代理/加速配置的问题，CLI 更新现在复用后端下载器和下载设置。

## v0.2.1 - 2026-06-04

### 修复

- 复刻原版 MSM 首次初始化向导，恢复 6 步初始化流程和原版视觉结构，并继续接入现有初始化 API。
- 修复订阅保存格式，前端按 `名称|URL` 换行提交，后端兼容旧格式并拒绝空 URL、`[]` 和非法协议，避免 Mihomo provider 出现 `unsupported protocol scheme ""`。
- 修复初始化页自定义节点输入，手动添加的节点会生成 `proxy_providers/msm_manual.yaml`，并作为 Mihomo 本地文件型供应商 `msm_manual` 注册。
- 补充常见手动节点分享链接解析，支持 `ss`、`ssr`、`vmess`、`vless`、`trojan`、`hysteria`、`hysteria2`、`tuic` 的基础转换。
- 修复初始化配置参数页 DNS 与 IPv6 滑动按钮偏移问题。
- 修复 GitHub 下载加速初始化配置，恢复原版勾选框样式；勾选后可填写 HTTP、HTTPS、SOCKS5 代理或 GitHub 加速前缀。
- 修复下载器读取 SOCKS5 代理配置，GitHub 组件下载代理不再只支持 HTTP/HTTPS。
- 校准 MosDNS 代理模式语义：关闭模式默认全部可访问外网；白名单模式仅名单内可访问外网；黑名单模式仅名单内不可访问外网。
- 修复 Mihomo 代理节点页在 13 寸 MacBook 宽度下的节点卡片自适应布局，减少内容挤压。
- 修复左侧导航栏点击底部菜单后滚动位置跳回顶部的问题。
- 修复系统更新页“可更新”误判，只以后端明确返回的 `has_update` 为准。
- 接通更新配置页的自动检查、检查间隔、自动下载、更新通知和升级方式保存回显。

### 暂缓

- 完全自定义 Mihomo `config.yaml` 模式暂未进入本版本。
- 自定义代理分组、规则集、在线 ruleset 的可视化管理暂未进入本版本。
- 升级时保护用户自定义 Mihomo `proxy-groups`、`rule-providers`、`rules` 的完整策略暂未进入本版本。
