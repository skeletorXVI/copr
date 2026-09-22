// Zed is built from its tagged source release, following upstream's
// instructions for distribution maintainers.
package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"copr.local/automation/internal/core"
	"copr.local/automation/internal/core/github"
)

const repo = "zed-industries/zed"

func tarball(version string) core.Source {
	name := "zed-" + version + ".tar.gz"
	return core.Source{Name: name, URL: github.TarballURL(repo, "v"+version, name)}
}

func main() {
	core.Run(core.Definition{
		Project:  "zed",
		Name:     "zed",
		Upstream: "https://github.com/zed-industries/zed",
		Policy:   core.ReviewAll{},
		Chroots: []string{
			"fedora-43-x86_64", "fedora-44-x86_64",
			"fedora-45-x86_64", "fedora-rawhide-x86_64",
		},
		// Cargo resolves its dependencies and upstream's WebRTC build fetches
		// its native library, so this build cannot run offline.
		Network:  true,
		Releases: github.Latest{Repo: repo},
		Vars:     vars,
		Spec:     spec,
	})
}

// vars reads the Rust version upstream pins, so the spec requires a compiler
// the source actually builds with. It reads the pinned archive rather than the
// upstream branch, so regenerating an old version stays reproducible.
func vars(_ context.Context, version string) (map[string]string, error) {
	path, err := core.SourceArchive("zed", version, tarball(version))
	if err != nil {
		return nil, err
	}
	b, err := core.FileFromTarGz(path, func(name string) bool {
		return strings.Count(name, "/") == 1 && strings.HasSuffix(name, "/rust-toolchain.toml")
	})
	if err != nil {
		return nil, err
	}
	m := regexp.MustCompile(`(?m)^channel\s*=\s*"([0-9]+\.[0-9]+\.[0-9]+)"`).FindSubmatch(b)
	if m == nil {
		return nil, fmt.Errorf("Zed does not pin a stable Rust toolchain")
	}
	return map[string]string{"rust": string(m[1])}, nil
}

func spec(r core.Release) core.Spec {
	return core.Spec{
		Globals: []core.Global{{Name: "debug_package", Value: "%{nil}"}},
		Summary: "High-performance collaborative code editor",
		License: "GPL-3.0-or-later AND Apache-2.0 AND AGPL-3.0-or-later",
		URL:     "https://zed.dev",
		Sources: []core.Source{tarball(r.Version)},

		ExclusiveArch: "x86_64",
		BuildRequires: []string{
			"cargo", "rust >= " + r.Var("rust"), "clang", "lld", "llvm-devel", "gcc-c++",
			"cmake", "make", "git", "curl", "perl-FindBin", "perl-IPC-Cmd",
			"perl-File-Compare", "perl-File-Copy",
			"pkgconfig(alsa)", "pkgconfig(fontconfig)", "pkgconfig(glib-2.0)",
			"pkgconfig(libva)", "pkgconfig(wayland-client)", "pkgconfig(xcb)",
			"pkgconfig(xkbcommon-x11)", "pkgconfig(openssl)", "pkgconfig(libzstd)",
			"pkgconfig(sqlite3)", "gettext", "desktop-file-utils", "patchelf",
		},
		Requires:   []string{"vulkan-loader", "libva", "mesa-libEGL", "xdg-desktop-portal"},
		Recommends: []string{"pipewire", "git"},

		Description: []string{
			"Zed is a fast, collaborative editor built from its tagged source release.",
			"Use your system package manager to update this installation.",
		},
		Prep: []string{
			"%setup -q",
			"# No trailing newline: required for stable-channel credential storage.",
			"printf stable > crates/zed/RELEASE_CHANNEL",
		},
		Build: []string{
			"# COPR must have networking enabled: Cargo.lock pins Rust dependencies, and",
			"# upstream's WebRTC build fetches its native library. See this package's README.",
			`export CARGO_HOME="$PWD/.cargo-home"`,
			`export PATH="$CARGO_HOME/bin:$PATH"`,
			"export CARGO_INCREMENTAL=0",
			"export CARGO_PROFILE_RELEASE_DEBUG=0",
			"export CARGO_BUILD_JOBS=%{_smp_build_ncpus}",
			"export CARGO_TARGET_X86_64_UNKNOWN_LINUX_GNU_LINKER=clang",
			"# RPM's RUSTFLAGS override .cargo/config.toml, so retain upstream's cfg.",
			`export RUSTFLAGS="$RUSTFLAGS --cfg tokio_unstable -C symbol-mangling-version=v0 -C link-arg=-fuse-ld=lld"`,
			"export CC=clang CXX=clang++",
			"export ZED_UPDATE_EXPLANATION='Zed is managed by RPM. Run sudo dnf upgrade zed to update.'",
			"export RELEASE_VERSION='%{version}'",
			"script/generate-licenses",
			"cargo build --locked --release -p zed -p cli",
		},
		Install: []string{
			"install -Dm755 target/release/cli %{buildroot}%{_bindir}/zed",
			"install -Dm755 target/release/zed %{buildroot}%{_libexecdir}/zed-editor",
			"# Upstream adds the standard system libdir as an RPATH; RPM resolves it normally.",
			"patchelf --remove-rpath %{buildroot}%{_libexecdir}/zed-editor",
			"export DO_STARTUP_NOTIFY=true APP_CLI=zed APP_ICON=zed APP_ARGS='%%U' APP_NAME=Zed",
			"mkdir -p %{buildroot}%{_datadir}/applications",
			"envsubst < crates/zed/resources/zed.desktop.in > %{buildroot}%{_datadir}/applications/dev.zed.Zed.desktop",
			"chmod 755 %{buildroot}%{_datadir}/applications/dev.zed.Zed.desktop",
			"install -Dm644 crates/zed/resources/app-icon.png %{buildroot}%{_datadir}/icons/hicolor/512x512/apps/zed.png",
			"install -Dm644 crates/zed/resources/app-icon@2x.png %{buildroot}%{_datadir}/icons/hicolor/1024x1024/apps/zed.png",
		},
		Check: []string{"desktop-file-validate %{buildroot}%{_datadir}/applications/dev.zed.Zed.desktop"},
		Files: []string{
			`%license LICENSE-* assets/licenses.md`,
			`%{_bindir}/zed`,
			`%{_libexecdir}/zed-editor`,
			`%attr(0755,root,root) %{_datadir}/applications/dev.zed.Zed.desktop`,
			`%{_datadir}/icons/hicolor/*/apps/zed.png`,
		},
	}
}
