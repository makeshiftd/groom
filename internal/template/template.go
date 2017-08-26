package template

import (
	"errors"
	htemplate "html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	ttemplate "github.com/makeshiftd/groom/internal/template/text/template"
	"github.com/xyplane/debugger"
)

type FuncMap map[string]interface{}

type Template struct {
	safe  bool
	tmpl  interface{}
	funcs FuncMap
}

var debug = debugger.Debug("template")

func New(funcs FuncMap, safe bool) *Template {
	return &Template{funcs: funcs, safe: safe}
}

func (t *Template) ParseFile(name, path string) (*Template, error) {
	return t.parseFileWithLevel(name, path, 0)
}

func (t *Template) parseFileWithLevel(name, path string, level int) (*Template, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf, rerr := ioutil.ReadAll(f)
	if rerr != nil {
		return nil, rerr
	}
	return t.parseTextWithLevel(name, path, string(buf), level)
}

func (t *Template) ParseText(name, path, text string) (tt *Template, err error) {
	return t.parseTextWithLevel(name, path, text, 0)
}

func (t *Template) parseTextWithLevel(name, path, text string, level int) (tt *Template, err error) {
	if level == 1000 {
		return nil, errors.New("template import level exceeded")
	}
	level += 1

	imps, ierr := parseImports(text)
	if ierr != nil {
		return nil, ierr
	}

	dir := filepath.Dir(path)
	for name, path := range imps {
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		_, err := t.parseFileWithLevel(name, path, level)
		if err != nil {
			return nil, err
		}
	}

	tt, err = t.parse(name, text)
	if err != nil {
		return nil, err
	}
	return tt, nil
}

func (t *Template) Lookup(name string) *Template {
	switch tmpl := t.tmpl.(type) {
	case *ttemplate.Template:
		tmpl = tmpl.Lookup(name)
		if tmpl == nil {
			return nil
		}
		return &Template{tmpl: tmpl, safe: t.safe, funcs: t.funcs}
	case *htemplate.Template:
		tmpl = tmpl.Lookup(name)
		if tmpl == nil {
			return nil
		}
		return &Template{tmpl: tmpl, safe: t.safe, funcs: t.funcs}
	default:
		return nil
	}
}

func (t *Template) Name() string {
	switch tmpl := t.tmpl.(type) {
	case *ttemplate.Template:
		return tmpl.Name()
	case *htemplate.Template:
		return tmpl.Name()
	default:
		return ""
	}
}

func (t *Template) Execute(w io.Writer, data interface{}) error {
	switch tmpl := t.tmpl.(type) {
	case *ttemplate.Template:
		return tmpl.Execute(w, data)
	case *htemplate.Template:
		return tmpl.Execute(w, data)
	default:
		return nil
	}
}

func (t *Template) parse(name, text string) (*Template, error) {
	switch tmpl := t.tmpl.(type) {
	case *ttemplate.Template:
		debug("Parse:", name)
		tmpl, err := tmpl.New(name).Parse(text)
		if err != nil {
			return nil, err
		}
		return &Template{tmpl: tmpl, safe: t.safe, funcs: t.funcs}, nil
	case *htemplate.Template:
		debug("Parse:", name)
		tt, err := tmpl.New(name).Parse(text)
		if tt == nil {
			return nil, err
		}
		return &Template{tmpl: tmpl, safe: t.safe, funcs: t.funcs}, nil
	default:
		if t.safe {
			debug("New:", name)
			funcs := htemplate.FuncMap(t.funcs)
			tmpl, err := htemplate.New(name).Funcs(funcs).Parse(text)
			if err != nil {
				return nil, err
			}
			t.tmpl = tmpl
		} else {
			debug("New:", name)
			funcs := ttemplate.FuncMap(t.funcs)
			tmpl, err := ttemplate.New(name).Funcs(funcs).Parse(text)
			if err != nil {
				return nil, err
			}
			t.tmpl = tmpl
		}
		return t, nil
	}
}

var COMMENT_REGEXP = regexp.MustCompile("^\\s*{{\\s*/\\*\\s*(.*?)\\s*\\*/}}")

var IMPORT_REGEXP = regexp.MustCompile("^\\+import\\s+(([^\\s]+)\\s+)?\"([^\"]+)\"\\s*$")

func parseImports(text string) (map[string]string, error) {
	imps := map[string]string{}
	for {
		comment := COMMENT_REGEXP.FindStringSubmatchIndex(text)
		if comment == nil {
			break
		}

		content := text[comment[2]:comment[3]]
		if strings.HasPrefix(content, "+import") {
			istmt := IMPORT_REGEXP.FindStringSubmatch(content)
			if istmt == nil {
				return nil, errors.New("template import error: " + content)
			}

			name, path := istmt[2], filepath.FromSlash(istmt[3])
			if filepath.Ext(path) == "" {
				path += ".grm"
			}
			if name == "" {
				name = nameFromPath(path)
			}

			if p, ok := imps[name]; ok && p != path {
				return nil, errors.New("duplicate template: " + name)
			}
			imps[name] = path
		}
		text = text[comment[1]:]
	}
	return imps, nil
}

func nameFromPath(path string) string {
	name := filepath.Base(path)
	for ext := filepath.Ext(name); ext != ""; ext = filepath.Ext(name) {
		name = name[:len(name)-len(ext)]
	}
	return name
}
