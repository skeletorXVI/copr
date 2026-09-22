// Dolphin integration: overlay icons and context-menu actions for KDE.
package main

import (
	"copr.local/automation/internal/core"
	"copr.local/automation/internal/core/github"
)

const repo = "opencloud-eu/desktop-shell-integration-dolphin"

func main() {
	core.Run(core.Definition{
		Project:  "opencloud-desktop",
		Name:     "opencloud-desktop-dolphin",
		Upstream: "https://github.com/opencloud-eu/desktop-shell-integration-dolphin",
		Policy:   core.ReviewAll{},
		Chroots: []string{
			"fedora-43-x86_64", "fedora-44-x86_64",
			"fedora-45-x86_64", "fedora-rawhide-x86_64",
		},
		Releases: github.Latest{Repo: repo},
		Spec:     spec,
	})
}

func spec(r core.Release) core.Spec {
	archive := "desktop-shell-integration-dolphin-" + r.Version + ".tar.gz"
	return core.Spec{
		Summary: "OpenCloud integration for Dolphin",
		License: "GPL-2.0-or-later",
		URL:     "https://github.com/" + repo,
		Sources: []core.Source{{Name: archive, URL: github.TarballURL(repo, "v"+r.Version, archive)}},

		BuildRequires: []string{
			"cmake", "gcc-c++", "extra-cmake-modules", "qt6-qtbase-devel",
			"kf6-kcoreaddons-devel", "kf6-kio-devel", "kf6-kbookmarks-devel",
			"opencloud-desktop-integration-resources >= 1.0.0",
		},
		Requires: []string{
			"dolphin", "opencloud-desktop",
			"opencloud-desktop-integration-resources >= 1.0.0",
		},
		Description: []string{"Sync status overlays and context-menu actions for OpenCloud in KDE Dolphin."},
		Prep:        []string{"%setup -q -n desktop-shell-integration-dolphin-%{version}"},
		Build: []string{
			"%cmake",
			"%cmake_build",
		},
		Install: []string{"%cmake_install"},
		Files: []string{
			`%license COPYING`,
			`%{_libdir}/libopenclouddolphinpluginhelper.so`,
			`%{_libdir}/qt6/plugins/kf6/overlayicon/openclouddolphinoverlayplugin.so`,
			`%{_libdir}/qt6/plugins/kf6/kfileitemaction/openclouddolphinactionplugin.so`,
		},
	}
}
