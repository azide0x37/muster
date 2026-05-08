# Muster Compliance: bt-audio-gateway

This example applies the root Muster contract to a Snapcast-to-Bluetooth audio gateway.

Architecture:

`Snapserver -> Raspberry Pi snapclient -> PipeWire/Pulse sink -> Bluetooth speaker`

Snapcast is the network transport. PipeWire is the local audio graph. Bluetooth is only the local last hop. systemd owns restart, routing, health checks, and update polling.

Configuration lives in `/etc/bt-audio-gateway/bt-audio-gateway.env`. Installed code lives in `/opt/bt-audio-gateway/releases/<version>`, with `/opt/bt-audio-gateway/current` pointing at the active release.
