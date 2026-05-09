# Release Process

Build release artifacts:

```sh
make package
```

Artifacts are written to `dist/`:

- `dvd-ingester-<version>.tar.gz`
- `dvd-ingester-<version>.tar.gz.sha256`
- `install.sh`
- `manifest.json`

Publish artifacts as immutable release assets. Autoupdate should consume release manifests and verify SHA256 before switching `/opt/dvd-ingester/current`.
