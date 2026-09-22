package core

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Minute}

// Get performs a plain HTTPS GET. Callers never send credentials to source
// hosts; only the release providers talk to authenticated APIs.
func Get(url string) (io.ReadCloser, error) {
	if !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("non-HTTPS URL %q", url)
	}
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}

// Digest downloads a source and returns its SHA256, caching the file so that
// repeated generator runs and a later archive inspection do not refetch it.
func Digest(source Source, cache string) (string, error) {
	if filepath.Base(source.Name) != source.Name || source.Name == "" || source.Name == "." {
		return "", fmt.Errorf("invalid source filename %q", source.Name)
	}
	path, err := Fetch(source, cache)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Fetch downloads a source into cache unless it is already there, and returns
// its path.
func Fetch(source Source, cache string) (string, error) {
	path := filepath.Join(cache, source.Name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.MkdirAll(cache, 0755); err != nil {
		return "", err
	}
	body, err := Get(source.URL)
	if err != nil {
		return "", err
	}
	defer body.Close()
	f, err := os.CreateTemp(cache, ".download-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	_, copyErr := io.Copy(f, body)
	closeErr := f.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return "", err
	}
	return path, nil
}

// FileFromTarGz returns the first file in a gzipped tarball whose name matches.
// Generators use it to read values out of an upstream source tree.
func FileFromTarGz(path string, match func(name string) bool) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	for {
		h, err := reader.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%s: no matching file in archive", filepath.Base(path))
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag == tar.TypeReg && match(h.Name) {
			return io.ReadAll(io.LimitReader(reader, 1<<20))
		}
	}
}

// checksums is the dist-git style file the source RPM build verifies before
// handing anything to rpmbuild.
func readChecksums(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	digests := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("%s: malformed line %q", path, line)
		}
		digests[fields[1]] = fields[0]
	}
	return digests, nil
}

func writeChecksums(path string, digests map[string]string) error {
	names := make([]string, 0, len(digests))
	for name := range digests {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		fmt.Fprintf(&b, "%s  %s\n", digests[name], name)
	}
	if b.Len() == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}
