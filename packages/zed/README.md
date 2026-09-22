# zed

The COPR project behind [`zed`](zed/README.md), the only package it holds.

## COPR project

Create a COPR project named `zed` under the owner set in the `COPR_OWNER` repository variable. Its name is both this directory and the `Project` field of the package definition; the two must agree.

| Setting            | Value                                                                                |
| ------------------ | ------------------------------------------------------------------------------------ |
| Chroots            | `fedora-43-x86_64`, `fedora-44-x86_64`, `fedora-45-x86_64`, `fedora-rawhide-x86_64`  |
| Build dependencies | Fedora only                                                                          |
| Network            | Required, and requested per build by the workflow from the package's `Network: true` |

A chroot the package names in `Chroots` but that does not exist on the COPR service causes a visible failure rather than being silently skipped, so create every one of them. Do not configure packages by hand: uploading a source RPM creates its COPR package on first use. Leave COPR's own webhook rebuilds off, because this repository decides what to rebuild.

Zed is a long source build; the workflow allows COPR six hours per build.
