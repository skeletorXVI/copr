package core

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Definition is everything the core needs to know about a package. Each
// package directory holds one program that fills this in and calls Run.
type Definition struct {
	// Project is the COPR project, and must match the parent directory.
	Project string
	// Name is the package name, and must match the package directory.
	Name string
	// Upstream is the project's home, used in update pull requests.
	Upstream string
	// Policy decides whether a discovered release should be packaged.
	Policy Policy

	// Chroots are the COPR chroots this package is built for. Where a package
	// can be built is the package's own business: one may reach an
	// architecture or a Fedora release that the others do not.
	Chroots []string
	// Network reports whether the binary build needs network access. Almost
	// nothing should; a build that fetches its own dependencies does.
	Network bool
	// Verify is a command run against what the rebuilt RPM installs. Setting
	// it asks the pull request check to rebuild the binary RPM from the
	// source RPM, install it and run this; that is cheap for repackaged
	// binaries and prohibitive for a large source build, so it is the
	// package's call.
	Verify []string

	// Releases discovers upstream versions. Every package answers "what is
	// the latest release?" its own way: a GitHub release, the highest version
	// tag, or anything else reachable from the generator.
	Releases Releases
	// Vars optionally derives extra values for a version. It is called for
	// the pinned version too, so regeneration never depends on what upstream
	// currently publishes.
	Vars func(ctx context.Context, version string) (map[string]string, error)
	// Checksum optionally returns the digest upstream publishes for a source,
	// which is compared against the bytes actually downloaded.
	Checksum func(ctx context.Context, version string, source Source) (string, error)
	// Spec describes the spec file for a release.
	Spec func(Release) Spec
}

func (d Definition) dir() string { return filepath.Join("packages", d.Project, d.Name) }

// chroot is a COPR chroot name. Only Fedora is recognised, because the
// workflows map the release onto a Fedora container image; anything else has
// to teach them how to build it first.
var chroot = regexp.MustCompile(`^fedora-([0-9]+|rawhide)-(x86_64|aarch64)$`)

// releases are the distinct Fedora releases behind the chroots. Building a
// source RPM does not depend on the architecture, so the check workflow
// matrixes over these rather than over every chroot.
func (d Definition) releases() []string {
	var out []string
	for _, c := range d.Chroots {
		release := chroot.FindStringSubmatch(c)[1]
		if !slices.Contains(out, release) {
			out = append(out, release)
		}
	}
	return out
}

func (d Definition) validate() error {
	switch {
	case d.Name == "" || d.Project == "":
		return fmt.Errorf("package definition needs a name and a project")
	case d.Releases == nil:
		return fmt.Errorf("%s: no release source", d.Name)
	case d.Spec == nil:
		return fmt.Errorf("%s: no spec function", d.Name)
	case d.Policy == nil:
		return fmt.Errorf("%s: no update policy", d.Name)
	case len(d.Chroots) == 0:
		return fmt.Errorf("%s: no chroots to build for", d.Name)
	}
	for _, c := range d.Chroots {
		if !chroot.MatchString(c) {
			return fmt.Errorf("%s: unsupported chroot %q", d.Name, c)
		}
	}
	return nil
}

const usage = `usage: <package> [-dir DIR] [-refresh] [-release N] COMMAND

  config       report where and how this package is built, as key=value lines
  check        ask upstream for its latest release and report whether it is
               newer than the packaged one, as key=value lines
  gen [VERSION] write the spec file and source checksums for VERSION, or for
               the packaged version when VERSION is omitted

check and config write GitHub Actions step outputs, so a workflow appends
them straight to $GITHUB_OUTPUT.`

// Run is the entry point of a package generator program. It is called from
// main and terminates the process.
func Run(def Definition) {
	if err := run(def, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", def.Name, err)
		os.Exit(1)
	}
}

func run(def Definition, args []string, out io.Writer) error {
	if err := def.validate(); err != nil {
		return err
	}
	flags := flag.NewFlagSet(def.Name, flag.ContinueOnError)
	dir := flags.String("dir", def.dir(), "package directory")
	refresh := flags.Bool("refresh", false, "re-download the sources and rewrite their checksums")
	release := flags.Int("release", 0, "RPM release number; the default keeps the packaged one and restarts at 1 for a new version")
	if err := flags.Parse(args); err != nil {
		return err
	}
	// The packaged version and release live in the committed spec file, which
	// records what is packaged now. What to package next is never read from
	// it: check asks upstream, and gen is told.
	packaged, built, committed, err := currentSpec(*dir, def.Name)
	if err != nil {
		return err
	}
	ctx := context.Background()
	switch flags.Arg(0) {
	case "config":
		// Nothing here is CI's decision to make: the workflows ask the
		// package where it builds, whether its build needs the network and
		// whether rebuilding it in a pull request is affordable.
		_, err := fmt.Fprintf(out, "project=%s\nname=%s\nchroots=%s\nreleases=%s\nnetwork=%s\nrebuild=%t\nverify=%s\n",
			def.Project, def.Name,
			strings.Join(def.Chroots, " "), strings.Join(def.releases(), " "),
			onOff(def.Network), len(def.Verify) > 0, strings.Join(def.Verify, " "))
		return err
	case "check":
		latest, err := def.Releases.LatestRelease(ctx)
		if err != nil {
			return err
		}
		if err := forward(packaged, latest.Version); err != nil {
			return err
		}
		update := packaged != latest.Version &&
			def.Policy.Decide(Update{Name: def.Name, From: packaged, To: latest, Release: 1}) != Skip
		_, err = fmt.Fprintf(out, "current=%s\nversion=%s\nupdate=%t\nupstream=%s\n",
			packaged, latest.Version, update, def.Upstream)
		return err
	case "gen":
		version := packaged
		if flags.NArg() > 1 {
			version = flags.Arg(1)
		}
		if version == "" {
			return fmt.Errorf("no spec file yet; name the version to pin: gen VERSION")
		}
		if err := forward(packaged, version); err != nil {
			return err
		}
		number := *release
		if number == 0 {
			// A new version restarts at 1; regenerating the packaged version
			// reproduces the committed file, so pass -release to rebuild it.
			number = 1
			if version == packaged {
				number = built
			}
		}
		log, err := changelog(committed, packaged, version, number)
		if err != nil {
			return err
		}
		return generate(ctx, def, *dir, version, number, log, version != packaged || *refresh, out)
	case "":
		return fmt.Errorf("no command\n%s", usage)
	default:
		return fmt.Errorf("unknown command %q\n%s", flags.Arg(0), usage)
	}
}

var (
	versionTag = regexp.MustCompile(`(?m)^Version:\s+(\S+)$`)
	releaseTag = regexp.MustCompile(`(?m)^Release:\s+([0-9]+)%\{\?dist\}$`)
)

// currentSpec reads the packaged version, release and changelog out of the
// committed spec. A package with no spec file yet has none of them.
func currentSpec(dir, name string) (string, int, []string, error) {
	b, err := os.ReadFile(filepath.Join(dir, name+".spec"))
	if os.IsNotExist(err) {
		return "", 0, nil, nil
	}
	if err != nil {
		return "", 0, nil, err
	}
	version := versionTag.FindSubmatch(b)
	release := releaseTag.FindSubmatch(b)
	if version == nil || release == nil {
		return "", 0, nil, fmt.Errorf("%s.spec: cannot read Version and Release", name)
	}
	n, err := strconv.Atoi(string(release[1]))
	if err != nil {
		return "", 0, nil, err
	}
	return string(version[1]), n, committedChangelog(string(b)), nil
}

// forward rejects moving a package backwards, whatever an upstream API or a
// command line says the next version is.
func forward(packaged, next string) error {
	if packaged == "" || packaged == next {
		return nil
	}
	a, err := ParseVersion(packaged)
	if err != nil {
		return err
	}
	b, err := ParseVersion(next)
	if err != nil {
		return err
	}
	if CompareVersions(b, a) < 0 {
		return fmt.Errorf("%s is older than the packaged %s", next, packaged)
	}
	return nil
}

// generate writes the spec file and the source checksums for one version.
// Output depends only on the definition and the version, so regenerating an
// unchanged package rewrites the same bytes and leaves no diff.
func generate(ctx context.Context, def Definition, dir, version string, release int, log []string, recompute bool, out io.Writer) error {
	vars := map[string]string{}
	if def.Vars != nil {
		v, err := def.Vars(ctx, version)
		if err != nil {
			return err
		}
		vars = v
	}
	spec := def.Spec(Release{Version: version, Vars: vars})
	spec.Name, spec.Version, spec.Release = def.Name, version, release
	spec.Changelog = log
	rendered, err := spec.Render()
	if err != nil {
		return err
	}
	if err := resolveChecksums(ctx, def, dir, version, spec, recompute); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, def.Name+".spec"), []byte(rendered), 0644); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s %s-%d\n", def.Name, version, release)
	return err
}

// resolveChecksums keeps the sources file in step with the spec: digests are
// reused while the version is unchanged, and recomputed from the network
// whenever the version moves or a refresh is requested.
func resolveChecksums(ctx context.Context, def Definition, dir, version string, spec Spec, recompute bool) error {
	path := filepath.Join(dir, "sources")
	known, err := readChecksums(path)
	if err != nil {
		return err
	}
	digests := map[string]string{}
	for _, source := range spec.RemoteSources() {
		if digest, ok := known[source.Name]; ok && !recompute {
			digests[source.Name] = digest
			continue
		}
		cache := filepath.Join(".cache", "sources", def.Name, version)
		digest, err := Digest(source, cache)
		if err != nil {
			return err
		}
		if def.Checksum != nil {
			expected, err := def.Checksum(ctx, version, source)
			if err != nil {
				return err
			}
			if expected != "" && expected != digest {
				return fmt.Errorf("%s: upstream publishes %s but downloaded %s", source.Name, expected, digest)
			}
		}
		digests[source.Name] = digest
	}
	return writeChecksums(path, digests)
}

// SourceArchive is a convenience for generators that need the upstream tarball
// of a version, for example to read a pinned toolchain out of it.
func SourceArchive(name, version string, source Source) (string, error) {
	if !strings.HasPrefix(source.URL, "https://") {
		return "", fmt.Errorf("%s: source is not remote", source.Name)
	}
	return Fetch(source, filepath.Join(".cache", "sources", name, version))
}

// onOff renders a flag the way copr-cli spells it.
func onOff(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}
