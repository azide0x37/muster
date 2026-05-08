# Muster Compliance: dvd-ingester

This example applies the root Muster contract to a lawful DVD ingest appliance.

Architecture:

`udev -> systemd one-shot service -> rip script -> local work dir -> atomic NAS publish -> eject`

The udev rule must only request a systemd unit with `SYSTEMD_WANTS`. It must not perform ripping directly.

Configuration lives in `/etc/dvd-ingester/dvd-ingester.env`. Installed code lives in `/opt/dvd-ingester/releases/<version>`, with `/opt/dvd-ingester/current` pointing at the active release.
