# msm-free Unraid Plugin

This directory builds the Unraid plugin artifacts for `msm-free`.

Generated files:

- `dist/unraid/msm-free.plg`
- `dist/unraid/msm-free-<version>-x86_64-1.txz`
- `msm-free.plg`

Build:

```bash
make unraid VERSION=0.1.8 UNRAID_VERSION=0.1.8 GITHUB_REPO=<owner>/msm-free RELEASE_TAG=v0.1.8
```

Publish the generated `.txz` package to the GitHub release named by `RELEASE_TAG`, then commit the generated root `msm-free.plg`.

Example:

```bash
gh release create v0.1.8 \
  dist/unraid/msm-free-0.1.8-x86_64-1.txz \
  dist/msm-free-linux-amd64.tar.gz \
  --title "msm-free v0.1.8" \
  --notes "Initial msm-free x86_64 and Unraid plugin release."
```

Install URL for Unraid:

```text
https://raw.githubusercontent.com/<owner>/msm-free/main/msm-free.plg
```

## Runtime Behavior

- The plugin installs the `msm-free` binary into `/usr/local/emhttp/plugins/msm-free/bin/msm-free`.
- The WebGUI control script is `/etc/rc.d/rc.msm-free`.
- Persistent config is `/boot/config/plugins/msm-free/msm-free.cfg`.
- Persistent application data defaults to `/mnt/user/appdata/msm-free`.
- On a fresh install, before setup exists, the plugin starts only the `msm-free` management WebUI. After setup is completed, `msm-free` restores enabled Mihomo, MosDNS and nftables state on subsequent starts.
- If the data directory is under `/mnt/user`, the rc script waits until the array user share path is available.

The MosDNS, Mihomo, and nftables behavior is controlled by `msm-free` itself after the user completes the setup wizard or changes service/network state in the WebUI.
