# msm-free

`msm-free` is an open-source reimplementation of the MSM management experience for the `mssb`-style MosDNS + Mihomo stack.

Current target:

- x86_64 Linux first.
- Mihomo is the only proxy core in scope for the first version.
- MosDNS, Mihomo, nftables transparent proxy, setup wizard, users, config history, logs and update APIs are implemented as open functionality.
- The generated runtime now follows the mssb MosDNS + Mihomo split-flow layout: MosDNS `:53`, Mihomo DNS `:6666`, fake-ip `28.0.0.0/8`, TProxy `7896`, redirect `7877`, plus the MosDNS-side `2222/3333/4444/5656/7777/8888/9099` ports.
- Sing-box is intentionally out of scope for the first version.
- Unraid plugin packaging is included as a second deployment target.

## Run Locally

```bash
go run ./cmd/msm-free serve -c ./data -p 7777
```

Then open `http://localhost:7777`.

## Install Linux x86_64 Release

```bash
curl -L -o msm-free-linux-amd64.tar.gz \
  https://github.com/scoltzero/msm-free/releases/download/v0.1.9/msm-free-linux-amd64.tar.gz
tar -xzf msm-free-linux-amd64.tar.gz -C /tmp
sudo /tmp/msm-free-*-linux-amd64/install.sh
```

The installer creates `/usr/local/bin/msm-free`, registers the compatibility command `/usr/local/bin/msm`, initializes `/opt/msm-free`, installs a systemd service, and starts the WebUI on port `7777`.

Open `http://<server-ip>:7777` and complete the setup wizard. After setup, `msm-free` persists the expected runtime state and restores Mihomo, MosDNS and nftables on subsequent starts unless the user explicitly stops or clears them.

Stop and uninstall:

```bash
sudo msm stop
sudo msm restart
msm logs --lines 200 mosdns
msm doctor
sudo msm update
sudo msm uninstall
sudo msm uninstall --purge
```

`uninstall` removes the systemd service and `/usr/local/bin/msm-free`. It keeps `/opt/msm-free` unless `--purge` is provided.

## Install Unraid Plugin

In Unraid WebGUI, open **Plugins / Install Plugin**, paste this URL, and install:

```text
https://raw.githubusercontent.com/scoltzero/msm-free/main/msm-free.plg
```

After installation, open **Settings / MSM Free**, then open the WebUI and complete the setup wizard.

On a fresh install, before setup exists, the plugin starts only the `msm-free` management WebUI. After setup is completed, `msm-free` restores enabled Mihomo, MosDNS and nftables state on subsequent starts.

Persistent Unraid data defaults to `/mnt/user/appdata/msm-free`.

## Build From Source

```bash
make build
make unraid VERSION=0.1.9 UNRAID_VERSION=0.1.9 GITHUB_REPO=scoltzero/msm-free RELEASE_TAG=v0.1.9
```

The generated artifacts are:

- `dist/msm-free-linux-amd64.tar.gz`
- `dist/unraid/msm-free-<version>-x86_64-1.txz`
- `msm-free.plg`

## Runtime Layout

The data directory defaults to `/opt/msm-free` on generic Linux and `/mnt/user/appdata/msm-free` on Unraid. It contains:

- `configs/mosdns`
- `configs/mihomo`
- `configs/network`
- `data/binaries`
- `logs`
- `database`
- `backups`

## Notes

This project does not contain MSM closed-source backend code. The UI and API behavior are reimplemented from public documentation, mssb behavior, and local compatibility observations.

## Acknowledgements

`msm-free` is a non-commercial open reimplementation of the MSM-style management experience. It is based on MSM's user-facing appearance and reconstructed around the mssb-style MosDNS + Mihomo workflow.

Thanks to:

- `msm9527/msm-wiki`, used as the public reference for the MSM management experience.
- `baozaodetudou/mssb`, used as the public reference for the MosDNS + Mihomo backend behavior.

This project is not affiliated with the upstream MSM or mssb projects.
