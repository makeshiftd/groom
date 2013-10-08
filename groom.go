//
// Groom is a command-line tool similar to 'mustache',
// except written in Go and using Go templates.
package main

import "io"
import "os"
import "fmt"
import "bytes"

var Version = "0.1.1a"

var cfgReg = NewConfigRegistry()

func main() {
	cfgs, err := cfgReg.Process();
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	var t Template
	if cfgReg.IsHtml() {
		fmt.Fprintln(os.Stderr, "HTML safe templates not supported (yet)")
		os.Exit(-1)
	} else {
		t = NewTextTemplate()
	}

	err = InitDefaultData(t)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error", err)
		os.Exit(-1)
	}

	err = InitDefaultFuncs(t)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error", err)
		os.Exit(-1)
	}

	for _, cfg := range cfgs {
		err = cfg.Configure(t)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Configuring", err)
			os.Exit(-1)
		}
	}

	var dest io.Writer = os.Stdout 
	if cfgReg.Dest() != "-" {
		stat, serr := os.Stat(cfgReg.Dest())
		if serr == nil {
			if stat.IsDir() {
				fmt.Fprintf(os.Stderr, "destination is a directory: %s\n", cfgReg.Dest())
				os.Exit(-1)
			}
			if !cfgReg.IsForce() {
				fmt.Fprintf(os.Stderr, "destination already exists: %s\n", cfgReg.Dest())
				os.Exit(-1)
			}
		}
		file, oerr := os.Create(cfgReg.Dest())
		if oerr != nil {
			fmt.Fprintf(os.Stderr, "destination invalid: %s\n", cfgReg.Dest())
			os.Exit(-1)
		}
		dest = file
	}

	if cfgReg.IsBuffer() {
		buf := &bytes.Buffer{}
		err = t.Execute(buf)
		if err == nil {
			_, err = buf.WriteTo(dest)
		}
	} else {
		err = t.Execute(dest)
	}

	if c, ok := dest.(io.Closer); ok {
		c.Close()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Executing", err)
		os.Exit(-1)
	}

	os.Exit(0)
}
