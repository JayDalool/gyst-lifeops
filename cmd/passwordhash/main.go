// Command passwordhash generates the bcrypt hash for GYST_PASSWORD_HASH.
//
// The password is read from stdin rather than a flag so it does not land in
// the shell history or in the process list:
//
//	go run ./cmd/passwordhash
//
// Only the hash is printed. Put it in .env; never commit it.
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"lifeops-tracker/internal/services"
)

func main() {
	if err := run(os.Stdin, os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}

// run reads one line from in, writes the prompt to prompt and only the
// GYST_PASSWORD_HASH= line to out. The password is never written.
func run(in io.Reader, out, prompt io.Writer) error {
	fmt.Fprint(prompt, "Password: ")

	reader := bufio.NewReader(in)
	password, err := reader.ReadString('\n')
	if err != nil && password == "" {
		return fmt.Errorf("read password: %w", err)
	}
	// Accept both a piped value and an interactive line. Only the line ending
	// is stripped, so the hash matches what is typed at login.
	password = strings.TrimRight(password, "\r\n")
	fmt.Fprintln(prompt)

	hash, err := services.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	fmt.Fprintf(out, "GYST_PASSWORD_HASH=%s\n", hash)
	return nil
}
