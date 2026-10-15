// One-off helper: prints an Argon2id hash in the backend's encoded format for
// a given password, to be pasted into the local dev database. Usage:
//
//	go run ./cmd/setpassword -p 'ChangeMe123!'
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

func main() {
	password := flag.String("p", "", "plaintext password to hash")
	flag.Parse()
	if *password == "" {
		fmt.Fprintln(os.Stderr, "usage: setpassword -p <password>")
		os.Exit(2)
	}
	hash, err := auth.HashPassword(*password)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(hash)
}
