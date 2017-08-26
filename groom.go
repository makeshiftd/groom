package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/makeshiftd/groom/internal/template"
	"github.com/xyplane/debugger"
)

var debug = debugger.Debug("groom")

func main() {
	os.Exit(groom(os.Args[1:]))
}

func groom(args []string) int {

	data, paths := parseArgs(args)

	debug("Data:", data)
	debug("Paths:", paths)

	var err error
	var tmpl *template.Template

	if len(paths) == 0 {
		buf, rerr := ioutil.ReadAll(os.Stdin)
		if rerr != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		name := "stdin.grm"
		path := filepath.Join(".", name)
		tmpl, err = template.New(funcs, false).ParseText(name, path, string(buf))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	} else {
		for _, path := range paths {
			name := filepath.Base(path)
			if tmpl == nil {
				debug("New Parse File:", name, path)
				tmpl, err = template.New(funcs, false).ParseFile(name, path)
			} else {
				debug("Parse File:", name, path)
				_, err = tmpl.ParseFile(name, path)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
	}

	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}

var ARG_DATA_REGEX = regexp.MustCompile("^--?((.*)\\s*=\\s*(.*?)\\s*|(.*?)\\s*)$")

func parseArgs(args []string) (map[string]string, []string) {
	data := map[string]string{}
	paths := []string{}
	for _, arg := range args {
		debug("Parse Arg:", arg)
		if !strings.HasPrefix(arg, "-") {
			paths = append(paths, arg)
			continue
		}
		match := ARG_DATA_REGEX.FindStringSubmatch(arg)
		if len(match) > 0 {
			if match[2] != "" {
				debug("Match Data: '%s'='%s'", match[2], match[3])
				data[match[2]] = match[3]
			} else if match[4] != "" {
				debug("Match Data: '%s'=''", match[4])
				data[match[4]] = ""
			}
		}
	}
	return data, paths
}
