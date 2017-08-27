package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/makeshiftd/groom/internal/template"
	"github.com/russross/blackfriday"
)

var funcs = template.FuncMap{
	"cat":      catFunc,
	"exec":     execFunc,
	"json":     jsonFunc,
	"stdin":    stdinFunc,
	"str":      strFunc,
	"markdown": markdownFunc,
}

func catFunc(args ...interface{}) ([]byte, error) {
	var data []byte
	for _, arg := range args {
		switch arg := arg.(type) {
		case string:
			buf, err := ioutil.ReadFile(arg)
			if err != nil {
				return nil, err
			}
			data = append(data, buf...)
		case []byte:
			buf, err := ioutil.ReadFile(string(arg))
			if err != nil {
				return nil, err
			}
			data = append(data, buf...)
		default:
			return nil, fmt.Errorf("groom: cat: unsupported type: %T", arg)
		}
	}
	return data, nil
}

func execFunc(args ...interface{}) ([]byte, error) {
	var strargs []string
	for _, arg := range args {
		switch arg := arg.(type) {
		case string:
			strargs = append(strargs, arg)
		case []byte:
			strargs = append(strargs, string(arg))
		default:
			return nil, fmt.Errorf("groom: exec: unsupported type: %T", arg)
		}
	}
	return exec.Command(strargs[0], strargs[1:]...).Output()
}

func jsonFunc(arg interface{}) (interface{}, error) {
	var v interface{}
	switch arg := arg.(type) {
	case string:
		err := json.Unmarshal([]byte(arg), &v)
		return v, err
	case []byte:
		err := json.Unmarshal(arg, &v)
		return v, err
	default:
		return json.Marshal(arg)
	}
}

func strFunc(arg interface{}) (string, error) {
	switch arg := arg.(type) {
	case string:
		return arg, nil
	case []byte:
		return string(arg), nil
	default:
		return "", fmt.Errorf("groom: str: unsupported type: %T", arg)
	}
}

func stdinFunc() ([]byte, error) {
	return ioutil.ReadAll(os.Stdin)
}

func markdownFunc(arg interface{}) (interface{}, error) {
	switch arg := arg.(type) {
	case string:
		return string(blackfriday.MarkdownCommon([]byte(arg))), nil
	case []byte:
		return blackfriday.MarkdownCommon(arg), nil
	default:
		return nil, fmt.Errorf("groom: markdown: unsupported type: %T", arg)
	}
}
