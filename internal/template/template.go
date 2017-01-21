package main

import (
    "os"
    "io"
    //"fmt"
    "regexp"
    //"bytes"
    "errors"
    "strings"
    "io/ioutil"
    //"reflect"
    //"go/ast"
    //"go/parser"
    "path/filepath"
    ttemplate "text/template"
    htemplate "html/template"
)

var COMMENT_REGEXP = regexp.MustCompile("^\\s*{{\\s*/\\*\\s*(.*?)\\s*\\*/}}")

type Template struct {
    tmpl interface{}
    imps map[string]string
}


func new(name, path, text string, safe bool) (*Template, error) {
    if safe {
        tmpl, err := htemplate.New(name).Parse(text)
        if err != nil {
            return nil, err
        }
        return &Template{ tmpl:tmpl, imps:map[string]string{ name:path }}, nil
    }
    tmpl, err := htemplate.New(name).Parse(text)
    if err != nil {
        return nil, err
    }
    return &Template{ tmpl:tmpl, imps:map[string]string{ name:path }}, nil
}


func NewFromFile(name, path string, safe bool) (*Template, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    buf, rerr := ioutil.ReadAll(f)
    if rerr != nil {
        return nil, rerr
    }
    return NewFromText(name, path, string(buf), safe)
}


func NewFromText(name, path, text string, safe bool) (t *Template, err error) {
    path, err = filepath.Abs(path)
    if err != nil {
        return nil, err
    }
    imps, ierr := parseImports(text)
    if ierr != nil {
        return nil, ierr
    }
    t, err = new(name, path, text, safe)
    if err != nil {
        return nil, err
    }
    dir := filepath.Dir(path)
    for n, p := range(imps) {
        if !filepath.IsAbs(p) {
            p = filepath.Clean(filepath.Join(dir, p))
        }
        _, err := t.ParseFile(n, p)
        if err != nil {
            return nil, err
        }
    }
    return t, nil
}


func (t *Template) Lookup(name string) *Template {
    switch tmpl := t.tmpl.(type) {
    case *ttemplate.Template:
        tt := tmpl.Lookup(name)
        if tt == nil {
            return nil
        }
        return &Template{ tmpl:tt, imps:t.imps }
    case *htemplate.Template:
        tt := tmpl.Lookup(name)
        if tt == nil {
            return nil
        }
        return &Template{ tmpl:tt, imps:t.imps }
    default:
        panic("template type error")
    }
}


func (t *Template) Execute(wr io.Writer, data interface{}) error {
    switch tmpl := t.tmpl.(type) {
    case *ttemplate.Template:
        return tmpl.Execute(wr, data)
    case *htemplate.Template:
        return tmpl.Execute(wr, data)
    default:
        panic("template type error")
    }
}


func (t *Template) parse(name, path, text string) (*Template, error) {
    switch tmpl := t.tmpl.(type) {
    case *ttemplate.Template:
        tt, err := tmpl.New(name).Parse(text)
        if err != nil {
            return nil, err
        }
        return &Template{ tmpl:tt, imps:t.imps }, nil
    case *htemplate.Template:
        tt, err := tmpl.New(name).Parse(text)
        if tt == nil {
            return nil, err
        }
        return &Template{ tmpl:tt, imps:t.imps }, nil
    default:
        panic("template type error")
    }
}


func (t *Template) ParseFile(name, path string) (*Template, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    buf, rerr := ioutil.ReadAll(f)
    if rerr != nil {
        return nil, rerr
    }
    return t.ParseText(name, path, string(buf))
}


func (t *Template) ParseText(name, path, text string) (tt *Template, err error) {
    path, err = filepath.Abs(path)
    if err != nil {
        return nil, err
    }
    if p, ok := t.imps[name]; ok && p != path {
        return nil, errors.New("duplicate template: " + name)
    }
    if t.Lookup(name) != nil {
        return nil, errors.New("duplicate template: " + name)
    }
    imps, ierr := parseImports(text)
    if ierr != nil {
        return nil, ierr
    }
    tt, err = t.parse(name, path, text)
    if err != nil {
        return nil, err
    }
    dir := filepath.Dir(path)
    for n, p := range(imps) {
        if !filepath.IsAbs(p) {
            p = filepath.Clean(filepath.Join(dir, p))
        }
        _, err := t.ParseFile(n, p)
        if err != nil {
            return nil, err
        }
    }
    return tt, nil
}


var IMPORT_REGEXP = regexp.MustCompile("^\\+import\\s+(([^\\s]+)\\s+)?\"([^\"]+)\"\\s*$")

func parseImports(text string) (map[string]string, error) {
    imps := map[string]string{}
    for {
        comment := COMMENT_REGEXP.FindStringSubmatchIndex(text)
        if comment == nil {
            break;
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
    for ext := filepath.Ext(name); ext != "";  ext = filepath.Ext(name){
        name = name[:len(name)-len(ext)]
    }
    return name
}
