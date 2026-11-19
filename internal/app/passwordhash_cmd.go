package app

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"lifeops-tracker/internal/services"
)

func passwordHashRequested(args []string) bool {
	for _, arg := range args {
		if arg == "passwordhash" {
			return true
		}
	}
	return false
}

// passwordHashFlagRequested reports the legacy -passwordhash / --passwordhash
// flag (optionally =true or =1) before any other setting is loaded. It stops at
// "--" and at the first non-flag argument, and skips the value of a
// value-taking flag, so `-db -passwordhash` is still a database path.
func passwordHashFlagRequested(args []string) bool {
	valueFlags := map[string]bool{"addr": true, "db": true, "migrations": true, "seed-file": true, "templates": true, "static": true}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" || arg == "-" || !strings.HasPrefix(arg, "-") {
			return false
		}
		name, value, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		switch {
		case name == "passwordhash":
			if !hasValue {
				return true
			}
			on, err := strconv.ParseBool(value)
			return err == nil && on
		case valueFlags[name] && !hasValue:
			i++
		}
	}
	return false
}

func runPasswordHashAndExit() {
	if err := writePasswordHash(os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

// writePasswordHash reads one line from in and writes the prompt and the hash
// to out. Only the trailing line ending is stripped, so the hash matches what
// is typed at login. The password itself is never written.
func writePasswordHash(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)

	fmt.Fprint(out, "Enter password: ")
	password, err := reader.ReadString('\n')
	if err != nil && password == "" {
		return fmt.Errorf("failed to read password: %w", err)
	}
	password = strings.TrimRight(password, "\r\n")

	hash, err := services.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	fmt.Fprintf(out, "\nPassword hash:\n%s\n", hash)
	fmt.Fprintln(out, "\nAdd this to your .env file as:")
	fmt.Fprintf(out, "GYST_PASSWORD_HASH=%s\n", hash)
	return nil
}
