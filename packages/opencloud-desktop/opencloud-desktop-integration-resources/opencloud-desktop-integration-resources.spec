Name:           opencloud-desktop-integration-resources
Version:        1.0.0
Release:        1%{?dist}
Summary:        Shared icons for OpenCloud file-manager integrations
License:        GPL-2.0-or-later
URL:            https://github.com/opencloud-eu/desktop-shell-integration-resources
Source0:        https://github.com/opencloud-eu/desktop-shell-integration-resources/archive/refs/tags/v1.0.0.tar.gz#/desktop-shell-integration-resources-1.0.0.tar.gz
BuildArch:      noarch
BuildRequires:  cmake
BuildRequires:  gcc
BuildRequires:  extra-cmake-modules
Requires:       hicolor-icon-theme

%description
Shared status icons and discovery metadata for OpenCloud desktop integrations.

%prep
%setup -q -n desktop-shell-integration-resources-%{version}

%build
%cmake -DCMAKE_INSTALL_LIBDIR=share
%cmake_build

%install
%cmake_install

%files
%license COPYING
%{_datadir}/icons/hicolor/*/apps/*.png
%{_datadir}/cmake/OpenCloudShellResources/

%changelog
* Tue Sep 22 2026 copr-bot <copr-bot@fabian-haenel.dev> - 1.0.0-1
- Update to 1.0.0
