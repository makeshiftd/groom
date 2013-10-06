package main

import "io"
import "os"
import "fmt"
import "time"
import "flag"
import "sync"
import "errors"
import "strconv"
import "math/rand"
import "io/ioutil"
import "text/template"


func init() {
	rand.Seed(time.Now().UnixNano())
}


// randNumSuffix appends the specified number
// of random digits to the given base name.
// Useful for generating random template names.
func randNumSuffix(name string, length int) string {
	for i := 0; i<length; i++ {
		name += strconv.Itoa(rand.Intn(10))
	}
	return name
}


// Template is the interface providing extra methods over
// the standard template types, but allow allows the two
// standard tyopes (text/template and html/template) to
// be unified under this common interface.
type Template interface {
	AddData(name string, data interface{}) error
	AddFunc(name string, fucn interface{}) error
	AddTmpl(name string, srcs ...string) error
	Execute(w io.Writer) error
}


// InitDefaultData adds the default data to the given template.
func InitDefaultData(t Template) {
	// No default data. Reserved for future use.
}


// InitDefaultFuncs adds the default functions to the given template.
func InitDefaultFuncs(t Template) {
	// No default functions. Reserved for future use.
}


// TextTemplate wraps the standard (text/template) type
// and addes the required functionality to satisfy the
// Template interface.
type TextTemplate struct {
	tmpl *template.Template
	rwmx sync.RWMutex
	data map[string]interface{}
	targ string
}


// NewTextTemplate constructs a new TextTemplate type.
func NewTextTemplate() *TextTemplate {
	data := make(map[string]interface{})
	tmpl := template.New(randNumSuffix("TMPL", 10))
	t := &TextTemplate{ tmpl:tmpl, data:data }
	return t;
}


// AddData adds the given data to the template.
// The data will be used during template execution.
func (t *TextTemplate) AddData(name string, data interface{}) error {
	t.rwmx.Lock()
	defer t.rwmx.Unlock()
	t.data[name] = data
	return nil
}


// AddFunc add the given function to the template.
// These functions will be used during template execution.
func (t *TextTemplate) AddFunc(name string, data interface{}) (err error) {
	defer func() { 
		// Recover from panic caused by adding function to template
		// that does not have the correct number or type of return values.
	 	if recover() != nil {
	 		err = errors.New("invalid template function \"" + name + "\"")
	 	}
	}()
	t.rwmx.Lock()
	defer t.rwmx.Unlock()
	fm := map[string]interface{} { name:data }
	t.tmpl.Funcs(template.FuncMap(fm))
	return nil
}


// AddTmpl adds the given (sub)template to the template set.
func (t *TextTemplate) AddTmpl(name string, srcs ...string) error {
	t.rwmx.Lock()
	defer t.rwmx.Unlock()
	var tt *template.Template
	if name == "" {
		tt = t.tmpl
		name = tt.Name()
	} else {
		tt = t.tmpl.Lookup(name)
		if tt == nil {
			tt = t.tmpl.New(name)
		}
	}
	for _, src := range srcs {
		_, err := tt.Parse(src)
		if err != nil {
			return err
		}
	}
	if t.targ == "" {
		t.targ = name
	}
	return nil
}


// Execute executes the template and writes the result to the given Writer.
func (t *TextTemplate) Execute(w io.Writer) error {
	if t.targ == "" {
		return errors.New("no template definitions")
	}
	return t.tmpl.ExecuteTemplate(w, t.targ, t.data)
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

// ConfigValue type implements the flag.Value interface
// and provides a wrapper around a instance of a ConfigBuilder.
type ConfigValue struct {
	cfgs []Configurer
	cbld ConfigBuilder
}


// Set passes the given option string to the wrapped
// instance of ConfigBuilder, then adds the resulting
// Configurer to the list of Configurers.
func (v *ConfigValue) Set(option string) error {
	cfg, err := v.cbld.Build(option)
	if err != nil {
		return err
	}
	v.cfgs = append(v.cfgs, cfg)
	return nil
}


// String delegates to the DefValue function of the ConfigBuilder.
func (v *ConfigValue) String() string {
	return v.cbld.DefValue()
}


// ConfigRegistry type manages the "Groom" flagSet and
// delegates the command-line flags to Configurers.
type ConfigRegistry struct {
	cfgs []Configurer
	flgs *flag.FlagSet
	help bool
	html bool
	vrsn bool
	buff bool
	frce bool
	dest string
}


// NewConfigRegistry function constructs and initializes a
//  ConfigRegistry type, which includes the default FlagSet.
func NewConfigRegistry() (*ConfigRegistry) {
	c := &ConfigRegistry{ flgs:flag.NewFlagSet("groom", flag.ExitOnError) }
	c.flgs.BoolVar(&c.help, "help", false, "Print Groom help information")
	c.flgs.BoolVar(&c.html, "html", false, "Use HTML safe templates")
	c.flgs.BoolVar(&c.vrsn, "version", false, "Print version information")
	c.flgs.BoolVar(&c.buff, "buffer", true, "Buffer the result")
	c.flgs.BoolVar(&c.frce, "f", false, "Overwrite destination")
	c.flgs.StringVar(&c.dest, "o", "-", "Output destination")
	return c
}


// Register registers a mapping between a ConfigBuilder and a flag.
func (c *ConfigRegistry) Register(cbld ConfigBuilder, name string, usage string) {
	c.flgs.Var(&ConfigValue{ cbld:cbld, cfgs:c.cfgs }, name, usage)
}


// Process parses the command-line options and returns a slice of configurers.
func (c *ConfigRegistry) Process() ([]Configurer, error) {
	if !c.flgs.Parsed() {
		c.flgs.Parse(os.Args[1:])
		var argCfgs []Configurer
		for _, arg := range c.flgs.Args() {
			argCfgs = append(argCfgs, NewArgConfigurer(arg))
		}
		c.cfgs = append(argCfgs, c.cfgs...)
	}
	if c.help {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		c.flgs.PrintDefaults()
		os.Exit(1)
	}
	if c.vrsn {
		fmt.Fprintf(os.Stderr, "Version: %s\n", Version)
		os.Exit(1)
	}
	return c.cfgs, nil
}


// IsHtml reports if the user has request HTML safe templates with the "-html" flag.
func (c *ConfigRegistry) IsHtml() bool {
	return c.html
}


// IsHtml reports if the user has reqested result to buffered with the "-buffer" flag.
func (c *ConfigRegistry) IsBuffer() bool {
	return c.buff
}


// IsForce reports if the user has required to overwrite destination with the "-f" flag.
func (c *ConfigRegistry) IsForce() bool {
	return c.frce
}


// Dest reports the output destination requested by the user with the "-o" flag.
func (c *ConfigRegistry) Dest() string {
	return c.dest
}


// ArgConfigurer implements the Configurer interface and 
// configures a template based on the non-flag command-line
// arguments.
type ArgConfigurer struct {
	src chan string
	err chan error
}


// NewArgConfigurer makes a new ArgConfigurer initialized with
// the provided command-line argument.  The given argument
// is assumed to be the path of a template file. A goroutine is
// started to read the file asynchronously.
func NewArgConfigurer(arg string) *ArgConfigurer {
	c := &ArgConfigurer{ src:make(chan string), err:make(chan error) }
	go func() {
		src, err := ioutil.ReadFile(arg)
		if err != nil {
			c.err <- err
		} else {
			c.src <- string(src)
		}
	}()
	return c
}


// Configure receives either the content of the file
// or an error from the reading goroutine then adds
// the template.
func (c *ArgConfigurer) Configure(t Template) error {
	var src string
	select {
	case src = <-c.src:
		break
	case err := <-c.err:
		return err
	}
	err := t.AddTmpl("", src)
	if err != nil {
		return err
	}
	return nil
}
