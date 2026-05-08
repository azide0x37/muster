# Security

`dvd-ingester` installs root-owned services and a udev rule that reacts to optical media insertion. Treat installer, updater, udev, and rip-script changes as privileged-code changes.

This example is for lawful, non-copy-protected discs or discs your tools can read without bypassing protection. Do not add copy-protection bypass behavior to this project.
