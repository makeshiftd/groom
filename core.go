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
	// AddFunc(name string, fucn  interface{}) error
	// AddTmpl(name string, srcs ...string) error
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

// Configurer is the interface that defines a type
// that is able to configure a template. Typically,
// these types handle a command-line option.
type Configurer interface {
	Configure(t Template) error
	DefValue() string
}

// ConfigBuilder is the interface that defines a type
// that can build a Configurer from the given option.
type ConfigBuilder interface {
	Build(option string) (Configurer, error)
}

type ConfigValue struct {
	cr []Configurer
	cb ConfigBuilder
}

func (v *ConfigValue) Set(option string) error {
	conf, err := v.cb.Build(option)
	if err != nil {
		return err
	}
	v.cr = append(v.cr, conf)
	return nil
}

func (v *ConfigValue) String() string {
	return v.DefValue()
}

// ConfigRegistry type manages the "Groom" flagSet and
// delegates the command-line flags to Configurers.
type ConfigRegistry struct {
	cfgs []Configurer
	flgs flag.FlagSet
}

// Register registers a mapping between a ConfigBuilder and a flag.
func (c *ConfigRegistry) Register(cb ConfigBuilder, name string, usage string) {
	c.flgs.Var(&ConfigValue{cb: cb, cr: c.cfgs}, name, usage)
}
