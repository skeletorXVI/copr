Name:           libre-graph-api-cpp-qt-client
Version:        1.0.7
Release:        1%{?dist}
Summary:        Qt client for the Libre Graph API
License:        Apache-2.0
URL:            https://github.com/opencloud-eu/libre-graph-api-cpp-qt-client
Source0:        https://github.com/opencloud-eu/libre-graph-api-cpp-qt-client/archive/refs/tags/v1.0.7.tar.gz#/libre-graph-api-cpp-qt-client-1.0.7.tar.gz
BuildRequires:  cmake
BuildRequires:  gcc-c++
BuildRequires:  qt6-qtbase-devel
BuildRequires:  zlib-devel

%description
C++ and Qt bindings for the Libre Graph collaboration API.

%package devel
Summary:        Development files for the Libre Graph Qt client
Requires:       %{name}%{?_isa} = %{version}-%{release}
Requires:       qt6-qtbase-devel

%description devel
Headers and CMake metadata for the Libre Graph Qt client.

%prep
%setup -q

%build
%cmake -S client -DBUILD_SHARED_LIBS=ON -DCMAKE_POLICY_VERSION_MINIMUM=3.5
%cmake_build

%install
%cmake_install

%files
%license LICENSE
%{_libdir}/libLibreGraphAPI.so.*

%files devel
%{_includedir}/OpenAPI/
%{_libdir}/libLibreGraphAPI.so
%{_libdir}/cmake/LibreGraphAPI/

%changelog
* Tue Sep 22 2026 copr-bot <copr-bot@fabian-haenel.dev> - 1.0.7-1
- Update to 1.0.7
