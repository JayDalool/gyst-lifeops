package app

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/tzdb"
)

// tzinfoRequested reports whether the process was invoked as
// `lifeops tzinfo`. Only the first argument counts.
func tzinfoRequested(args []string) bool {
	return len(args) > 1 && args[1] == "tzinfo"
}

// hostZoneinfoDir is where Debian-style images keep zone files. GYST never
// loads zones from it; tzinfo only reports what it holds.
var hostZoneinfoDir = "/usr/share/zoneinfo"

// RunTzinfo writes GYST's time zone facts (Issue #193): the pinned database
// in the binary, the application timezone with the current local time and
// day, and the known-offset probe from clock.DefaultZoneProbes. It is
// read-only: it opens no database, makes no network call and is given only
// the GYST_TIMEZONE value, which it prints (a zone name is not a secret).
// With -check it returns an error if the zone is invalid or the probe fails.
func RunTzinfo(w io.Writer, args []string, timezone string, now time.Time) error {
	fs := flag.NewFlagSet("tzinfo", flag.ContinueOnError)
	fs.SetOutput(w)
	check := fs.Bool("check", false, "exit non-zero if the zone is invalid or the probe fails")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		// "tzinfo check" (no dash) must not look like a passing check.
		return fmt.Errorf("unexpected argument %q (flags: -check)", fs.Arg(0))
	}
	fmt.Fprintf(w, "tzdb: %s, pinned in the binary (zoneinfo.zip sha256 %s)\n", tzdb.Version(), tzdb.SHA256())
	fmt.Fprintf(w, "tzdb source: %s%s, signed by %s\n", tzdb.Field("source"), tzdb.Field("tzdata"), tzdb.Field("signing_key"))
	fmt.Fprintf(w, "go: %s\n", runtime.Version())

	var failures []error
	setting := fmt.Sprintf("%q", strings.TrimSpace(timezone))
	if strings.TrimSpace(timezone) == "" {
		setting = "unset, default " + clock.DefaultTimezone
	}
	location, err := clock.Load(timezone)
	if err != nil {
		fmt.Fprintf(w, "application timezone: INVALID (GYST_TIMEZONE %s): %v\n", setting, err)
		failures = append(failures, err)
	} else {
		c := clock.Clock{Now: func() time.Time { return now }, Loc: location}
		local := c.InLocal(now)
		fmt.Fprintf(w, "application timezone: %s (GYST_TIMEZONE %s)\n", location, setting)
		fmt.Fprintf(w, "now: %s UTC = %s local, UTC offset %s, local day %s\n",
			now.UTC().Format(time.RFC3339), local.Format("2006-01-02 15:04:05"), local.Format("-07:00"), c.TodayString())
	}
	if err := clock.CheckDefaultZone(); err != nil {
		fmt.Fprintf(w, "probe: %s FAILED\n%v\n", clock.DefaultTimezone, err)
		failures = append(failures, err)
	} else {
		fmt.Fprintf(w, "probe: %s, %d known UTC offsets (history and permanent UTC-05): OK\n", clock.DefaultTimezone, len(clock.DefaultZoneProbes))
	}
	fmt.Fprintf(w, "host %s: %s (not used by GYST)\n", hostZoneinfoDir, hostZoneinfo(hostZoneinfoDir))
	if *check {
		return errors.Join(failures...)
	}
	return nil
}

// hostZoneinfo says whether dir holds a zone database and which release
// its tzdata.zi names.
func hostZoneinfo(dir string) string {
	file, err := os.Open(dir + "/tzdata.zi")
	if err != nil {
		if _, statErr := os.Stat(dir); statErr != nil {
			return "absent"
		}
		return "present, release unknown (no tzdata.zi)"
	}
	defer file.Close()
	line, _ := bufio.NewReader(file).ReadString('\n')
	if version, ok := strings.CutPrefix(strings.TrimSpace(line), "# version "); ok {
		return "present, tzdb " + version
	}
	return "present, release unknown"
}

func runTzinfoAndExit() {
	err := RunTzinfo(os.Stdout, os.Args[2:], os.Getenv("GYST_TIMEZONE"), clock.System().Now())
	switch {
	case err == nil:
		return
	case errors.Is(err, flag.ErrHelp):
		return
	default:
		fmt.Fprintf(os.Stderr, "tzinfo: %v\n", err)
		os.Exit(1)
	}
}
