package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(originalWd)
	
	templateFile := filepath.Join(tmpDir, "template.tmpl")
	includeFile := filepath.Join(tmpDir, "include.txt")
	outputFile := "output.txt"
	
	includeContent := "service:\n  name: test\n  port: 8080"
	err = os.WriteFile(includeFile, []byte(includeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create include file: %v", err)
	}
	
	os.Setenv("TEST_ENV", "production")
	defer os.Unsetenv("TEST_ENV")
	
	templateContent := `config:
  env: {{ GetEnv "TEST_ENV" }}
  services:
    {{ ReadFileIndent "` + includeFile + `" }}
  
manual_indent:
{{ Indent 3 "line1\nline2" }}

simple_include:
{{ ReadFile "` + includeFile + `" }}`
	
	err = os.WriteFile(templateFile, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}
	
	cmd := CmdTempl{
		TemplFileName:  templateFile,
		OutputFileName: outputFile,
	}
	
	cmd.Run()
	
	result, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	
	resultStr := string(result)
	
	if !strings.Contains(resultStr, "env: production") {
		t.Error("Template should contain env variable value")
	}
	
	if !strings.Contains(resultStr, "    service:") {
		t.Error("Template should contain indented file content")
	}
	
	if !strings.Contains(resultStr, "line1\n      line2") {
		t.Errorf("Template should contain manually indented content, got: %q", resultStr)
	}
	
	if !strings.Contains(resultStr, "service:\n  name: test") {
		t.Error("Template should contain raw file content")
	}
}

func TestTemplateWithMultipleIndentCalls(t *testing.T) {
	tmpDir := t.TempDir()
	
	templateFile := filepath.Join(tmpDir, "template.tmpl")
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	
	err := os.WriteFile(file1, []byte("content1\nline2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	
	err = os.WriteFile(file2, []byte("content2\nline2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}
	
	templateContent := `root:
  level1:
    {{ ReadFileIndent "` + file1 + `" }}
  level2:
      {{ ReadFileIndent "` + file2 + `" }}`
	
	err = os.WriteFile(templateFile, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}
	
	cmd := CmdTempl{
		TemplFileName: templateFile,
	}
	
	body := string(must(os.ReadFile(templateFile)))
	cmd.Idents = cmd.IdentsByTemplateAST(body, "ReadFileIndent")
	
	if len(cmd.Idents) != 2 {
		t.Errorf("Expected 2 indentations, got %d", len(cmd.Idents))
	}
	
	if cmd.Idents[0] != "    " {
		t.Errorf("First indent should be '    ', got %q", cmd.Idents[0])
	}
	
	if cmd.Idents[1] != "      " {
		t.Errorf("Second indent should be '      ', got %q", cmd.Idents[1])
	}
}

func TestEnvFileLoading(t *testing.T) {
	tmpDir := t.TempDir()
	
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(originalWd)
	
	envContent := "TEST_FROM_FILE=loaded_value\nANOTHER_VAR=another_value"
	err = os.WriteFile(".env.properties", []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .env.properties: %v", err)
	}
	
	os.Unsetenv("TEST_FROM_FILE")
	defer os.Unsetenv("TEST_FROM_FILE")
	
	initEnvVars()
	
	value := os.Getenv("TEST_FROM_FILE")
	if value != "loaded_value" {
		t.Errorf("Expected env var to be 'loaded_value', got %q", value)
	}
}

func TestTemplateWithMissingEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	
	templateFile := filepath.Join(tmpDir, "template.tmpl")
	templateContent := `config:
  value: {{ GetEnv "NON_EXISTENT_VAR" }}`
	
	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}
	
	os.Unsetenv("NON_EXISTENT_VAR")
	
	cmd := CmdTempl{
		TemplFileName: templateFile,
	}
	
	defer func() {
		if r := recover(); r == nil {
			t.Error("Template execution should panic for missing env var")
		}
	}()
	
	cmd.Run()
}

func must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}