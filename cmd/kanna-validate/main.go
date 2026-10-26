// Command kanna-validate generates Validate methods from the validate tags of
// structs marked //kanna:validate.
package main

import (
	"os"

	"github.com/go-kanna/kanna/internal/gen/validate"
	"github.com/go-kanna/kanna/internal/version"
)

func main() {
	os.Exit(validate.NewCLI(version.String()).Run(os.Args[1:]))
}
