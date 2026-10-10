package app

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
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
