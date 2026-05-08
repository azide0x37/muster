# Release Process

Muster release artifacts are built into `dist/`.

Required artifacts:

- `bt-audio-gateway-<version>.tar.gz`
- `bt-audio-gateway-<version>.tar.gz.sha256`
- `install.sh`
- `manifest.json`

Build:

```sh
make package
```

The release manifest records the version, artifact name, SHA256, and expected install/update entry points. The public install path should prefer a tagged GitHub release asset:

```sh
curl -fsSL https://github.com/azide0x37/bt-audio-gateway/releases/latest/download/install.sh | sudo sh
```

The raw `main` installer may exist for development, but stable installs and autoupdates should consume immutable release artifacts.
