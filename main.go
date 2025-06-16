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
	Errf = gg.Errf
	Errv = gg.Errv
	log  = l.New(os.Stderr, `[got] `, 0)
	cwd  = gg.Cwd()
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

func (self Opt) TemplateFileName() string {
	return self.Args[0]
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

	var out io.Writer

	if gg.IsNotZero(main.Opt.OutputFileName) {
		outPathName := path.Join(cwd, main.Opt.OutputFileName)
		file := gg.Try1(os.OpenFile(outPathName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644))
		defer file.Close()
		out = file
	} else {
		out = os.Stdout
	}

	Render(main.Opt.TemplateFileName(), out, nil)
}

func fatal(err error) {
	log.Printf(`%+v`, err)
	os.Exit(1)
}

// TODO Check file include cycle dependency
func Render(pathname string, out io.Writer, used []string) {
	body := gg.ReadFile[string](ToAbsPath(pathname))

	funcs := template.FuncMap{
		`GetEnv`:         GetEnv,
		`Indent`:         Indent,
		`ReadFile`:       ReadFile,
		`ReadFileIndent`: Render_ReadFileIndent(pathname, used),
	}

	body = ReadFileIndent(body).Validate().Process()
	templ := gg.Try1(template.New(`templ`).Funcs(funcs).Parse(body))
	gg.Try(templ.Execute(out, nil))
}

type ReadFileIndentFunc func(indent string, name string) string

func Render_ReadFileIndent(ancestor string, used []string) ReadFileIndentFunc {
	return func(indent string, pathname string) string {
		if gg.Has(used, pathname) {
			panic(Errf(`cyclic dependency file %q reads %q`, pathname, ancestor))
		}

		used = append(used, pathname)

		var buf gg.Buf
		Render(pathname, &buf, used)
		body := indent + buf.String()
		return strings.Replace(body, gg.Newline, gg.Newline+indent, -1)
	}
}

/*
TODO

  - Panic if `ReadFileIndent` more than one in a line
  - To support conditional usage of `ReadFileIndent` use preprocessing with add
    indentation as argument.
*/

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
	parseEnvFileOpt(`.env.default.properties`)
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

var (
	ReadFileIndentValid   = regexp.MustCompile(`(?m){{\s*ReadFileIndent.*{{\s*ReadFileIndent|^\s*ReadFileIndent`)
	ReadFileIndentPattern = regexp.MustCompile(`(?m)^(?P<indent>.*)(?P<func>{{\s*ReadFileIndent)\b\s*(?P<rest>.*)$`)
)

type ReadFileIndent string

func (self ReadFileIndent) Invalid() bool {
	return ReadFileIndentValid.MatchString(string(self))
}

func (self ReadFileIndent) Validate() ReadFileIndent {
	if self.Invalid() {
		panic(Errv(`Invalid template: Multiple "ReadFileIndent" calls found on a single line.`))
	}

	return self
}

/* Returns indentation inplaced. */
func (self ReadFileIndent) Process() string {
	return ReadFileIndentPattern.ReplaceAllString(string(self), "${func} `${indent}` ${rest}")
}

func ToAbsPath(path string) string {
	if strings.HasPrefix(path, `/`) || strings.Contains(path, `:\\`) {
		return path
	}

	if strings.HasPrefix(path, `~/`) || path == "~" {
		return path
	}

	return filepath.Join(gg.Cwd(), path)
}
