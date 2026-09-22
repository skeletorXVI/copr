// Zen Browser is repackaged from the official Linux binary release.
package main

import (
	"context"

	"copr.local/automation/internal/core"
	"copr.local/automation/internal/core/github"
)

const (
	repo  = "zen-browser/desktop"
	asset = "zen.linux-x86_64.tar.xz"
)

func main() {
	core.Run(core.Definition{
		Project:  "zen-browser",
		Name:     "zen-browser",
		Upstream: "https://github.com/zen-browser/desktop",
		Policy:   core.ReviewAll{},
		Chroots: []string{
			"fedora-43-x86_64", "fedora-44-x86_64",
			"fedora-45-x86_64", "fedora-rawhide-x86_64",
		},
		// Repackaged binaries, so a full rebuild in a pull request is cheap
		// enough to prove the RPM installs and runs.
		Verify:   []string{"zen-browser", "--headless", "--version"},
		Releases: github.Latest{Repo: repo},
		// Zen publishes a SHA256 for its release assets; hold the download to it.
		Checksum: func(ctx context.Context, version string, source core.Source) (string, error) {
			return github.AssetDigest(ctx, repo, version, source.Name)
		},
		Spec: spec,
	})
}

func spec(r core.Release) core.Spec {
	return core.Spec{
		Globals: []core.Global{
			// The bundle is prebuilt; stripping it would break upstream's binaries.
			{Name: "debug_package", Value: "%{nil}"},
			{Name: "__strip", Value: "/bin/true"},
			{Name: "__brp_strip", Value: "/bin/true"},
			{Name: "__brp_strip_comment_note", Value: "/bin/true"},
			{Name: "__brp_strip_static_archive", Value: "/bin/true"},
			// Private libraries must not advertise system-wide provides, and
			// their internal requirements are filtered without hiding system ones.
			{Name: "__provides_exclude_from", Value: `^%{_libdir}/zen-browser/.*$`},
			{Name: "__requires_exclude", Value: `^lib(moz[^.]*|gkcodecs|lgpllibs|xul|onnxruntime)\\.so.*$`},
		},
		Summary: "Privacy-focused web browser based on Firefox",
		License: "MPL-2.0",
		URL:     "https://zen-browser.app",
		Sources: []core.Source{
			{Name: asset, URL: "https://github.com/" + repo + "/releases/download/" + r.Version + "/" + asset},
			{Name: "zen-browser.desktop"},
			{Name: "policies.json"},
		},
		ExclusiveArch: "x86_64",
		BuildRequires: []string{"desktop-file-utils", "patchelf"},
		Requires: []string{
			"gtk3", "dbus-glib", "alsa-lib", "libXt", "libX11-xcb", "libXcomposite",
			"libXdamage", "libXrandr", "mesa-libgbm", "nss", "nspr", "mozilla-filesystem",
		},
		Description: []string{
			"Zen Browser, repackaged from the official Linux binary release. Application",
			"updates are managed by RPM; browser profiles remain in the user's home directory.",
		},
		Prep:  []string{"%setup -q -n zen"},
		Build: []string{"# Upstream binaries are intentionally left intact."},
		Install: []string{
			"mkdir -p %{buildroot}%{_libdir}/%{name}",
			"cp -a . %{buildroot}%{_libdir}/%{name}/",
			"# Upstream ONNX carries a literal '$' RUNPATH, rejected by Fedora's RPATH check.",
			"# It needs no extra search directory in this private library installation.",
			"patchelf --remove-rpath %{buildroot}%{_libdir}/%{name}/libonnxruntime.so",
			"install -Dm644 %{SOURCE2} %{buildroot}%{_libdir}/%{name}/distribution/policies.json",
			"mkdir -p %{buildroot}%{_bindir}",
			"ln -s %{_libdir}/%{name}/zen %{buildroot}%{_bindir}/%{name}",
			"desktop-file-install --dir=%{buildroot}%{_datadir}/applications %{SOURCE1}",
			"for size in 16 32 48 64 128; do",
			`    install -Dm644 browser/chrome/icons/default/default${size}.png \`,
			"        %{buildroot}%{_datadir}/icons/hicolor/${size}x${size}/apps/%{name}.png",
			"done",
		},
		Check: []string{
			"desktop-file-validate %{buildroot}%{_datadir}/applications/%{name}.desktop",
			"test -x %{buildroot}%{_libdir}/%{name}/zen",
		},
		Files: []string{
			`%{_bindir}/%{name}`,
			`%{_libdir}/%{name}/`,
			`%{_datadir}/applications/%{name}.desktop`,
			`%{_datadir}/icons/hicolor/*/apps/%{name}.png`,
		},
	}
}
