Name:           opencloud-desktop-nautilus
Version:        1.0.0
Release:        1%{?dist}
Summary:        OpenCloud integration for Nautilus
License:        GPL-2.0-or-later
URL:            https://github.com/opencloud-eu/desktop-shell-integration-nautilus
Source0:        https://github.com/opencloud-eu/desktop-shell-integration-nautilus/archive/refs/tags/v1.0.0.tar.gz#/desktop-shell-integration-nautilus-1.0.0.tar.gz
BuildArch:      noarch
BuildRequires:  cmake
BuildRequires:  gcc
BuildRequires:  opencloud-desktop-integration-resources >= 1.0.0
Requires:       opencloud-desktop
Requires:       opencloud-desktop-integration-resources >= 1.0.0
Requires:       nautilus-python
Requires:       python3-gobject

%description
Sync status emblems and context-menu actions for OpenCloud in GNOME Files.

%prep
%setup -q -n desktop-shell-integration-nautilus-%{version}

%build
%cmake
%cmake_build

%install
%cmake_install
# Upstream installs all three integrations; this package targets Nautilus.
rm -r %{buildroot}%{_datadir}/nemo-python %{buildroot}%{_datadir}/caja-python

%files
%license COPYING*
%{_datadir}/nautilus-python/extensions/syncstate-OpenCloud.py

%changelog
* Tue Sep 22 2026 copr-bot <copr-bot@fabian-haenel.dev> - 1.0.0-1
- Update to 1.0.0
