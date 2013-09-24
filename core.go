package main

import "flag"
import "sync"
import "text/template"

// Template is the interface providing extra methods over
// the standard template types, but allow allows the two
// standard tyopes (text/template and html/template) to
// be unified under this common interface.
type Template interface {
	AddData(name string, data interface{}) error
	// AddFunc(name string, fucn interface{}) error
	AddTmpl(name string, srcs ...string) error
	// LkupData(name string) interface{}
	// LkupFunc(name string) interface{}
	// LkupTmpl(name string)
	// ExecTmpl(w io.Writer, name string) error
	// Execute(w io.writer) error
}

// TextTemplate wraps the standard (text/template) type
// and addes the required functionality to satisfy the
// Template interface.
type TextTemplate struct {
	tmpl *template.Template
	rwmx sync.RWMutex
	data map[string]interface{}
}

// NewTextTemplate constructs a new TextTemplate type.
func NewTextTemplate() *TextTemplate {
	return &TextTemplate{data: make(map[string]interface{})}
}

// AddData adds the given data to the template.
// The data will be used during template execution.
func (t *TextTemplate) AddData(name string, data interface{}) error {
	t.rwmx.Lock()
	t.data[name] = data
	t.rwmx.Unlock()
	return nil
}


// AddTmpl adds the given (sub)template to the template set.
func (t *TextTemplate) AddTmpl(name string, srcs ...string) error {
	var tt *template.Template
	if t.tmpl == nil {
		tt = template.New(name)
		t.tmpl = tt
	} else {
		tt = t.tmpl.New(name)
	}
	for _, src := range srcs {
		_, err := tt.Parse(src)
		if err != nil {
			return err
		}
	}
	return nil
}


// Configurer is the interface that defines a type
// that is able to configure a template. Typically,
// these types handle a command-line option.
type Configurer interface {
	Configure(t Template) error
}

// ConfigBuilder is the interface that defines a type
// that can build a Configurer from the given option.
type ConfigBuilder interface {
	Build(option string) (Configurer, error)
	DefValue() string
}

type ConfigValue struct {
	cfgs []Configurer
	cbld ConfigBuilder
}

func (v *ConfigValue) Set(option string) error {
	cfg, err := v.cbld.Build(option)
	if err != nil {
		return err
	}
	v.cfgs = append(v.cfgs, cfg)
	return nil
}

func (v *ConfigValue) String() string {
	return v.cbld.DefValue()
}

// ConfigRegistry type manages the "Groom" flagSet and
// delegates the command-line flags to Configurers.
type ConfigRegistry struct {
	cfgs []Configurer
	flgs flag.FlagSet
}

// Register registers a mapping between a ConfigBuilder and a flag.
func (c *ConfigRegistry) Register(cbld ConfigBuilder, name string, usage string) {
	c.flgs.Var(&ConfigValue{ cbld:cbld, cfgs:c.cfgs }, name, usage)
}
