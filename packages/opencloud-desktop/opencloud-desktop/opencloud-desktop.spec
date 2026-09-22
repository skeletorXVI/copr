Name:           opencloud-desktop
Version:        4.0.0
Release:        1%{?dist}
Summary:        Desktop synchronization client for OpenCloud
License:        GPL-2.0-or-later
URL:            https://opencloud.eu
Source0:        https://github.com/opencloud-eu/desktop/archive/refs/tags/v4.0.0.tar.gz#/desktop-4.0.0.tar.gz
BuildRequires:  cmake
BuildRequires:  gcc-c++
BuildRequires:  extra-cmake-modules
BuildRequires:  qt6-qtbase-devel >= 6.8
BuildRequires:  qt6-qtdeclarative-devel
BuildRequires:  qt6-qttools-devel
BuildRequires:  qtkeychain-qt6-devel
BuildRequires:  cmake(KDSingleApplication-qt6)
BuildRequires:  libre-graph-api-cpp-qt-client-devel >= 1.0.4
BuildRequires:  sqlite-devel
BuildRequires:  zlib-devel
BuildRequires:  desktop-file-utils
Requires:       qt6-qtdeclarative
Requires:       hicolor-icon-theme
Recommends:     (opencloud-desktop-dolphin if dolphin)
Recommends:     (opencloud-desktop-nautilus if nautilus)

%description
Synchronize files between an OpenCloud server and your desktop. Updates are
managed by the system package manager. Install the optional Dolphin or Nautilus
package for file-manager status icons and context-menu actions.

%package devel
Summary:        Development files for OpenCloud Desktop
Requires:       %{name}%{?_isa} = %{version}-%{release}

%description devel
Headers and CMake metadata for the OpenCloud synchronization library.

%prep
%setup -q -n desktop-%{version}

%build
%cmake -DBUILD_TESTING=OFF -DWITH_AUTO_UPDATER=OFF \
    -DWITH_UPDATE_NOTIFICATION=OFF -DWITH_CRASHREPORTER=OFF \
    -DVIRTUAL_FILE_SYSTEM_PLUGINS=off
%cmake_build

%install
%cmake_install

%check
desktop-file-validate %{buildroot}%{_datadir}/applications/*.desktop

%files
%license COPYING*
%{_bindir}/opencloud
%{_bindir}/opencloudcmd
%{_libdir}/libOpenCloud*.so.*
%{_libdir}/libOpenCloudGui.so
%{_libdir}/qt6/plugins/OpenCloud_vfs_off.so
%{_libdir}/qt6/qml/eu/OpenCloud/
%{_datadir}/applications/*.desktop
%{_datadir}/metainfo/*.xml
%{_datadir}/icons/hicolor/*/apps/opencloud.png

%files devel
%{_includedir}/OpenCloud/
%{_libdir}/libOpenCloudLibSync.so
%{_libdir}/libOpenCloudResources.so
%{_libdir}/cmake/OpenCloud/

%changelog
* Tue Sep 22 2026 copr-bot <copr-bot@fabian-haenel.dev> - 4.0.0-1
- Update to 4.0.0
