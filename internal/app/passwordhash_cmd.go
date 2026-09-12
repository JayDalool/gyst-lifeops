package app

import (
	"bufio"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
	"strings"
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
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter password: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("failed to read password: %v", err)
	}
	password = strings.TrimSpace(password)

	if password == "" {
		log.Fatal("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	fmt.Printf("\nPassword hash:\n%s\n", string(hash))
	fmt.Println("\nAdd this to your .env file as:")
	fmt.Printf("GYST_PASSWORD_HASH=%s\n", string(hash))
	os.Exit(0)
}
