// OpenCloud Desktop is the synchronization client; the file-manager
// integrations are packaged separately in this same COPR project.
package main

import (
	"copr.local/automation/internal/core"
	"copr.local/automation/internal/core/github"
)

const repo = "opencloud-eu/desktop"

func main() {
	core.Run(core.Definition{
		Project:  "opencloud-desktop",
		Name:     "opencloud-desktop",
		Upstream: "https://github.com/opencloud-eu/desktop",
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
	archive := "desktop-" + r.Version + ".tar.gz"
	return core.Spec{
		Summary: "Desktop synchronization client for OpenCloud",
		License: "GPL-2.0-or-later",
		URL:     "https://opencloud.eu",
		Sources: []core.Source{{Name: archive, URL: github.TarballURL(repo, "v"+r.Version, archive)}},

		BuildRequires: []string{
			"cmake", "gcc-c++", "extra-cmake-modules", "qt6-qtbase-devel >= 6.8",
			"qt6-qtdeclarative-devel", "qt6-qttools-devel", "qtkeychain-qt6-devel",
			"cmake(KDSingleApplication-qt6)", "libre-graph-api-cpp-qt-client-devel >= 1.0.4",
			"sqlite-devel", "zlib-devel", "desktop-file-utils",
		},
		Requires: []string{"qt6-qtdeclarative", "hicolor-icon-theme"},
		Recommends: []string{
			"(opencloud-desktop-dolphin if dolphin)",
			"(opencloud-desktop-nautilus if nautilus)",
		},
		Description: []string{
			"Synchronize files between an OpenCloud server and your desktop. Updates are",
			"managed by the system package manager. Install the optional Dolphin or Nautilus",
			"package for file-manager status icons and context-menu actions.",
		},
		SubPackages: []core.SubPackage{{
			Suffix:      "devel",
			Summary:     "Development files for OpenCloud Desktop",
			Requires:    []string{`%{name}%{?_isa} = %{version}-%{release}`},
			Description: []string{"Headers and CMake metadata for the OpenCloud synchronization library."},
			Files: []string{
				`%{_includedir}/OpenCloud/`,
				`%{_libdir}/libOpenCloudLibSync.so`,
				`%{_libdir}/libOpenCloudResources.so`,
				`%{_libdir}/cmake/OpenCloud/`,
			},
		}},
		Prep: []string{"%setup -q -n desktop-%{version}"},
		// The RPM owns updates, and the experimental VFS plugins stay off.
		Build: []string{
			`%cmake -DBUILD_TESTING=OFF -DWITH_AUTO_UPDATER=OFF \`,
			`    -DWITH_UPDATE_NOTIFICATION=OFF -DWITH_CRASHREPORTER=OFF \`,
			"    -DVIRTUAL_FILE_SYSTEM_PLUGINS=off",
			"%cmake_build",
		},
		Install: []string{"%cmake_install"},
		Check:   []string{"desktop-file-validate %{buildroot}%{_datadir}/applications/*.desktop"},
		Files: []string{
			`%license COPYING*`,
			`%{_bindir}/opencloud`,
			`%{_bindir}/opencloudcmd`,
			`%{_libdir}/libOpenCloud*.so.*`,
			`%{_libdir}/libOpenCloudGui.so`,
			`%{_libdir}/qt6/plugins/OpenCloud_vfs_off.so`,
			`%{_libdir}/qt6/qml/eu/OpenCloud/`,
			`%{_datadir}/applications/*.desktop`,
			`%{_datadir}/metainfo/*.xml`,
			`%{_datadir}/icons/hicolor/*/apps/opencloud.png`,
		},
	}
}
