# Codex Task: Implement dvd-ingester With Muster

Build a Muster-compliant Linux appliance that ingests DVDs on insert.

The architecture is:

`udev -> systemd one-shot service -> shell/Python rip script -> local work dir -> NAS -> eject`

Important constraints:

- Do not run long work directly from udev.
- udev must use `TAG+="systemd"` and `ENV{SYSTEMD_WANTS}`.
- Fingerprint the disc before ripping.
- Store success records locally.
- Write final files atomically to NAS.
- Eject the tray on completion or failure.
- Leave transcoding/library management to Plex, Jellyfin, NAS, or a desktop system.
- Keep the implementation lawful and do not add copy-protection bypass behavior.
