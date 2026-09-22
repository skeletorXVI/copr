Name:           opencloud-desktop-dolphin
Version:        1.0.0
Release:        1%{?dist}
Summary:        OpenCloud integration for Dolphin
License:        GPL-2.0-or-later
URL:            https://github.com/opencloud-eu/desktop-shell-integration-dolphin
Source0:        https://github.com/opencloud-eu/desktop-shell-integration-dolphin/archive/refs/tags/v1.0.0.tar.gz#/desktop-shell-integration-dolphin-1.0.0.tar.gz
BuildRequires:  cmake
BuildRequires:  gcc-c++
BuildRequires:  extra-cmake-modules
BuildRequires:  qt6-qtbase-devel
BuildRequires:  kf6-kcoreaddons-devel
BuildRequires:  kf6-kio-devel
BuildRequires:  kf6-kbookmarks-devel
BuildRequires:  opencloud-desktop-integration-resources >= 1.0.0
Requires:       dolphin
Requires:       opencloud-desktop
Requires:       opencloud-desktop-integration-resources >= 1.0.0

%description
Sync status overlays and context-menu actions for OpenCloud in KDE Dolphin.

%prep
%setup -q -n desktop-shell-integration-dolphin-%{version}

%build
%cmake
%cmake_build

%install
%cmake_install

%files
%license COPYING
%{_libdir}/libopenclouddolphinpluginhelper.so
%{_libdir}/qt6/plugins/kf6/overlayicon/openclouddolphinoverlayplugin.so
%{_libdir}/qt6/plugins/kf6/kfileitemaction/openclouddolphinactionplugin.so

%changelog
* Tue Sep 22 2026 copr-bot <copr-bot@fabian-haenel.dev> - 1.0.0-1
- Update to 1.0.0
