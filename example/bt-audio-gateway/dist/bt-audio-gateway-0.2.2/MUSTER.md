# Muster Compliance: bt-audio-gateway

This example applies the root Muster contract to a Snapcast-to-Bluetooth audio gateway.

Architecture:

`Snapserver -> Raspberry Pi snapclient -> PipeWire/Pulse sink -> Bluetooth speaker`

Snapcast is the network transport. PipeWire is the local audio graph. Bluetooth is only the local last hop. systemd owns restart, routing, health checks, and update polling.

Configuration lives in `/etc/bt-audio-gateway/bt-audio-gateway.env`. Installed code lives in `/opt/bt-audio-gateway/releases/<version>`, with `/opt/bt-audio-gateway/current` pointing at the active release.

## Inspector Projection

`muster.yaml` schema 2 declares the six owned systemd units, configuration,
release pointer, doctor evidence, and a partial
`T2R1.bluetooth-audio-gateway` tree. `muster.lock.json` freezes that graph at
MPL commit `ea6d02aaa6860e5102a760473b2ffe9b90d13c75`.

The T2R1 claim remains explicitly partial: device binding and reconnection are
implemented, while the complete R4 state ledger and outward local-control
surface are not overstated. Runtime doctor evidence is written atomically to
`/run/muster/bt-audio-gateway/observations/doctor.json`.

The installer bootstraps the shared core only when necessary and owns only
`/etc/muster/implementations.d/bt-audio-gateway.json`. Its uninstaller does not
remove `/opt/muster` or `/usr/local/bin/muster`.
