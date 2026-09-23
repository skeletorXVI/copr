package core

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// The changelog is here for reproducibility rather than for prose. Fedora
// derives SOURCE_DATE_EPOCH from the newest entry's date and clamps build
// mtimes to it, so a date that is written once and then committed is what
// makes two builds of the same NEVR agree. An entry is therefore treated like
// the release number: assigned by the generator, and preserved from the
// committed spec ever after.

// defaultBotName is the only identity default. The address is who the packages
// are published under, and the one the release workflow guards its update
// branches with, so guessing it here would be worse than refusing to write the
// entry at all.
const defaultBotName = "copr-bot"

// ErrNoAuthor is returned when a new changelog entry is needed and no author
// address is configured.
var ErrNoAuthor = errors.New("RELEASE_BOT_EMAIL is not set: it authors the changelog entry this version needs")

// now is the clock a new entry is dated by, replaced in tests.
var now = func() time.Time { return time.Now().UTC() }

// author is the identity entries are written under, from the same variables
// the release workflow commits with.
func author() (string, error) {
	email := os.Getenv("RELEASE_BOT_EMAIL")
	if email == "" {
		return "", ErrNoAuthor
	}
	name := os.Getenv("RELEASE_BOT_NAME")
	if name == "" {
		name = defaultBotName
	}
	return fmt.Sprintf("%s <%s>", name, email), nil
}

// entryDate is how a changelog entry spells its date.
const entryDate = "Mon Jan 02 2006"

// changelog returns the entries for version-release: the committed ones, with
// a new entry prepended unless the newest already describes this build. So
// regenerating what is packaged rewrites the same bytes, and only a version or
// release that actually moved is dated anew. An author is needed only when an
// entry is actually written, so regenerating what is already packaged requires
// no identity at all.
//
// A new version is dated by when upstream released it, which released looks
// up only when that entry is written. A new release of the same version is a
// packaging event, and is dated by the clock.
func changelog(committed []string, packaged, version string, release int, released func() (time.Time, error)) ([]string, error) {
	stamp := fmt.Sprintf(" - %s-%d", version, release)
	if len(committed) > 0 && strings.HasSuffix(committed[0], stamp) {
		return committed, nil
	}
	who, err := author()
	if err != nil {
		return nil, err
	}
	// Only the release moved, so nothing about the packaged software changed.
	what, date := "Update to "+version, now()
	if len(committed) > 0 && packaged == version {
		what = "Rebuild for packaging changes"
	} else if date, err = released(); err != nil {
		return nil, err
	}
	// Entries must stay newest first, so a release published before the last
	// packaging rebuild is dated no earlier than that rebuild.
	if len(committed) > 0 {
		if fields := strings.Fields(committed[0]); len(fields) >= 5 {
			if newest, err := time.Parse(entryDate, strings.Join(fields[1:5], " ")); err == nil && date.Before(newest) {
				date = newest
			}
		}
	}
	entry := []string{
		fmt.Sprintf("* %s %s%s", date.UTC().Format(entryDate), who, stamp),
		"- " + what,
	}
	if len(committed) == 0 {
		return entry, nil
	}
	return append(append(entry, ""), committed...), nil
}

// committedChangelog is the %changelog body of a spec file, which the
// generator carries forward rather than regenerating.
func committedChangelog(spec string) []string {
	const header = "\n%changelog\n"
	i := strings.Index(spec, header)
	if i < 0 {
		return nil
	}
	body := strings.Trim(spec[i+len(header):], "\n")
	if body == "" {
		return nil
	}
	return strings.Split(body, "\n")
}
