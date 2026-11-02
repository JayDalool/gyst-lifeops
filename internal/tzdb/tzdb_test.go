package tzdb

import (
	"archive/zip"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func zoneBytes(t *testing.T, name string) []byte {
	t.Helper()
	all, err := zones()
	if err != nil {
		t.Fatal(err)
	}
	file, ok := all[name]
	if !ok {
		t.Fatalf("%s is not in the pinned database", name)
	}
	reader, err := file.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// PROVENANCE pins the embedded zip: a swapped or hand-edited zoneinfo.zip
// fails here, and the pinned release has Manitoba's permanent UTC-05.
func TestProvenancePinsEmbeddedDatabase(t *testing.T) {
	if got, want := SHA256(), Field("zoneinfo_zip_sha256"); got != want {
		t.Fatalf("zoneinfo.zip sha256 = %s; PROVENANCE pins %s (rebuild with scripts/tzdb/update.sh)", got, want)
	}
	version := Version()
	if !regexp.MustCompile(`^[0-9]{4}[a-z]$`).MatchString(version) || version < "2026e" {
		t.Fatalf("pinned tzdb version %q; want 2026e or later", version)
	}
	for _, key := range []string{"source", "tzcode", "tzcode_sha512", "tzdata", "tzdata_sha512", "signing_key", "build", "zip"} {
		if Field(key) == "" {
			t.Errorf("PROVENANCE has no %s", key)
		}
	}
	if !strings.Contains(Field("tzdata"), version) || !strings.Contains(Field("tzcode"), version) {
		t.Errorf("PROVENANCE sources %q, %q are not release %s", Field("tzdata"), Field("tzcode"), version)
	}
	all, err := zones()
	if err != nil {
		t.Fatal(err)
	}
	if count, _ := strconv.Atoi(Field("zones")); count != len(all) || count < 400 {
		t.Fatalf("PROVENANCE zones=%s; the zip has %d", Field("zones"), len(all))
	}
}

// Every entry is a TZif zone that Load accepts by name.
func TestEveryEmbeddedZoneLoads(t *testing.T) {
	all, err := zones()
	if err != nil {
		t.Fatal(err)
	}
	for name := range all {
		if !validName(name) {
			t.Errorf("%s: rejected by validName", name)
			continue
		}
		location, err := Load(name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if location.String() != name && name != "UTC" {
			t.Errorf("%s loaded as %s", name, location)
		}
	}
}

func TestLoadRejectsInvalidNames(t *testing.T) {
	for _, name := range []string{
		"", "Local", "Mars/Base", "america/winnipeg", "AMERICA/WINNIPEG", " America/Winnipeg",
		"../etc/passwd", "/etc/passwd", "/America/Winnipeg", "America/../Etc/UTC", "./UTC",
		"America//Winnipeg", "America/Winnipeg/", `America\Winnipeg`, "America/Winnipeg\x00",
		"PROVENANCE", "zoneinfo.zip", "posixrules", "right/America/Winnipeg", strings.Repeat("A", 300),
	} {
		if location, err := Load(name); err == nil {
			t.Errorf("Load(%q) = %v; want an error", name, location)
		}
	}
	if location, err := Load("UTC"); err != nil || location != time.UTC {
		t.Fatalf("Load(UTC) = %v, %v; want time.UTC", location, err)
	}
}

// The host cannot change the zones GYST uses. ZONEINFO is checked first by
// time.LoadLocation, so a planted zip that says Winnipeg is UTC+09 fools Go's
// loader (shown, so the test cannot pass vacuously) but not this package.
// It runs in a child process because the time package reads ZONEINFO once.
func TestHostZoneFilesCannotOverridePinnedData(t *testing.T) {
	if os.Getenv("GYST_TZDB_CHILD") == "1" {
		at := time.Date(2027, 1, 15, 12, 0, 0, 0, time.UTC)
		planted, err := time.LoadLocation("America/Winnipeg")
		if err != nil {
			fmt.Println("planted zip not loaded:", err)
			os.Exit(3)
		}
		pinned, err := Load("America/Winnipeg")
		if err != nil {
			fmt.Println("pinned load failed:", err)
			os.Exit(3)
		}
		_, plantedOffset := at.In(planted).Zone()
		_, pinnedOffset := at.In(pinned).Zone()
		fmt.Printf("planted=%d pinned=%d\n", plantedOffset, pinnedOffset)
		os.Exit(0)
	}
	var fake bytes.Buffer
	zw := zip.NewWriter(&fake)
	tokyo := zoneBytes(t, "Asia/Tokyo")
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "America/Winnipeg", Method: zip.Store}) // ZONEINFO zips must be stored
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(tokyo); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	planted := filepath.Join(t.TempDir(), "planted.zip")
	if err := os.WriteFile(planted, fake.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestHostZoneFilesCannotOverridePinnedData$")
	child.Env = append(os.Environ(), "GYST_TZDB_CHILD=1", "ZONEINFO="+planted, "TZ=Asia/Tokyo")
	out, err := child.CombinedOutput()
	if err != nil {
		t.Fatalf("child: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "planted=32400 pinned=-18000") {
		t.Fatalf("want Go's loader fooled (+09:00) and the pinned zone at -05:00; child said:\n%s", out)
	}
}

// systemVersion reads "# version 2026e" from the first line of tzdata.zi.
func systemVersion(dir string) (string, error) {
	file, err := os.Open(filepath.Join(dir, "tzdata.zi"))
	if err != nil {
		return "", err
	}
	defer file.Close()
	line, err := bufio.NewReader(file).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	version, ok := strings.CutPrefix(strings.TrimSpace(line), "# version ")
	if !ok {
		return "", fmt.Errorf("no version line in %s/tzdata.zi", dir)
	}
	return version, nil
}

// firstDifference walks every zone boundary of a and b from from to to and
// describes the first instant where the UTC offset or abbreviation differ.
func firstDifference(a, b *time.Location, from, to time.Time) string {
	at := from
	for steps := 0; at.Before(to); steps++ {
		if steps > 100000 {
			return "zone walk did not finish"
		}
		aName, aOffset := at.In(a).Zone()
		bName, bOffset := at.In(b).Zone()
		if aName != bName || aOffset != bOffset {
			return fmt.Sprintf("at %s: pinned %s %+d, other %s %+d", at.UTC().Format(time.RFC3339), aName, aOffset, bName, bOffset)
		}
		var next time.Time
		for _, location := range []*time.Location{a, b} {
			_, end := at.In(location).ZoneBounds()
			if end.IsZero() {
				continue // this zone has no later change
			}
			if !end.After(at) {
				// For a rule-based period (the TZif footer) Go clips the
				// bounds to "start of year + 365 days", which in a leap
				// year is Dec 31 00:00 UTC, so ZoneBounds stops advancing
				// for that day. No rule changes clocks then; skip a day.
				end = at.Add(24 * time.Hour)
			}
			if next.IsZero() || end.Before(next) {
				next = end
			}
		}
		if next.IsZero() {
			return ""
		}
		at = next
	}
	return ""
}

// The comparison has teeth: it finds nothing between a zone and itself, and
// against the Go toolchain's older copy of America/Winnipeg it finds the
// 2026-11-01 fall-back that copy still has.
func TestFirstDifferenceFindsTheManitobaChange(t *testing.T) {
	pinned, err := Load("America/Winnipeg")
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	if diff := firstDifference(pinned, pinned, time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), to); diff != "" {
		t.Fatalf("a zone differs from itself: %s", diff)
	}
	path := filepath.Join(runtime.GOROOT(), "lib", "time", "zoneinfo.zip")
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Skipf("no Go toolchain zone database at %s: %v", path, err)
	}
	defer archive.Close()
	for _, file := range archive.File {
		if file.Name != "America/Winnipeg" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		old, err := time.LoadLocationFromTZData("America/Winnipeg", data)
		if err != nil {
			t.Fatal(err)
		}
		if _, offset := time.Date(2027, 1, 15, 12, 0, 0, 0, time.UTC).In(old).Zone(); offset == -5*60*60 {
			t.Skipf("the toolchain copy at %s already has permanent UTC-05", path)
		}
		if diff := firstDifference(pinned, old, from, to); !strings.Contains(diff, "at 2026-11-01T07:00:00Z") {
			t.Fatalf("pinned vs %s: first difference %q; want the 2026-11-01T07:00:00Z fall-back", path, diff)
		}
		return
	}
	t.Fatalf("%s has no America/Winnipeg", path)
}

// systemVNames are the old UNIX System V names that IANA turned into links
// in 2024b. Ubuntu 24.04 keeps their earlier definitions on purpose (its
// tzdata changelog: "Do not replace CET, CST6CDT, EET, EST*, HST, MET,
// MST*, PST8PDT, WET"), so a distribution build may differ on these names
// only. GYST follows upstream IANA.
var systemVNames = map[string]bool{
	"CET": true, "CST6CDT": true, "EET": true, "EST": true, "EST5EDT": true, "HST": true,
	"MET": true, "MST": true, "MST7MDT": true, "PST8PDT": true, "WET": true,
}

// The independent check (Issue #193, acceptance criterion 2): every pinned
// zone agrees, 1970-2100, with a separate build of the same IANA release,
// such as the OS package (Debian and Ubuntu compile theirs with glibc's
// zic). It runs wherever the system database is the pinned version, which
// includes the CI image (Ubuntu tzdata 2026e). GYST_TZDB_COMPARE_DIR points
// it at another zoneinfo directory. Before 1970 IANA does not promise that
// builds agree (backzone options differ: Ubuntu gives some old links their
// own pre-1970 history), and newer Debian and Ubuntu move old link names to
// a tzdata-legacy package, so missing names are skipped.
func TestPinnedAgreesWithSystemBuildOfSameRelease(t *testing.T) {
	dir := os.Getenv("GYST_TZDB_COMPARE_DIR")
	if dir == "" {
		dir = "/usr/share/zoneinfo"
	}
	version, err := systemVersion(dir)
	if err != nil {
		t.Skipf("no comparable zone database: %v", err)
	}
	if version != Version() {
		t.Skipf("%s is tzdb %s, pinned is %s: not comparable", dir, version, Version())
	}
	all, err := zones()
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	from := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	compared, sawWinnipeg := 0, false
	for _, name := range names {
		if systemVNames[name] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if errors.Is(err, fs.ErrNotExist) {
			continue // a distribution may leave out a backward link
		}
		if err != nil {
			t.Fatal(err)
		}
		other, err := time.LoadLocationFromTZData(name, data)
		if err != nil {
			t.Errorf("%s: system file: %v", name, err)
			continue
		}
		pinned, err := Load(name)
		if err != nil {
			t.Fatal(err)
		}
		if name == "UTC" {
			pinned, _ = time.LoadLocationFromTZData(name, zoneBytes(t, name))
		}
		if diff := firstDifference(pinned, other, from, to); diff != "" {
			t.Errorf("%s: %s", name, diff)
		}
		compared++
		sawWinnipeg = sawWinnipeg || name == "America/Winnipeg"
	}
	if !sawWinnipeg || compared < 300 {
		t.Fatalf("compared only %d of %d zones (America/Winnipeg: %v) with %s", compared, len(names), sawWinnipeg, dir)
	}
	t.Logf("pinned tzdb %s agrees with %s (tzdb %s) for %d of %d zones, 1970-2100 (%d System V names not compared)",
		Version(), dir, version, compared, len(names), len(systemVNames))
}
