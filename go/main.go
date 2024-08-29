package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	r "reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/joho/godotenv"
	"github.com/mitranim/cmd"
	"github.com/mitranim/gg"
)

const (
	INDENT_MUL int = 2
)

var (
	// CLI commands. Mutated by `init` functions.
	commands = cmd.Map{
		`run`: CmdRunner[CmdTempl]{}.Run,
	}
)

type CmdRunner[_ gg.Runner] struct{}

func (CmdRunner[A]) Run() { gg.FlagParseTo[A](cmd.Args()).Run() }

func main() {
	defer gg.RecWith(fatal)
	initEnvVars()

	// flags
	// jsonFmt

	cmd := commands.Get()
	cmd()
}

func fatal(err error) {
	fmt.Println(`err:`, err)
	os.Exit(1)
}

type CmdTempl struct {
	TemplFileName string `flag:"-t" desc:"path of a template file"`
	Pads          []string
}

/*
TODO
	Doesn't work if ifs shadow.
*/

var templ *template.Template

func (self CmdTempl) Run() {
	funcs := template.FuncMap{
		`Indent`:         Indent,
		`ReadFile`:       ReadFile,
		`GetEnv`:         GetEnv,
		`ReadFileIndent`: func(name string) string { return self.ReadFileIndent(name) },
	}

	body := gg.ReadFile[string](self.TemplFileName)
	pads = PadsByFuncName(body, `ReadFileIndent`)

	templ = gg.Try1(template.New(`templ`).Funcs(funcs).Parse(body))
	gg.Try(templ.Execute(os.Stdout, nil))
}

var pads []string

func (self CmdTempl) ReadFileIndent(name string) (out string) {
	body := ReadFile(name)
	out = strings.Replace(body, gg.Newline, gg.Newline+pads[0], -1)
	pads = pads[1:]
	return
}

func PadsByFuncName(body string, name string) (out []string) {
	templFuncCallRegex := regexp.MustCompile(`\n(\s*){{\s*` + name)
	vals := templFuncCallRegex.FindAllStringSubmatch(body, -1)

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
