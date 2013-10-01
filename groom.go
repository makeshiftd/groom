//
// Groom is a command-line tool similar to 'mustache',
// except written in Go and using Go templates.
package main

import "fmt"

var Version = "0.0.1"

var cfgReg = NewConfigRegistry()

func main() {
	cfgReg.Process();

	if cfgReg.IsHtml() {
		fmt.Println("HTML safe templates requested.")
	} else {
		fmt.Println("Standard text templates requested")
	}
}
