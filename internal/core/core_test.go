package core

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRender(t *testing.T) {
	spec := Spec{
		Globals: []Global{{Name: "debug_package", Value: "%{nil}"}},
		Name:    "example",
		Version: "1.2.3",
		Release: 2,
		Summary: "An example",
		License: "MIT",
		URL:     "https://example.invalid",
		Sources: []Source{
			{Name: "example-1.2.3.tar.gz", URL: "https://example.invalid/example-1.2.3.tar.gz"},
			{Name: "example.desktop"},
		},
		BuildArch:     "noarch",
		BuildRequires: []string{"cmake"},
		Requires:      []string{"bash"},
		Description:   []string{"Example package."},
		SubPackages: []SubPackage{{
			Suffix:      "devel",
			Summary:     "Development files",
			Requires:    []string{"%{name}%{?_isa} = %{version}-%{release}"},
			Description: []string{"Headers."},
			Files:       []string{"%{_includedir}/example/"},
		}},
		Prep:    []string{"%setup -q"},
		Install: []string{"%cmake_install"},
		Files:   []string{"%{_bindir}/example"},
	}
	want := `%global debug_package %{nil}

Name:           example
Version:        1.2.3
Release:        2%{?dist}
Summary:        An example
License:        MIT
URL:            https://example.invalid
Source0:        https://example.invalid/example-1.2.3.tar.gz
Source1:        example.desktop
BuildArch:      noarch
BuildRequires:  cmake
Requires:       bash

%description
Example package.

%package devel
Summary:        Development files
Requires:       %{name}%{?_isa} = %{version}-%{release}

%description devel
Headers.

%prep
%setup -q

%install
%cmake_install

%files
%{_bindir}/example

%files devel
%{_includedir}/example/
`
	got, err := spec.Render()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("rendered spec differs:\n%s", got)
	}
}

func TestRenderRejectsIncompleteSpec(t *testing.T) {
	if _, err := (Spec{Name: "x", Version: "1"}).Render(); err == nil {
		t.Fatal("incomplete spec accepted")
	}
}

// gen writes exactly what it is told to: a version it is given and a release
// number it keeps, so regenerating a package reproduces the committed file.
func TestGenerateIsReproducible(t *testing.T) {
	t.Setenv("RELEASE_BOT_EMAIL", "copr-bot@example.invalid")
	dir := t.TempDir()
	summary := "First"
	def := Definition{
		Project: "example", Name: "example", Policy: ReviewAll{},
		Chroots: []string{"fedora-44-x86_64"},
		Releases: ReleasesFunc(func(context.Context) (Release, error) {
			return Release{Version: "1.0.0"}, nil
		}),
		Spec: func(Release) Spec {
			return Spec{
				Summary: summary, License: "MIT", URL: "https://example.invalid",
				Description: []string{"Example."}, Files: []string{"%{_bindir}/example"},
			}
		},
	}
	gen := func(args ...string) string {
		t.Helper()
		if err := run(def, append([]string{"-dir", dir}, args...), io.Discard); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dir, "example.spec"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	release := func(spec string) string {
		t.Helper()
		for _, line := range strings.Split(spec, "\n") {
			if strings.HasPrefix(line, "Release:") {
				return strings.TrimSpace(strings.TrimPrefix(line, "Release:"))
			}
		}
		t.Fatal("no Release tag")
		return ""
	}
	first := gen("gen", "1.0.0")
	if got := release(first); got != "1%{?dist}" {
		t.Fatalf("first pin: Release %s", got)
	}
	if again := gen("gen", "1.0.0"); again != first {
		t.Fatal("regenerating an unchanged package rewrote the spec")
	}
	// The version is not read back out of the spec as the next version, but
	// omitting it regenerates whatever is packaged now.
	if again := gen("gen"); again != first {
		t.Fatal("regenerating without a version rewrote the spec")
	}
	// A packaging change at the same version keeps its release number until
	// the packager asks for the next build.
	summary = "Second"
	changed := gen("gen", "1.0.0")
	if changed == first {
		t.Fatal("a changed definition did not change the spec")
	}
	if got := release(changed); got != "1%{?dist}" {
		t.Fatalf("packaging change: Release %s", got)
	}
	if got := release(gen("-release", "2", "gen", "1.0.0")); got != "2%{?dist}" {
		t.Fatalf("explicit release: Release %s", got)
	}
	if got := release(gen("gen", "1.0.1")); got != "1%{?dist}" {
		t.Fatalf("new version: Release %s", got)
	}
}

// Nothing may move a package backwards, whether it comes from upstream or
// from the command line.
func TestRejectsDowngrade(t *testing.T) {
	t.Setenv("RELEASE_BOT_EMAIL", "copr-bot@example.invalid")
	dir := t.TempDir()
	version := "2.0.0"
	def := Definition{
		Project: "example", Name: "example", Policy: ReviewAll{},
		Chroots: []string{"fedora-44-x86_64"},
		Releases: ReleasesFunc(func(context.Context) (Release, error) {
			return Release{Version: version}, nil
		}),
		Spec: func(Release) Spec {
			return Spec{
				Summary: "S", License: "MIT", URL: "https://example.invalid",
				Description: []string{"Example."}, Files: []string{"%{_bindir}/example"},
			}
		},
	}
	if err := run(def, []string{"-dir", dir, "gen", "2.0.0"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := run(def, []string{"-dir", dir, "gen", "1.9.0"}, io.Discard); err == nil {
		t.Fatal("downgrade accepted")
	}
	version = "1.9.0"
	if err := run(def, []string{"-dir", dir, "check"}, io.Discard); err == nil {
		t.Fatal("upstream downgrade accepted")
	}
}

// check reports what a workflow step needs and writes nothing.
func TestCheckReports(t *testing.T) {
	t.Setenv("RELEASE_BOT_EMAIL", "copr-bot@example.invalid")
	dir := t.TempDir()
	version := "1.0.0"
	skip := false
	def := Definition{
		Project: "example", Name: "example", Upstream: "https://example.invalid/src",
		Chroots: []string{"fedora-44-x86_64"},
		Policy: PolicyFunc(func(Update) Action {
			if skip {
				return Skip
			}
			return Review
		}),
		Releases: ReleasesFunc(func(context.Context) (Release, error) {
			return Release{Version: version}, nil
		}),
		Spec: func(Release) Spec {
			return Spec{
				Summary: "S", License: "MIT", URL: "https://example.invalid",
				Description: []string{"Example."}, Files: []string{"%{_bindir}/example"},
			}
		},
	}
	check := func() string {
		t.Helper()
		var out strings.Builder
		if err := run(def, []string{"-dir", dir, "check"}, &out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	if got := check(); got != "current=\nversion=1.0.0\nupdate=true\nupstream=https://example.invalid/src\n" {
		t.Fatalf("unpinned package: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "example.spec")); !os.IsNotExist(err) {
		t.Fatal("check wrote a spec file")
	}
	if err := run(def, []string{"-dir", dir, "gen", version}, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got := check(); got != "current=1.0.0\nversion=1.0.0\nupdate=false\nupstream=https://example.invalid/src\n" {
		t.Fatalf("packaged version: %q", got)
	}
	version = "1.1.0"
	if got := check(); got != "current=1.0.0\nversion=1.1.0\nupdate=true\nupstream=https://example.invalid/src\n" {
		t.Fatalf("new release: %q", got)
	}
	// A policy that declines the release reports no update, so the workflow
	// never opens a pull request for it.
	skip = true
	if got := check(); got != "current=1.0.0\nversion=1.1.0\nupdate=false\nupstream=https://example.invalid/src\n" {
		t.Fatalf("declined release: %q", got)
	}
}

func TestVersions(t *testing.T) {
	for _, tt := range []struct {
		old, next string
		patch     bool
	}{
		{"1.2.3", "1.2.4", true},
		{"1.2.3", "1.3.0", false},
		{"1.22b", "1.22.1b", true},
		{"1.22.1b", "1.23b", false},
		{"1.2.3", "1.2.3", false},
	} {
		if got := IsPatch(tt.old, tt.next); got != tt.patch {
			t.Errorf("IsPatch(%s, %s) = %t", tt.old, tt.next, got)
		}
	}
	a, err := ParseVersion("v1.22.2b")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := ParseVersion("1.22.2")
	if CompareVersions(a, b) <= 0 {
		t.Fatal("suffix must order after the plain version")
	}
	if _, err := ParseVersion("nightly"); err == nil {
		t.Fatal("unknown tag format accepted")
	}
}

// A policy acts on the update itself, so a package can decline a release
// without the core knowing any modes.
func TestPolicyDecides(t *testing.T) {
	update := Update{Name: "example", From: "1.2.3", To: Release{Version: "1.2.4"}, Release: 1}
	if got := (ReviewAll{}).Decide(update); got != Review {
		t.Fatalf("ReviewAll: %s", got)
	}
	// A rebuild at an unchanged version is a packaging change, not a patch.
	rebuild := Update{From: "1.2.3", To: Release{Version: "1.2.3"}, Release: 2}
	held := PolicyFunc(func(u Update) Action {
		if IsPatch(u.From, u.To.Version) {
			return Review
		}
		return Skip
	})
	if got := held.Decide(update); got != Review {
		t.Fatalf("patch update: %s", got)
	}
	if got := held.Decide(rebuild); got != Skip {
		t.Fatalf("rebuild: %s", got)
	}
}

// Where and how a package is built is the package's own statement, so the
// workflows never carry a rule about any particular one.
func TestConfigReportsBuildTargets(t *testing.T) {
	def := Definition{
		Project: "example", Name: "example", Policy: ReviewAll{},
		Releases: ReleasesFunc(func(context.Context) (Release, error) {
			return Release{Version: "1.0.0"}, nil
		}),
		Spec: func(Release) Spec { return Spec{} },
		// Two architectures of one release, because building a source RPM
		// does not depend on the architecture.
		Chroots: []string{"fedora-44-x86_64", "fedora-44-aarch64", "fedora-rawhide-x86_64"},
		Network: true,
		Verify:  []string{"example", "--version"},
	}
	var out strings.Builder
	if err := run(def, []string{"-dir", t.TempDir(), "config"}, &out); err != nil {
		t.Fatal(err)
	}
	want := "project=example\nname=example\n" +
		"chroots=fedora-44-x86_64 fedora-44-aarch64 fedora-rawhide-x86_64\n" +
		"releases=44 rawhide\nnetwork=on\nrebuild=true\nverify=example --version\n"
	if out.String() != want {
		t.Fatalf("config reported:\n%s\nwant:\n%s", out.String(), want)
	}
	def.Chroots = []string{"debian-13-amd64"}
	if err := run(def, []string{"config"}, io.Discard); err == nil {
		t.Fatal("a chroot the workflows cannot build was accepted")
	}
	def.Chroots = nil
	if err := run(def, []string{"config"}, io.Discard); err == nil {
		t.Fatal("a package with nowhere to build was accepted")
	}
}

// The changelog is what Fedora derives SOURCE_DATE_EPOCH from, so an entry is
// dated once and then carried forward: only a version or release that moved
// gets a new one.
func TestChangelogIsDatedOnceAndPreserved(t *testing.T) {
	day := time.Date(2026, time.September, 22, 0, 0, 0, 0, time.UTC)
	defer func(original func() time.Time) { now = original }(now)
	now = func() time.Time { return day }
	t.Setenv("RELEASE_BOT_NAME", "copr-bot")
	t.Setenv("RELEASE_BOT_EMAIL", "copr-bot@example.invalid")

	dir := t.TempDir()
	summary := "First"
	def := Definition{
		Project: "example", Name: "example", Policy: ReviewAll{},
		Chroots: []string{"fedora-44-x86_64"},
		Releases: ReleasesFunc(func(context.Context) (Release, error) {
			return Release{Version: "1.0.0"}, nil
		}),
		Spec: func(Release) Spec {
			return Spec{
				Summary: summary, License: "MIT", URL: "https://example.invalid",
				Description: []string{"Example."}, Files: []string{"%{_bindir}/example"},
			}
		},
	}
	gen := func(args ...string) string {
		t.Helper()
		if err := run(def, append([]string{"-dir", dir}, args...), io.Discard); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dir, "example.spec"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	entries := func(spec string) []string {
		t.Helper()
		var out []string
		for _, line := range strings.Split(spec, "\n") {
			if strings.HasPrefix(line, "* ") {
				out = append(out, line)
			}
		}
		return out
	}

	first := gen("gen", "1.0.0")
	want := "* Tue Sep 22 2026 copr-bot <copr-bot@example.invalid> - 1.0.0-1"
	if got := entries(first); len(got) != 1 || got[0] != want {
		t.Fatalf("first pin: %q", got)
	}
	if !strings.HasSuffix(first, "\n%changelog\n"+want+"\n- Update to 1.0.0\n") {
		t.Fatalf("changelog is not the last section:\n%s", first)
	}

	// A later regeneration must not redate the entry, even a year on.
	now = func() time.Time { return day.AddDate(1, 0, 0) }
	if again := gen("gen", "1.0.0"); again != first {
		t.Fatal("regenerating an unchanged package redated the changelog")
	}
	// A packaging change that keeps the release number keeps the entry too:
	// nothing new has been built.
	summary = "Second"
	if got := entries(gen("gen", "1.0.0")); len(got) != 1 || got[0] != want {
		t.Fatalf("packaging change: %q", got)
	}
	// A new release of the same version, and a new version, each earn one.
	if got := entries(gen("-release", "2", "gen", "1.0.0")); len(got) != 2 ||
		!strings.HasSuffix(got[0], " - 1.0.0-2") {
		t.Fatalf("explicit release: %q", got)
	}
	newest := gen("gen", "1.0.1")
	if got := entries(newest); len(got) != 3 || !strings.HasSuffix(got[0], " - 1.0.1-1") {
		t.Fatalf("new version: %q", got)
	}
	if !strings.Contains(newest, "- Update to 1.0.1") || !strings.Contains(newest, "- Rebuild for packaging changes") {
		t.Fatalf("entries do not say what changed:\n%s", newest)
	}
}

// The address the packages are published under is never guessed: writing an
// entry without one fails rather than inventing an author.
func TestChangelogRequiresAnAuthorAddress(t *testing.T) {
	t.Setenv("RELEASE_BOT_EMAIL", "")
	dir := t.TempDir()
	def := Definition{
		Project: "example", Name: "example", Policy: ReviewAll{},
		Chroots: []string{"fedora-44-x86_64"},
		Releases: ReleasesFunc(func(context.Context) (Release, error) {
			return Release{Version: "1.0.0"}, nil
		}),
		Spec: func(Release) Spec {
			return Spec{
				Summary: "Example", License: "MIT", URL: "https://example.invalid",
				Description: []string{"Example."}, Files: []string{"%{_bindir}/example"},
			}
		},
	}
	err := run(def, []string{"-dir", dir, "gen", "1.0.0"}, io.Discard)
	if !errors.Is(err, ErrNoAuthor) {
		t.Fatalf("pinned a version with no author address: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "example.spec")); !os.IsNotExist(err) {
		t.Fatal("a spec was written without an author address")
	}

	// Regenerating what is already packaged writes no entry, so it needs no
	// address either.
	t.Setenv("RELEASE_BOT_EMAIL", "copr-bot@example.invalid")
	if err := run(def, []string{"-dir", dir, "gen", "1.0.0"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RELEASE_BOT_EMAIL", "")
	if err := run(def, []string{"-dir", dir, "gen"}, io.Discard); err != nil {
		t.Fatalf("regenerating the packaged version demanded an address: %v", err)
	}
}
