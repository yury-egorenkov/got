package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	r "reflect"
	"strings"

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

	cmd := commands.Get()
	cmd()
}

func fatal(err error) {
	fmt.Println(`err:`, err)
	os.Exit(1)
}

type CmdTempl struct {
	TemplFileName string `flag:"-t" desc:"path of a template file"`
}

func (self CmdTempl) Run() {
	funcs := template.FuncMap{
		`indent`:   Indent,
		`readFile`: ReadFile,
		`getEnv`:   GetEnv,
	}

	body := gg.ReadFile[string](self.TemplFileName)
	templ := gg.Try1(template.New(`templ`).Funcs(funcs).Parse(body))
	gg.Try(templ.Execute(os.Stdout, nil))
}

func Indent(spaces int, val string) string {
	pad := strings.Repeat(gg.Space, spaces*INDENT_MUL)
	return strings.Replace(val, gg.Newline, gg.Newline+pad, -1)
}

func ReadFile(name string) string { return gg.ReadFile[string](name) }

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

	parseEnvFile(`.env.default.properties`)
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
