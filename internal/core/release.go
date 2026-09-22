// Package core provides the packaging primitives shared by every package
// generator: how an upstream release is discovered, how a spec file is
// described, and how both are turned into the files COPR builds from.
package core

import "context"

// Source is one SourceN entry of a spec file. A source with no URL is a file
// committed next to the spec; a source with a URL is downloaded and verified
// against the checksum recorded in the package's sources file.
type Source struct {
	Name string
	URL  string
}

func (s Source) Remote() bool { return s.URL != "" }

// Release is an upstream release a package can be built from. Vars carries
// values a particular package needs in its spec and nothing else understands,
// such as a minimum compiler version read from the upstream source tree.
type Release struct {
	Version string
	Vars    map[string]string
}

func (r Release) Var(name string) string { return r.Vars[name] }

// Releases is a package's upstream. Implementations decide entirely on their
// own what "released" means: a GitHub release, the highest version tag, a
// distribution index, or anything else reachable from the generator.
type Releases interface {
	LatestRelease(ctx context.Context) (Release, error)
}

// ReleasesFunc adapts a plain function to Releases.
type ReleasesFunc func(ctx context.Context) (Release, error)

func (f ReleasesFunc) LatestRelease(ctx context.Context) (Release, error) { return f(ctx) }
