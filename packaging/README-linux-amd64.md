# msm-free linux-amd64 package

This package contains the `msm-free` x86_64 Linux binary and systemd installer.

## Install

```sh
tar -xzf msm-free-linux-amd64.tar.gz
cd msm-free-*-linux-amd64
sudo ./install.sh
```

Default paths:

- Binary: `/usr/local/bin/msm-free`
- Data directory: `/opt/msm-free`
- Web UI: `http://<server-ip>:7777`
- systemd service: `msm-free`

Custom install:

```sh
sudo ./install.sh --data-dir /opt/msm-free --host 0.0.0.0 --port 7777
```

## Upgrade

Run the new package installer again. It overwrites the binary and restarts the service, but keeps the existing data directory.

## Uninstall

```sh
sudo ./uninstall.sh
```

Remove data as well:

```sh
sudo ./uninstall.sh --purge
```

## Verify

```sh
sha256sum -c SHA256SUMS
systemctl status msm-free
```

