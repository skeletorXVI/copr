# zen-browser

Zen is repackaged from the official `zen.linux-x86_64.tar.xz` GitHub release asset. Initial pinning verifies GitHub's asset SHA256; every later build re-downloads the asset and verifies it against the committed `sources` checksum before creating the SRPM. The application lives in `/usr/lib64/zen-browser`, with `/usr/bin/zen-browser` as its launcher. Upstream private libraries do not advertise system-wide RPM provides, and their internal requirements are filtered without suppressing system-library requirements.

`distribution/policies.json` disables application self-updates. The policy does not disable extension updates or browser security services. RPM upgrades own the application files; user profiles are outside the package.

Fedora's automatic stripping is disabled for the precompiled bundle. One targeted fix removes an invalid literal `$` RUNPATH from upstream's `libonnxruntime.so`; Fedora's RPATH checks remain enabled.

Because repackaging a prebuilt binary is cheap, this package sets `Verify`, so a pull request that touches it rebuilds the binary RPM, installs it and runs the named command.
