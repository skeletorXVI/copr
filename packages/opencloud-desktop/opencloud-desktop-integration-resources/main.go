// Shared icons and CMake discovery metadata for the file-manager integrations.
package main

import (
	"copr.local/automation/internal/core"
	"copr.local/automation/internal/core/github"
)

const repo = "opencloud-eu/desktop-shell-integration-resources"

func main() {
	core.Run(core.Definition{
		Project:  "opencloud-desktop",
		Name:     "opencloud-desktop-integration-resources",
		Upstream: "https://github.com/opencloud-eu/desktop-shell-integration-resources",
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
	archive := "desktop-shell-integration-resources-" + r.Version + ".tar.gz"
	return core.Spec{
		Summary: "Shared icons for OpenCloud file-manager integrations",
		License: "GPL-2.0-or-later",
		URL:     "https://github.com/" + repo,
		Sources: []core.Source{{Name: archive, URL: github.TarballURL(repo, "v"+r.Version, archive)}},

		BuildArch:     "noarch",
		BuildRequires: []string{"cmake", "gcc", "extra-cmake-modules"},
		Requires:      []string{"hicolor-icon-theme"},
		Description:   []string{"Shared status icons and discovery metadata for OpenCloud desktop integrations."},
		Prep:          []string{"%setup -q -n desktop-shell-integration-resources-%{version}"},
		// The metadata is architecture independent, so keep it out of %{_libdir}.
		Build: []string{
			"%cmake -DCMAKE_INSTALL_LIBDIR=share",
			"%cmake_build",
		},
		Install: []string{"%cmake_install"},
		Files: []string{
			`%license COPYING`,
			`%{_datadir}/icons/hicolor/*/apps/*.png`,
			`%{_datadir}/cmake/OpenCloudShellResources/`,
		},
	}
}
