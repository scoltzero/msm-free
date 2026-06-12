# msf Unraid Plugin

This directory builds the Unraid plugin artifacts for `msf`.

Generated files:

- `dist/unraid/msf.plg`
- `dist/unraid/msf-<version>-x86_64-1.txz`
- `msf.plg`

Build:

```bash
make unraid VERSION=0.3.2 UNRAID_VERSION=0.3.2 GITHUB_REPO=scoltzero/msf RELEASE_TAG=v0.3.2
```

Publish the generated `.txz` package and `.plg` file to the GitHub release named by `RELEASE_TAG`, then commit the generated root `msf.plg` when you want a branch-based install URL.

Example:

```bash
gh release create v0.3.2 \
  dist/unraid/msf-0.3.2-x86_64-1.txz \
  dist/unraid/msf.plg \
  dist/msf-linux-amd64.tar.gz \
  dist/msf-linux-amd64.tar.gz.sha256 \
  dist/msf-linux-arm64.tar.gz \
  dist/msf-linux-arm64.tar.gz.sha256 \
  --title "v0.3.2" \
  --notes-file /tmp/msf-v0.3.2-release-notes.md
```

Recommended install URL for the v0.3.2 release:

```text
https://github.com/scoltzero/msf/releases/download/v0.3.2/msf.plg
```

Branch install URL, only after the generated root `msf.plg` has been committed to that branch:

```text
https://raw.githubusercontent.com/scoltzero/msf/<branch>/msf.plg
```

## Runtime Behavior

- The plugin installs the `msf` binary into `/usr/local/emhttp/plugins/msf/bin/msf`.
- The plugin registers the compatibility command `/usr/local/bin/msf`.
- The WebGUI control script is `/etc/rc.d/rc.msf`.
- Persistent config is `/boot/config/plugins/msf/msf.cfg`.
- Persistent application data defaults to `/mnt/user/appdata/msf`.
- The Settings page is a lightweight Unraid plugin control page only: enablement, listen host/port, data directory, status, and Open WebUI.
- The full management interface runs in the separate msf WebUI.
- On a fresh install, before setup exists, the plugin starts only the `msf` management WebUI. After setup is completed, `msf` restores enabled Mihomo, MosDNS and nftables state on subsequent starts.
- If the data directory is under `/mnt/user`, the rc script waits until the array user share path is available.
- Online MosDNS, Mihomo, and Zashboard downloads must verify the SHA-256 digest supplied by the GitHub Release API asset before install. Local uploads are user-supplied and are marked as `local-upload`.

The MosDNS, Mihomo, and nftables behavior is controlled by `msf` itself after the user completes the setup wizard or changes service/network state in the WebUI.

## Stop and Uninstall

Stop the Unraid service without removing files:

```bash
/etc/rc.d/rc.msf stop
msf stop --config /mnt/user/appdata/msf
```

Restart it:

```bash
/etc/rc.d/rc.msf restart
msf restart --config /mnt/user/appdata/msf
```

Useful CLI commands:

```bash
msf status --config /mnt/user/appdata/msf
msf logs --config /mnt/user/appdata/msf --lines 200 mosdns
msf logs --config /mnt/user/appdata/msf --lines 200 mihomo
msf doctor --config /mnt/user/appdata/msf
msf license status
```

Do not use `msf update` or `msf uninstall` on Unraid. Updates and removal must go through the Unraid plugin manager so the `.plg` package state stays consistent.

Remove the plugin from the Unraid WebGUI plugin page. The plugin remove hook stops the rc service and removes the package files, but it keeps the application data directory by default:

```text
/mnt/user/appdata/msf
```

Delete that directory manually only when you want to remove all configuration, database, logs, downloaded components, and backups.
