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
	"github.com/makeshiftd/groom/internal/template/text/template/parse"
	"github.com/xyplane/debugger"
)

type FuncMap map[string]interface{}

type Template struct {
	safe  bool
	tmpl  interface{}
	funcs FuncMap
}

var debug = debugger.Debug("groom:template")

func New(funcs FuncMap, safe bool) *Template {
	return &Template{funcs: funcs, safe: safe}
}

func (t *Template) ParseFile(name, path string) (*Template, error) {
	return t.parseFileWithLevel(name, path, 0)
}

func (t *Template) parseFileWithLevel(name, path string, level int) (*Template, error) {
	debug("Open path: %s", path)
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
	level++

	dir := filepath.Dir(path)

	trees, perr := parse.Parse(name, text, "{{", "}}", t.funcs)
	if perr != nil {
		return nil, perr
	}

	var stack []parse.Node

	push := func(nodes ...parse.Node) {
		for idx := len(nodes) - 1; idx >= 0; idx-- {
			stack = append(stack, nodes[idx])
		}
	}

	pop := func() (parse.Node, bool) {
		if len(stack) == 0 {
			return nil, false
		}
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return node, true
	}

	for _, tree := range trees {
		push(tree.Root.Nodes...)

		for node, ok := pop(); ok; node, ok = pop() {
			switch node := node.(type) {
			case *parse.TemplateNode:
				imp := strings.Split(node.Name, " ")
				var name, path string
				if len(imp) > 0 && (imp[0] == "import") {
					if len(imp) == 3 {
						name = imp[1]
						path = imp[2]
					} else if len(imp) == 2 {
						path = imp[1]
					} else {
						// ERROR
					}
					//name, path := istmt[2], filepath.FromSlash(istmt[3])
					if filepath.Ext(path) == "" {
						path += ".grm"
					}
					if name == "" {
						name = nameFromPath(path)
					}
					if !filepath.IsAbs(path) {
						path = filepath.Join(dir, path)
					}
					_, err := t.parseFileWithLevel(name, path, level)
					if err != nil {
						return nil, err
					}
					node.Name = name
				}
			case *parse.BranchNode:
				push(node.ElseList)
				push(node.List)
			case *parse.ApplyNode:
				push(node.ElseList)
				push(node.List)
			}
		}
	}

	for name, tree := range trees {
		_, err := t.addParseTree(name, tree)
		if err != nil {
			return nil, err
		}
	}

	return t.Lookup(name), nil
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

func (t *Template) addParseTree(name string, tree *parse.Tree) (*Template, error) {
	switch tmpl := t.tmpl.(type) {
	case *ttemplate.Template:
		debug("Parse:", name)
		tt, err := tmpl.AddParseTree(name, tree)
		if err != nil {
			return nil, err
		}
		return &Template{tmpl: tt, safe: t.safe, funcs: t.funcs}, nil
	// case *htemplate.Template:
	// 	debug("Parse:", name)
	// 	tt, err := tmpl.New(name).AddParseTree(name, tree)
	// 	if tt == nil {
	// 		return nil, err
	// 	}
	// 	return &Template{tmpl: tmpl, safe: t.safe, funcs: t.funcs}, nil
	default:
		if t.safe {
			// debug("New:", name)
			// funcs := htemplate.FuncMap(t.funcs)
			// tmpl, err := htemplate.New(name).Funcs(funcs).Parse(text)
			// if err != nil {
			// 	return nil, err
			// }
			// t.tmpl = tmpl
		} else {
			debug("New:", name)
			funcs := ttemplate.FuncMap(t.funcs)
			tt, err := ttemplate.New(name).Funcs(funcs).AddParseTree(name, tree)
			if err != nil {
				return nil, err
			}
			t.tmpl = tt
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
