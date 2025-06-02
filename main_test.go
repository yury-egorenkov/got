package main

import (
	"os"
	"testing"

	"github.com/mitranim/gg"
	"github.com/mitranim/gg/gtest"
)

func TestMain(m *testing.M) {
	gg.TraceBaseDir = gg.Cwd()
	gg.TraceShortName = false

	os.Exit(m.Run())
}

func Test_Pathname(t *testing.T) {
	defer gtest.Catch(t)
	gtest.Eq(ToAbsPath(`./file.txt`), `/Users/yury/Projects/me/got/file.txt`)
	gtest.Eq(ToAbsPath(`../parent/file.txt`), `/Users/yury/Projects/me/parent/file.txt`)
	gtest.Eq(ToAbsPath(`relative/path/to/file.txt`), `/Users/yury/Projects/me/got/relative/path/to/file.txt`)
	gtest.Eq(ToAbsPath(`/absolute/path/file.txt`), `/absolute/path/file.txt`)
	gtest.Eq(ToAbsPath(`~/path/file.txt`), `~/path/file.txt`)
	gtest.Eq(ToAbsPath(`C:\\Windows\\System32`), `C:\\Windows\\System32`)
}

func Test_ReadFileIndent(t *testing.T) {
	defer gtest.Catch(t)

	t.Run(`validate_one_line_call`, func(t *testing.T) {
		src := ReadFileIndent(`{{ ReadFileIndent }} {{ ReadFileIndent }}`)

		src1 := ReadFileIndent(`
		{{ ReadFileIndent }} {{
			ReadFileIndent }}
	`)

		gtest.True(src.Invalid())
		gtest.True(src1.Invalid())
	})

	t.Run(`single_line_indent`, func(t *testing.T) {
		src := ReadFileIndent("    	{{ReadFileIndent `fixtures/main.cf`}}")
		expected := "{{ReadFileIndent `    	` `fixtures/main.cf`}}"

		gtest.Eq(expected, src.Process())
	})

	t.Run(`single_line_indent`, func(t *testing.T) {
		src := `
    	{{ReadFileIndent ` + "`fixtures/main.cf`}}" + `
{{ReadFileIndent ` + "`fixtures/main.cf`}}" + `
	{{ReadFileIndent ` + "`fixtures/main.cf`}}"

		expected := `
{{ReadFileIndent ` + "`    	` `fixtures/main.cf`}}" + `
{{ReadFileIndent ` + "`` `fixtures/main.cf`}}" + `
{{ReadFileIndent ` + "`	` `fixtures/main.cf`}}"

		actual := ReadFileIndentPattern.ReplaceAllString(src, "${func} `${indent}` ${rest}")

		gtest.Eq(expected, actual)
	})
}
