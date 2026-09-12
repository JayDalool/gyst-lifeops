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
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is deliberately above the library default. This runs once per
// login on a machine serving one person, so a slower hash costs nothing that
// matters and buys real resistance to offline cracking if .env ever leaks.
const bcryptCost = 12

func main() {
	fmt.Fprint(os.Stderr, "Password: ")

	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil && password == "" {
		log.Fatalf("read password: %v", err)
	}
	// Accept both a piped value and an interactive line.
	password = strings.TrimRight(password, "\r\n")
	fmt.Fprintln(os.Stderr)

	if strings.TrimSpace(password) == "" {
		log.Fatal("password must not be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	fmt.Printf("GYST_PASSWORD_HASH=%s\n", hash)
}
