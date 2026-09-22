// Package github provides release sources for upstreams hosted on GitHub.
// Packages with a different kind of upstream implement core.Releases elsewhere;
// the core knows nothing about GitHub.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"copr.local/automation/internal/core"
)

const api = "https://api.github.com"

var client = &http.Client{Timeout: 2 * time.Minute}

// get calls the GitHub API, using GH_TOKEN when present purely for rate limits.
func get(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token := os.Getenv("GH_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(dest)
}

// Latest tracks a repository's latest non-prerelease GitHub release.
type Latest struct {
	Repo string
	// Vars adds package-specific values to the release, given its tag.
	Vars func(ctx context.Context, tag string) (map[string]string, error)
}

func (l Latest) LatestRelease(ctx context.Context) (core.Release, error) {
	var release struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := get(ctx, "/repos/"+l.Repo+"/releases/latest", &release); err != nil {
		return core.Release{}, err
	}
	if release.Draft || release.Prerelease {
		return core.Release{}, fmt.Errorf("%s: latest release is not stable", l.Repo)
	}
	return build(ctx, release.Tag, l.Vars)
}

// HighestTag selects the highest stable version tag. It exists for upstreams
// that publish tags without GitHub Releases.
type HighestTag struct {
	Repo string
	Vars func(ctx context.Context, tag string) (map[string]string, error)
}

func (h HighestTag) LatestRelease(ctx context.Context) (core.Release, error) {
	var tags []struct {
		Name string `json:"name"`
	}
	if err := get(ctx, "/repos/"+h.Repo+"/tags?per_page=100", &tags); err != nil {
		return core.Release{}, err
	}
	var best string
	var highest core.Version
	for _, t := range tags {
		// API order is not version order, so compare every tag.
		v, err := core.ParseVersion(t.Name)
		if err == nil && v.Suffix == "" && (best == "" || core.CompareVersions(v, highest) > 0) {
			best, highest = t.Name, v
		}
	}
	if best == "" {
		return core.Release{}, fmt.Errorf("%s has no stable version tags", h.Repo)
	}
	return build(ctx, best, h.Vars)
}

func build(ctx context.Context, tag string, vars func(context.Context, string) (map[string]string, error)) (core.Release, error) {
	if _, err := core.ParseVersion(tag); err != nil {
		return core.Release{}, err
	}
	release := core.Release{Version: strings.TrimPrefix(tag, "v")}
	if vars != nil {
		v, err := vars(ctx, tag)
		if err != nil {
			return core.Release{}, err
		}
		release.Vars = v
	}
	return release, nil
}

// AssetDigest returns the SHA256 GitHub publishes for a release asset, so a
// binary repackage can be verified against upstream's own digest on first pin.
func AssetDigest(ctx context.Context, repo, tag, asset string) (string, error) {
	var release struct {
		Assets []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := get(ctx, "/repos/"+repo+"/releases/tags/"+tag, &release); err != nil {
		return "", err
	}
	for _, a := range release.Assets {
		if a.Name == asset {
			return strings.TrimPrefix(a.Digest, "sha256:"), nil
		}
	}
	return "", fmt.Errorf("%s %s: no asset %q", repo, tag, asset)
}

// TarballURL is the source archive URL for a tag, renamed so the file is
// unambiguous once rpmbuild has it in the source directory.
func TarballURL(repo, tag, filename string) string {
	return fmt.Sprintf("https://github.com/%s/archive/refs/tags/%s.tar.gz#/%s", repo, tag, filename)
}
