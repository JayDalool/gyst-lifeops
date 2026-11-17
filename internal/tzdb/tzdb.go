// Package tzdb is GYST's pinned copy of the IANA time zone database (Issue
// #193). internal/clock loads the application timezone only from here.
//
// Go's time.LoadLocation prefers the host's /usr/share/zoneinfo and falls
// back to the copy embedded in the Go toolchain, so the same binary gave
// different Winnipeg times depending on the machine: tzdb 2026e (Manitoba on
// permanent UTC-05 from 2026-11-01) where the OS had it, Go 1.25's 2025b
// (still falling back to UTC-06) where it did not. This package never reads
// the host, $ZONEINFO or Go's copy, so production, CI and every developer
// machine compute the same local times from one reviewed release.
//
// zoneinfo.zip is built by scripts/tzdb/update.sh with Go's own
// lib/time/update.bash recipe; PROVENANCE records the release, the source
// hashes, the signing key and the zip's SHA-256. Refresh it when IANA
// publishes a rule change that affects GYST users.
package tzdb

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

//go:embed zoneinfo.zip
var zoneinfoZip []byte

//go:embed PROVENANCE
var provenance string

var (
	indexOnce sync.Once
	index     map[string]*zip.File
	indexErr  error
)

func zones() (map[string]*zip.File, error) {
	indexOnce.Do(func() {
		reader, err := zip.NewReader(bytes.NewReader(zoneinfoZip), int64(len(zoneinfoZip)))
		if err != nil {
			indexErr = fmt.Errorf("tzdb: read embedded zoneinfo.zip: %w", err)
			return
		}
		index = make(map[string]*zip.File, len(reader.File))
		for _, file := range reader.File {
			index[file.Name] = file
		}
	})
	return index, indexErr
}

// Version is the pinned IANA release, for example "2026e".
func Version() string {
	return Field("tzdb_version")
}

// SHA256 is the hex SHA-256 of the embedded zoneinfo.zip. PROVENANCE pins
// the same value.
func SHA256() string {
	sum := sha256.Sum256(zoneinfoZip)
	return hex.EncodeToString(sum[:])
}

// Field returns one key=value line of PROVENANCE, or "" if it is absent.
func Field(key string) string {
	for _, line := range strings.Split(provenance, "\n") {
		line = strings.TrimSpace(line)
		if value, ok := strings.CutPrefix(line, key+"="); ok {
			return value
		}
	}
	return ""
}

// Load returns the named zone from the pinned database. As with
// time.LoadLocation, "UTC" is time.UTC. Unlike it, "" and "Local" are
// errors, and nothing outside the binary is consulted.
func Load(name string) (*time.Location, error) {
	if name == "UTC" {
		return time.UTC, nil
	}
	if !validName(name) {
		return nil, fmt.Errorf("invalid time zone name %q", name)
	}
	all, err := zones()
	if err != nil {
		return nil, err
	}
	file, ok := all[name]
	if !ok {
		return nil, fmt.Errorf("unknown time zone %s (tzdb %s)", name, Version())
	}
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("tzdb: open %s: %w", name, err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("tzdb: read %s: %w", name, err)
	}
	return time.LoadLocationFromTZData(name, data)
}

// validName accepts what an IANA zone name can look like (tz "Theory"):
// slash-separated non-empty elements of ASCII letters, digits and "._+-",
// none of them "." or "..".
func validName(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}
	for _, element := range strings.Split(name, "/") {
		if element == "" || element == "." || element == ".." {
			return false
		}
		for _, r := range element {
			switch {
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			case r == '.' || r == '_' || r == '+' || r == '-':
			default:
				return false
			}
		}
	}
	return true
}
