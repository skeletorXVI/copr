%global debug_package %{nil}

Name:           zed
Version:        1.21.0
Release:        1%{?dist}
Summary:        High-performance collaborative code editor
License:        GPL-3.0-or-later AND Apache-2.0 AND AGPL-3.0-or-later
URL:            https://zed.dev
Source0:        https://github.com/zed-industries/zed/archive/refs/tags/v1.21.0.tar.gz#/zed-1.21.0.tar.gz
ExclusiveArch:  x86_64
BuildRequires:  cargo
BuildRequires:  rust >= 1.98.1
BuildRequires:  clang
BuildRequires:  lld
BuildRequires:  llvm-devel
BuildRequires:  gcc-c++
BuildRequires:  cmake
BuildRequires:  make
BuildRequires:  git
BuildRequires:  curl
BuildRequires:  perl-FindBin
BuildRequires:  perl-IPC-Cmd
BuildRequires:  perl-File-Compare
BuildRequires:  perl-File-Copy
BuildRequires:  pkgconfig(alsa)
BuildRequires:  pkgconfig(fontconfig)
BuildRequires:  pkgconfig(glib-2.0)
BuildRequires:  pkgconfig(libva)
BuildRequires:  pkgconfig(wayland-client)
BuildRequires:  pkgconfig(xcb)
BuildRequires:  pkgconfig(xkbcommon-x11)
BuildRequires:  pkgconfig(openssl)
BuildRequires:  pkgconfig(libzstd)
BuildRequires:  pkgconfig(sqlite3)
BuildRequires:  gettext
BuildRequires:  desktop-file-utils
BuildRequires:  patchelf
Requires:       vulkan-loader
Requires:       libva
Requires:       mesa-libEGL
Requires:       xdg-desktop-portal
Recommends:     pipewire
Recommends:     git

%description
Zed is a fast, collaborative editor built from its tagged source release.
Use your system package manager to update this installation.

%prep
%setup -q
# No trailing newline: required for stable-channel credential storage.
printf stable > crates/zed/RELEASE_CHANNEL

%build
# COPR must have networking enabled: Cargo.lock pins Rust dependencies, and
# upstream's WebRTC build fetches its native library. See this package's README.
export CARGO_HOME="$PWD/.cargo-home"
export PATH="$CARGO_HOME/bin:$PATH"
export CARGO_INCREMENTAL=0
export CARGO_PROFILE_RELEASE_DEBUG=0
export CARGO_BUILD_JOBS=%{_smp_build_ncpus}
export CARGO_TARGET_X86_64_UNKNOWN_LINUX_GNU_LINKER=clang
# RPM's RUSTFLAGS override .cargo/config.toml, so retain upstream's cfg.
export RUSTFLAGS="$RUSTFLAGS --cfg tokio_unstable -C symbol-mangling-version=v0 -C link-arg=-fuse-ld=lld"
export CC=clang CXX=clang++
export ZED_UPDATE_EXPLANATION='Zed is managed by RPM. Run sudo dnf upgrade zed to update.'
export RELEASE_VERSION='%{version}'
script/generate-licenses
cargo build --locked --release -p zed -p cli

%install
install -Dm755 target/release/cli %{buildroot}%{_bindir}/zed
install -Dm755 target/release/zed %{buildroot}%{_libexecdir}/zed-editor
# Upstream adds the standard system libdir as an RPATH; RPM resolves it normally.
patchelf --remove-rpath %{buildroot}%{_libexecdir}/zed-editor
export DO_STARTUP_NOTIFY=true APP_CLI=zed APP_ICON=zed APP_ARGS='%%U' APP_NAME=Zed
mkdir -p %{buildroot}%{_datadir}/applications
envsubst < crates/zed/resources/zed.desktop.in > %{buildroot}%{_datadir}/applications/dev.zed.Zed.desktop
chmod 755 %{buildroot}%{_datadir}/applications/dev.zed.Zed.desktop
install -Dm644 crates/zed/resources/app-icon.png %{buildroot}%{_datadir}/icons/hicolor/512x512/apps/zed.png
install -Dm644 crates/zed/resources/app-icon@2x.png %{buildroot}%{_datadir}/icons/hicolor/1024x1024/apps/zed.png

%check
desktop-file-validate %{buildroot}%{_datadir}/applications/dev.zed.Zed.desktop

%files
%license LICENSE-* assets/licenses.md
%{_bindir}/zed
%{_libexecdir}/zed-editor
%attr(0755,root,root) %{_datadir}/applications/dev.zed.Zed.desktop
%{_datadir}/icons/hicolor/*/apps/zed.png

%changelog
* Wed Sep 23 2026 copr-bot <copr-bot@fabian-haenel.dev> - 1.21.0-1
- Update to 1.21.0

* Tue Sep 22 2026 copr-bot <copr-bot@fabian-haenel.dev> - 1.20.2-1
- Update to 1.20.2

* Tue Sep 22 2026 copr-bot <copr-bot@fabian-haenel.dev> - 1.20.1-2
- Update to 1.20.1
