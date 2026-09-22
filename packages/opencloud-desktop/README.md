# opencloud-desktop

The five packages in this project share one COPR project so their build dependencies resolve within it. The split follows the [greve/opencloud-desktop COPR reference](https://copr.fedorainfracloud.org/coprs/greve/opencloud-desktop/), using current upstream source releases rather than its uploaded RPMs.

Desktop requires Libre Graph; Dolphin and Nautilus require Desktop and integration resources. Those requirements are spec `Requires`, resolved by COPR inside this project. Only the packages a change actually touches are rebuilt, and they are submitted in parallel, so a consumer can be built against the prerequisite currently in the project rather than one being rebuilt alongside it; its next build picks the new one up.

Each package's own notes are in its directory: [`opencloud-desktop`](opencloud-desktop/README.md), [`libre-graph-api-cpp-qt-client`](libre-graph-api-cpp-qt-client/README.md), [`opencloud-desktop-integration-resources`](opencloud-desktop-integration-resources/README.md), [`opencloud-desktop-dolphin`](opencloud-desktop-dolphin/README.md), [`opencloud-desktop-nautilus`](opencloud-desktop-nautilus/README.md).

## COPR project

Create a COPR project named `opencloud-desktop` under the owner set in the `COPR_OWNER` repository variable. Its name is both this directory and the `Project` field of all five package definitions; they must agree.

| Setting            | Value                                                                               |
| ------------------ | ----------------------------------------------------------------------------------- |
| Chroots            | `fedora-43-x86_64`, `fedora-44-x86_64`, `fedora-45-x86_64`, `fedora-rawhide-x86_64` |
| Build dependencies | Fedora, plus this project's own repository                                          |
| Network            | Not needed; all five build offline                                                  |

Enabling the project's own repository as a build dependency source is what lets Desktop find Libre Graph, and the shell integrations find Desktop — COPR's normal project behavior, but these packages do not build without it.

A chroot a package names in `Chroots` but that does not exist on the COPR service causes a visible failure rather than being silently skipped, so create every one of them. Do not configure packages by hand: uploading a source RPM creates its COPR package on first use. Leave COPR's own webhook rebuilds off, because this repository decides what to rebuild.
