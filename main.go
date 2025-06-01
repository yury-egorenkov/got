package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	l "log"
	"os"
	"path"
	"path/filepath"
	r "reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/joho/godotenv"
	"github.com/mitranim/gg"
)

const (
	INDENT_MUL int = 2
)

type (
	Templ = template.Template
)

var (
	log = l.New(os.Stderr, `[got] `, 0)
	cwd = gg.Cwd()
)

func OptDefault() Opt { return gg.FlagParseTo[Opt](nil) }

type Opt struct {
	Args           []string `flag:""`
	Help           bool     `flag:"-h" desc:"Print help and exit."`
	OutputFileName string   `flag:"-o" desc:"Output path to file. Stdout if not set."`
}

func (self *Opt) Init(src []string) {
	err := gg.FlagParseCatch(src, self)
	if err != nil {
		self.LogErr(err)
		gg.Write(log.Writer(), gg.Newline)
		self.PrintHelp()
		os.Exit(1)
	}

	if self.Help || gg.Head(self.Args) == `help` {
		self.PrintHelp()
		os.Exit(0)
	}

	if len(self.Args) != 1 {
		self.PrintHelp()
		os.Exit(1)
	}
}

func (self Opt) PrintHelp() {
	gg.FlagFmtDefault.Prefix = "\t"
	gg.FlagFmtDefault.Head = false

	gg.Nop2(fmt.Fprintf(
		log.Writer(),
		TrimLines(`
"got" is the missing Go template processing cli tool.
Use any text file as Go tmplate, inline values from env, call funcs and more.

Usage:

	got <got_flags> <path_to_template>

Flags:
%v
`),
		gg.FlagHelp[Opt](),
	))
}

func (self Opt) LogErr(err error) {
	if err != nil {
		log.Printf(`%+v`, err)
	}
}

type Main struct {
	Opt
	Idents []string
}

func main() {
	defer gg.RecWith(fatal)
	initEnvVars()

	var main Main
	main.Opt.Init(os.Args[1:])
	main.Run()
}

func fatal(err error) {
	log.Printf(`%+v`, err)
	os.Exit(1)
}

func (self Main) Templ() *Templ {
	funcs := template.FuncMap{
		`GetEnv`:         GetEnv,
		`Indent`:         Indent,
		`ReadFile`:       ReadFile,
		`ReadFileIndent`: func(name string) string { return self.ReadFileIndent(name) },
	}

	return template.New(`templ`).Funcs(funcs)
}

func (self Main) Run() {
	var out io.Writer

	if gg.IsNotZero(self.OutputFileName) {
		outPathName := path.Join(cwd, self.OutputFileName)
		file := gg.Try1(os.OpenFile(outPathName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644))
		defer file.Close()
		out = file
	} else {
		out = os.Stdout
	}

	inpFileName := self.Opt.Args[0]
	body := gg.ReadFile[string](inpFileName)
	self.Idents = IdentsByFuncName(body, `ReadFileIndent`)

	templ := gg.Try1(self.Templ().Parse(body))
	gg.Try(templ.Execute(out, nil))
}

func (self *Main) ReadFileIndent(name string) (out string) {
	body := ReadFile(name)
	ident := gg.Newline + self.Idents[0]
	out = strings.Replace(body, gg.Newline, ident, -1)
	self.Idents = self.Idents[1:]
	return
}

/*
TODO

  - Panic if `ReadFileIndent` more than one in a line
  - To support conditional usage of `ReadFileIndent` use preprocessing with add
    indentation as argument.
*/
func IdentsByFuncName(body string, name string) (out []string) {
	funCallRegex := regexp.MustCompile(`\n(\s*){{\s*` + name)
	vals := funCallRegex.FindAllStringSubmatch(body, -1)

	for _, val := range vals {
		if len(val) > 0 {
			out = append(out, val[1])
		}
	}

	return
}

func Indent(spaces int, val string) string {
	pad := strings.Repeat(gg.Space, spaces*INDENT_MUL)
	return strings.Replace(val, gg.Newline, gg.Newline+pad, -1)
}

func ReadFile(name string) string {
	return gg.ReadFile[string](name)
}

func GetEnv(name string) (out string) {
	out = os.Getenv(name)

	if gg.IsTextEmpty(out) {
		panic(gg.Errf(`env var "%s" is not defined`, name))
	}

	return
}

/*
Using `CONF` allows to alter the location of `.env.properties`
when invoking the app. May provide multiple paths, comma-separated.

Example:

	CONF=conf/one          make
	CONF=conf/one,conf/two make
	CONF=conf/test         make test
*/
func initEnvVars() {
	for _, base := range gg.Reversed(CommaSplit(os.Getenv(`CONF`))) {
		parseEnvFileOpt(filepath.Join(base, `.env.properties`))
	}

	// TODO consider making this optional.
	parseEnvFileOpt(`.env.properties`)

	// parseEnvFile(`.env.default.properties`)
}

func parseEnvFile(path string) { gg.Try(godotenv.Load(path)) }

func parseEnvFileOpt(path string) {
	defer gg.SkipOnly(IsErrFileNotFound)
	parseEnvFile(path)
}

/*
Similar to `strings.Split` and `bytes.Split`. Differences:

  - Supports all text types.
  - Returns nil for empty input.

TODO consider moving to `gg`.
*/
func TextSplit[A gg.Text](src, sep A) []A {
	if len(src) <= 0 {
		return nil
	}

	if gg.Kind[A]() == r.String {
		return gg.CastSlice[A](strings.Split(gg.ToString(src), gg.ToString(sep)))
	}

	return gg.CastSlice[A](bytes.Split(gg.ToBytes(src), gg.ToBytes(sep)))
}

func CommaSplit[A gg.Text](src A) []A {
	return TextSplit(src, gg.ToText[A](`,`))
}

func IsErrFileNotFound(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

func TrimLines(src string) string {
	return strings.Trim(src, "\n")
}
