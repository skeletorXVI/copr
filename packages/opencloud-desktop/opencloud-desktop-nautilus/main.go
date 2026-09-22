// Nautilus integration: sync emblems and context-menu actions for GNOME Files.
package main

import (
	"copr.local/automation/internal/core"
	"copr.local/automation/internal/core/github"
)

const repo = "opencloud-eu/desktop-shell-integration-nautilus"

func main() {
	core.Run(core.Definition{
		Project:  "opencloud-desktop",
		Name:     "opencloud-desktop-nautilus",
		Upstream: "https://github.com/opencloud-eu/desktop-shell-integration-nautilus",
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
	archive := "desktop-shell-integration-nautilus-" + r.Version + ".tar.gz"
	return core.Spec{
		Summary: "OpenCloud integration for Nautilus",
		License: "GPL-2.0-or-later",
		URL:     "https://github.com/" + repo,
		Sources: []core.Source{{Name: archive, URL: github.TarballURL(repo, "v"+r.Version, archive)}},

		BuildArch: "noarch",
		BuildRequires: []string{
			"cmake", "gcc", "opencloud-desktop-integration-resources >= 1.0.0",
		},
		Requires: []string{
			"opencloud-desktop", "opencloud-desktop-integration-resources >= 1.0.0",
			"nautilus-python", "python3-gobject",
		},
		Description: []string{"Sync status emblems and context-menu actions for OpenCloud in GNOME Files."},
		Prep:        []string{"%setup -q -n desktop-shell-integration-nautilus-%{version}"},
		Build: []string{
			"%cmake",
			"%cmake_build",
		},
		Install: []string{
			"%cmake_install",
			"# Upstream installs all three integrations; this package targets Nautilus.",
			"rm -r %{buildroot}%{_datadir}/nemo-python %{buildroot}%{_datadir}/caja-python",
		},
		Files: []string{
			`%license COPYING*`,
			`%{_datadir}/nautilus-python/extensions/syncstate-OpenCloud.py`,
		},
	}
}
