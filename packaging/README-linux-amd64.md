# msm-free linux-amd64 安装包

这个压缩包包含 `msm-free` x86_64 Linux 二进制、systemd 安装脚本和卸载脚本。

## 安装

```sh
tar -xzf msm-free-linux-amd64.tar.gz
cd msm-free-*-linux-amd64
sudo ./install.sh
```

默认路径：

- 二进制：`/usr/local/bin/msm-free`
- 兼容命令：`/usr/local/bin/msm`
- 数据目录：`/opt/msm-free`
- WebUI：`http://<server-ip>:7777`
- systemd 服务：`msm-free`

自定义安装：

```sh
sudo ./install.sh --data-dir /opt/msm-free --host 0.0.0.0 --port 7777
```

## 停止

systemd 停止：

```sh
sudo systemctl stop msm-free
```

也可以直接使用二进制命令：

```sh
sudo msm stop
```

`stop` 会优雅停止 `msm-free` 管理进程，并由管理进程停止托管的 MosDNS 和 Mihomo 子进程。超时仍未退出时可以使用：

```sh
sudo msm stop --timeout 20s --force
```

常用 CLI：

```sh
msm status
msm restart
msm logs msm
msm logs --lines 200 mosdns
msm logs --lines 200 mihomo
msm doctor
msm license status
sudo msm update
```

## 升级

重新运行新版本安装包中的安装脚本即可。安装脚本会覆盖二进制并重启服务，但保留现有数据目录。

```sh
sudo ./install.sh
```

## 卸载

推荐直接使用二进制自带卸载命令：

```sh
sudo msm uninstall
```

也可以使用压缩包内的卸载脚本：

```sh
sudo ./uninstall.sh
```

默认卸载只删除 systemd 服务和 `/usr/local/bin/msm-free`，保留 `/opt/msm-free`。如需彻底删除数据目录：

```sh
sudo msm uninstall --purge
sudo ./uninstall.sh --purge
```

## 校验

```sh
sha256sum -c SHA256SUMS
systemctl status msm-free
```
