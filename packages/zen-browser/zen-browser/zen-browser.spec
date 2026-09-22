%global debug_package %{nil}
%global __strip /bin/true
%global __brp_strip /bin/true
%global __brp_strip_comment_note /bin/true
%global __brp_strip_static_archive /bin/true
%global __provides_exclude_from ^%{_libdir}/zen-browser/.*$
%global __requires_exclude ^lib(moz[^.]*|gkcodecs|lgpllibs|xul|onnxruntime)\\.so.*$

Name:           zen-browser
Version:        1.22.2b
Release:        1%{?dist}
Summary:        Privacy-focused web browser based on Firefox
License:        MPL-2.0
URL:            https://zen-browser.app
Source0:        https://github.com/zen-browser/desktop/releases/download/1.22.2b/zen.linux-x86_64.tar.xz
Source1:        zen-browser.desktop
Source2:        policies.json
ExclusiveArch:  x86_64
BuildRequires:  desktop-file-utils
BuildRequires:  patchelf
Requires:       gtk3
Requires:       dbus-glib
Requires:       alsa-lib
Requires:       libXt
Requires:       libX11-xcb
Requires:       libXcomposite
Requires:       libXdamage
Requires:       libXrandr
Requires:       mesa-libgbm
Requires:       nss
Requires:       nspr
Requires:       mozilla-filesystem

%description
Zen Browser, repackaged from the official Linux binary release. Application
updates are managed by RPM; browser profiles remain in the user's home directory.

%prep
%setup -q -n zen

%build
# Upstream binaries are intentionally left intact.

%install
mkdir -p %{buildroot}%{_libdir}/%{name}
cp -a . %{buildroot}%{_libdir}/%{name}/
# Upstream ONNX carries a literal '$' RUNPATH, rejected by Fedora's RPATH check.
# It needs no extra search directory in this private library installation.
patchelf --remove-rpath %{buildroot}%{_libdir}/%{name}/libonnxruntime.so
install -Dm644 %{SOURCE2} %{buildroot}%{_libdir}/%{name}/distribution/policies.json
mkdir -p %{buildroot}%{_bindir}
ln -s %{_libdir}/%{name}/zen %{buildroot}%{_bindir}/%{name}
desktop-file-install --dir=%{buildroot}%{_datadir}/applications %{SOURCE1}
for size in 16 32 48 64 128; do
    install -Dm644 browser/chrome/icons/default/default${size}.png \
        %{buildroot}%{_datadir}/icons/hicolor/${size}x${size}/apps/%{name}.png
done

%check
desktop-file-validate %{buildroot}%{_datadir}/applications/%{name}.desktop
test -x %{buildroot}%{_libdir}/%{name}/zen

%files
%{_bindir}/%{name}
%{_libdir}/%{name}/
%{_datadir}/applications/%{name}.desktop
%{_datadir}/icons/hicolor/*/apps/%{name}.png

%changelog
* Tue Sep 22 2026 copr-bot <copr-bot@fabian-haenel.dev> - 1.22.2b-1
- Update to 1.22.2b
