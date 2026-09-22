package core

import (
	"fmt"
	"sort"
	"strings"
)

// Global is a %global macro definition emitted above the preamble.
type Global struct{ Name, Value string }

// SubPackage is a %package section. Suffix is the part after the main package
// name, so "devel" produces %package devel and %files devel.
type SubPackage struct {
	Suffix      string
	Summary     string
	BuildArch   string
	Requires    []string
	Description []string
	Files       []string
}

// Spec describes a spec file. The preamble is typed because the generator edits
// it; the build sections stay verbatim shell, one line per slice element,
// because inventing a portable abstraction over %prep/%build/%install buys
// less than it costs.
type Spec struct {
	Globals []Global

	Name    string
	Version string
	// Release is assigned by the generator, not by a package definition.
	Release int
	Summary string
	License string
	URL     string
	Sources []Source

	ExclusiveArch string
	BuildArch     string
	BuildRequires []string
	Requires      []string
	Recommends    []string
	Provides      []string
	Conflicts     []string
	Obsoletes     []string

	Description []string
	SubPackages []SubPackage

	Prep    []string
	Build   []string
	Install []string
	Check   []string
	Files   []string

	// Changelog is assigned by the generator, not by a package definition.
	// Fedora reads SOURCE_DATE_EPOCH out of the newest entry's date.
	Changelog []string
}

func (s Spec) validate() error {
	switch {
	case s.Name == "":
		return fmt.Errorf("spec has no name")
	case s.Version == "":
		return fmt.Errorf("%s: spec has no version", s.Name)
	case s.Summary == "" || s.License == "" || s.URL == "":
		return fmt.Errorf("%s: spec needs Summary, License and URL", s.Name)
	case len(s.Description) == 0:
		return fmt.Errorf("%s: spec has no description", s.Name)
	case len(s.Files) == 0:
		return fmt.Errorf("%s: spec lists no files", s.Name)
	}
	for _, source := range s.Sources {
		if source.Name == "" {
			return fmt.Errorf("%s: source without a file name", s.Name)
		}
	}
	return nil
}

// RemoteSources are the sources that must be downloaded and checksum-verified.
func (s Spec) RemoteSources() []Source {
	var remote []Source
	for _, source := range s.Sources {
		if source.Remote() {
			remote = append(remote, source)
		}
	}
	return remote
}

const tagWidth = 16

func tag(b *strings.Builder, name, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(b, "%-*s%s\n", tagWidth, name+":", value)
}

func tags(b *strings.Builder, name string, values []string) {
	for _, value := range values {
		tag(b, name, value)
	}
}

// section writes a body of lines under its section header. Blank lines at the
// edges are dropped, so a definition may pad its lines for readability.
func section(b *strings.Builder, name string, lines []string) {
	body := strings.Trim(strings.Join(lines, "\n"), "\n")
	if body == "" {
		return
	}
	fmt.Fprintf(b, "\n%s\n%s\n", name, body)
}

// Render writes the spec file. Output depends only on the Spec, so an
// unchanged package regenerates byte for byte and produces no diff.
func (s Spec) Render() (string, error) {
	if err := s.validate(); err != nil {
		return "", err
	}
	var b strings.Builder
	for _, global := range s.Globals {
		fmt.Fprintf(&b, "%%global %s %s\n", global.Name, global.Value)
	}
	if len(s.Globals) > 0 {
		b.WriteString("\n")
	}
	tag(&b, "Name", s.Name)
	tag(&b, "Version", s.Version)
	tag(&b, "Release", fmt.Sprintf("%d%%{?dist}", s.Release))
	tag(&b, "Summary", s.Summary)
	tag(&b, "License", s.License)
	tag(&b, "URL", s.URL)
	for i, source := range s.Sources {
		value := source.Name
		if source.Remote() {
			value = source.URL
		}
		tag(&b, fmt.Sprintf("Source%d", i), value)
	}
	tag(&b, "ExclusiveArch", s.ExclusiveArch)
	tag(&b, "BuildArch", s.BuildArch)
	tags(&b, "BuildRequires", s.BuildRequires)
	tags(&b, "Requires", s.Requires)
	tags(&b, "Recommends", s.Recommends)
	tags(&b, "Provides", s.Provides)
	tags(&b, "Conflicts", s.Conflicts)
	tags(&b, "Obsoletes", s.Obsoletes)

	section(&b, "%description", s.Description)
	for _, sub := range s.SubPackages {
		fmt.Fprintf(&b, "\n%%package %s\n", sub.Suffix)
		tag(&b, "Summary", sub.Summary)
		tag(&b, "BuildArch", sub.BuildArch)
		tags(&b, "Requires", sub.Requires)
		section(&b, "%description "+sub.Suffix, sub.Description)
	}
	section(&b, "%prep", s.Prep)
	section(&b, "%build", s.Build)
	section(&b, "%install", s.Install)
	section(&b, "%check", s.Check)
	section(&b, "%files", s.Files)
	for _, sub := range s.SubPackages {
		section(&b, "%files "+sub.Suffix, sub.Files)
	}
	section(&b, "%changelog", s.Changelog)
	return b.String(), nil
}

// Sorted returns a copy of names in a stable order, for callers that assemble
// dependency lists from maps.
func Sorted(names []string) []string {
	out := append([]string(nil), names...)
	sort.Strings(out)
	return out
}
