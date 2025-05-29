package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	r "reflect"
	"regexp"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/joho/godotenv"
	"github.com/mitranim/cmd"
	"github.com/mitranim/gg"
)

const (
	INDENT_MUL int = 2
)

type (
	Templ = template.Template
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
	fmt.Println(`Error:`, err)
	os.Exit(1)
}

type CmdTempl struct {
	TemplFileName  string `flag:"-t" desc:"Path of a template file."`
	OutputFileName string `flag:"-o" desc:"Path to output file. Stdout if unset."`
	Idents         []string
}

func (self CmdTempl) Templ() *Templ {
	funcs := template.FuncMap{
		`GetEnv`:         GetEnv,
		`Indent`:         Indent,
		`ReadFile`:       ReadFile,
		`ReadFileIndent`: func(name string) string { return self.ReadFileIndent(name) },
	}

	return template.New(`templ`).Funcs(funcs)
}

func (self CmdTempl) Run() {
	out := os.Stdout
	if gg.IsNotZero(self.OutputFileName) {
		outPathName := path.Join(gg.Cwd(), self.OutputFileName)
		file := gg.Try1(os.OpenFile(outPathName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644))
		defer file.Close()

		out = file
	}

	body := gg.ReadFile[string](self.TemplFileName)
	self.Idents = self.IdentsByTemplateAST(body, `ReadFileIndent`)

	templ := gg.Try1(self.Templ().Parse(body))
	gg.Try(templ.Execute(out, nil))
}

func (self *CmdTempl) ReadFileIndent(name string) (out string) {
	body := ReadFile(name)
	ident := gg.Newline + self.Idents[0]
	out = strings.Replace(body, gg.Newline, ident, -1)
	self.Idents = self.Idents[1:]
	return
}

func IdentsByFuncName(body string, name string) (out []string) {
	// Enhanced regex to capture indentation before template blocks or direct calls
	funCallRegex := regexp.MustCompile(`\n(\s*)(?:{{.*?}}[\s\n]*)*{{\s*` + name + `|(\s*)(?:{{.*?}}[\s\n]*)*{{\s*` + name)
	vals := funCallRegex.FindAllStringSubmatch(body, -1)

	for _, val := range vals {
		if len(val) > 1 {
			// Use first captured group if available, otherwise second
			if val[1] != "" {
				out = append(out, val[1])
			} else if len(val) > 2 && val[2] != "" {
				out = append(out, val[2])
			}
		}
	}

	return
}

// Parse template AST to find indentation context
func (self CmdTempl) IdentsByTemplateAST(body string, name string) (out []string) {
	templ := gg.Try1(self.Templ().Parse(body))
	lines := strings.Split(body, "\n")
	tree := templ.Tree
	if tree != nil && tree.Root != nil {
		walkNodes(tree.Root, lines, name, &out)
	}

	return
}

func walkNodes(node parse.Node, lines []string, funcName string, indents *[]string) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *parse.ActionNode:
		if containsFunction(n.Pipe, funcName) {
			// Find line number and extract indentation
			pos := int(n.Position())
			lineNum := findLineNumber(lines, pos)
			if lineNum >= 0 && lineNum < len(lines) {
				line := lines[lineNum]
				indent := extractIndentation(line)
				*indents = append(*indents, indent)
			}
		}
	case *parse.ListNode:
		if n != nil {
			for _, child := range n.Nodes {
				walkNodes(child, lines, funcName, indents)
			}
		}
	case *parse.IfNode:
		walkNodes(n.List, lines, funcName, indents)
		walkNodes(n.ElseList, lines, funcName, indents)
	case *parse.RangeNode:
		walkNodes(n.List, lines, funcName, indents)
		walkNodes(n.ElseList, lines, funcName, indents)
	case *parse.WithNode:
		walkNodes(n.List, lines, funcName, indents)
		walkNodes(n.ElseList, lines, funcName, indents)
	}
}

func containsFunction(pipe *parse.PipeNode, funcName string) bool {
	if pipe == nil {
		return false
	}
	for _, cmd := range pipe.Cmds {
		for _, arg := range cmd.Args {
			if ident, ok := arg.(*parse.IdentifierNode); ok && ident.Ident == funcName {
				return true
			}
		}
	}
	return false
}

func findLineNumber(lines []string, pos int) int {
	charCount := 0
	for i, line := range lines {
		charCount += len(line) + 1 // +1 for newline
		if charCount > pos {
			return i
		}
	}
	return -1
}

func extractIndentation(line string) string {
	for i, char := range line {
		if char != ' ' && char != '\t' {
			return line[:i]
		}
	}
	return line
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
