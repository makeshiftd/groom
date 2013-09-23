//
// Groom is a command-line tool similar to 'mustache',
// except written in Go and using Go templates.
package main

import "fmt"

var Version = "0.0.0"

func main() {
	fmt.Printf("Welcome to Groom! (v%s)\n", Version)
}
