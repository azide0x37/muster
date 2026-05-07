# Muster Example: bt-audio-gateway

A Muster reference service repo for bridging Snapcast audio to a local Bluetooth speaker through PipeWire/Pulse on a Raspberry Pi.

Architecture:

```text
Home Assistant / Music Assistant
  -> Snapserver / Snapcast stream
  -> Raspberry Pi snapclient
  -> PipeWire Pulse-compatible sink
  -> Bluetooth speaker
```

Bluetooth is only the local last hop. Snapcast is the network audio transport. PipeWire is the audio plumbing. systemd owns lifecycle, restart, health checks, and update polling.

## Install

Preferred release install:

```sh
curl -fsSL https://github.com/<owner>/<repo>/releases/latest/download/install.sh | sudo sh
```

Development install from a checkout:

```sh
sudo ./bin/install.sh
```

Then edit:

```sh
sudoedit /etc/bt-audio-gateway/bt-audio-gateway.env
```

Enable the snapclient instance for your audio user:

```sh
sudo systemctl enable --now bt-audio-watch.service
sudo systemctl enable --now snapclient-bt@pi.service
sudo systemctl enable --now bt-audio-doctor.timer bt-audio-update.timer
```

Replace `pi` with the configured `AUDIO_USER`.

## Config

Config lives at `/etc/bt-audio-gateway/bt-audio-gateway.env`.

```sh
BT_MAC=AA:BB:CC:DD:EE:FF
AUDIO_USER=pi
SNAPSERVER_HOST=homeassistant.local
SNAPSERVER_PORT=1704
SNAPCLIENT_ID=bt-kitchen-speaker
BT_VOLUME=80%
SCAN_SECONDS=12
SLEEP_SECONDS=8
AUTOUPDATE=1
UPDATE_CHANNEL=latest
UPDATE_MANIFEST_URL='https://github.com/<owner>/<repo>/releases/latest/download/manifest.json'
```

## systemd Units

- `bt-audio-watch.service`: keeps the Bluetooth speaker trusted, discovered, connected, and routed.
- `snapclient-bt@.service`: runs `snapclient` as the audio user and targets the PipeWire Pulse server.
- `bt-audio-doctor.service`: runs health checks.
- `bt-audio-doctor.timer`: periodic health check.
- `bt-audio-update.service`: checks for release updates.
- `bt-audio-update.timer`: periodic autoupdate polling with jitter.

## Update And Rollback

`bt-audio-update.timer` runs `bin/update.sh`. Updates are skipped when `AUTOUPDATE=0`.

The updater fetches a release manifest, downloads the release archive, verifies SHA256, installs it into `/opt/bt-audio-gateway/releases/<version>`, switches `/opt/bt-audio-gateway/current`, restarts services, and runs `doctor.sh`. If the health check fails, it switches back to the prior release and restarts services.

## Troubleshooting

Watch the Bluetooth reconnect loop:

```sh
sudo journalctl -u bt-audio-watch.service -f
```

Check the configured speaker:

```sh
bluetoothctl info AA:BB:CC:DD:EE:FF
```

Check PipeWire/Pulse sinks:

```sh
AUDIO_USER=pi
UID_NUM="$(id -u "$AUDIO_USER")"
sudo -u "$AUDIO_USER" \
  XDG_RUNTIME_DIR="/run/user/$UID_NUM" \
  PULSE_SERVER="unix:/run/user/$UID_NUM/pulse/native" \
  pactl list short sinks
```

Run the bundled health check:

```sh
sudo /opt/bt-audio-gateway/current/bin/doctor.sh
```

## Home Assistant And Music Assistant

Use Music Assistant's Snapcast provider with either its built-in Snapserver or an external Snapserver. Home Assistant can expose Music Assistant players as `media_player` entities. If using an external Snapserver, allow the Snapcast control and stream ports required by your Music Assistant/Snapcast configuration.

## Development

```sh
make test
make package
```

Run a staged idempotence check without touching the host:

```sh
MUSTER_ROOT="$(mktemp -d)" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh
MUSTER_ROOT="$MUSTER_ROOT" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh
```

## Muster Self-Certification

| Requirement | Status | Evidence |
|---|---:|---|
| systemd owns lifecycle | PASS | `systemd/*.service` |
| timer-based update polling exists | PASS | `systemd/bt-audio-update.timer` |
| idempotent installer exists | PASS | `bin/install.sh` and `tests/test_installer_idempotent.sh` |
| config lives in `/etc/<project>/` | PASS | `/etc/bt-audio-gateway/bt-audio-gateway.env` |
| versioned install path exists | PASS | `/opt/bt-audio-gateway/releases/<version>` |
| rollback path exists | PASS | `bin/update.sh` |
| doctor check exists | PASS | `bin/doctor.sh` |
| release artifacts buildable | PASS | `make package` |
| tests runnable | PASS | `make test` |
