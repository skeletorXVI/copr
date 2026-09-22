// The Libre Graph Qt client is a shared dependency of OpenCloud Desktop.
// Upstream publishes version tags without GitHub Releases.
package main

import (
	"copr.local/automation/internal/core"
	"copr.local/automation/internal/core/github"
)

const (
	repo = "opencloud-eu/libre-graph-api-cpp-qt-client"
	name = "libre-graph-api-cpp-qt-client"
)

func main() {
	core.Run(core.Definition{
		Project:  "opencloud-desktop",
		Name:     name,
		Upstream: "https://github.com/opencloud-eu/libre-graph-api-cpp-qt-client",
		Policy:   core.ReviewAll{},
		Chroots: []string{
			"fedora-43-x86_64", "fedora-44-x86_64",
			"fedora-45-x86_64", "fedora-rawhide-x86_64",
		},
		Releases: github.HighestTag{Repo: repo},
		Spec:     spec,
	})
}

func spec(r core.Release) core.Spec {
	archive := name + "-" + r.Version + ".tar.gz"
	return core.Spec{
		Summary: "Qt client for the Libre Graph API",
		License: "Apache-2.0",
		URL:     "https://github.com/" + repo,
		Sources: []core.Source{{Name: archive, URL: github.TarballURL(repo, "v"+r.Version, archive)}},

		BuildRequires: []string{"cmake", "gcc-c++", "qt6-qtbase-devel", "zlib-devel"},
		Description:   []string{"C++ and Qt bindings for the Libre Graph collaboration API."},
		SubPackages: []core.SubPackage{{
			Suffix:      "devel",
			Summary:     "Development files for the Libre Graph Qt client",
			Requires:    []string{`%{name}%{?_isa} = %{version}-%{release}`, "qt6-qtbase-devel"},
			Description: []string{"Headers and CMake metadata for the Libre Graph Qt client."},
			Files: []string{
				`%{_includedir}/OpenAPI/`,
				`%{_libdir}/libLibreGraphAPI.so`,
				`%{_libdir}/cmake/LibreGraphAPI/`,
			},
		}},
		Prep: []string{"%setup -q"},
		// Upstream keeps the library in a client/ subdirectory.
		Build: []string{
			"%cmake -S client -DBUILD_SHARED_LIBS=ON -DCMAKE_POLICY_VERSION_MINIMUM=3.5",
			"%cmake_build",
		},
		Install: []string{"%cmake_install"},
		Files: []string{
			`%license LICENSE`,
			`%{_libdir}/libLibreGraphAPI.so.*`,
		},
	}
}
