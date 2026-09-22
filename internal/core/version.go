package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var versionPattern = regexp.MustCompile(`^v?([0-9]+)\.([0-9]+)(?:\.([0-9]+))?([a-z])?$`)

type Version struct {
	Major, Minor, Patch uint64
	Suffix              string
}

func ParseVersion(tag string) (Version, error) {
	m := versionPattern.FindStringSubmatch(tag)
	if m == nil {
		return Version{}, fmt.Errorf("unsupported release tag %q", tag)
	}
	var parts [3]uint64
	for i := range parts {
		if m[i+1] == "" {
			continue
		}
		n, err := strconv.ParseUint(m[i+1], 10, 64)
		if err != nil {
			return Version{}, err
		}
		parts[i] = n
	}
	return Version{parts[0], parts[1], parts[2], m[4]}, nil
}

func CompareVersions(a, b Version) int {
	for i, x := range []uint64{a.Major, a.Minor, a.Patch} {
		y := []uint64{b.Major, b.Minor, b.Patch}[i]
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	// A letter suffix can be part of an upstream's stable numbering, as in
	// Zen's 1.22.2b, so it orders rather than marking a prerelease.
	return strings.Compare(a.Suffix, b.Suffix)
}

// IsPatch reports whether next only increments the patch component of old.
// It is a building block for package policies that treat patch releases
// differently from the rest; nothing in the core calls it.
func IsPatch(old, next string) bool {
	a, e1 := ParseVersion(old)
	b, e2 := ParseVersion(next)
	return e1 == nil && e2 == nil && a.Major == b.Major && a.Minor == b.Minor && a.Suffix == b.Suffix && b.Patch > a.Patch
}
