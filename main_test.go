package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIndent(t *testing.T) {
	tests := []struct {
		name     string
		spaces   int
		input    string
		expected string
	}{
		{
			name:     "single line",
			spaces:   2,
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "multiple lines",
			spaces:   2,
			input:    "line1\nline2\nline3",
			expected: "line1\n    line2\n    line3",
		},
		{
			name:     "zero spaces",
			spaces:   0,
			input:    "line1\nline2",
			expected: "line1\nline2",
		},
		{
			name:     "empty string",
			spaces:   2,
			input:    "",
			expected: "",
		},
		{
			name:     "only newline",
			spaces:   1,
			input:    "\n",
			expected: "\n  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Indent(tt.spaces, tt.input)
			if result != tt.expected {
				t.Errorf("Indent(%d, %q) = %q, want %q", tt.spaces, tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name        string
		envVar      string
		envValue    string
		expectPanic bool
	}{
		{
			name:     "existing env var",
			envVar:   "TEST_VAR_EXISTS",
			envValue: "test_value",
		},
		{
			name:        "non-existing env var",
			envVar:      "TEST_VAR_NOT_EXISTS",
			expectPanic: true,
		},
		{
			name:        "empty env var",
			envVar:      "TEST_VAR_EMPTY",
			envValue:    "",
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			} else if !tt.expectPanic {
				os.Unsetenv(tt.envVar)
			}

			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("GetEnv(%q) should have panicked", tt.envVar)
					}
				}()
			}

			result := GetEnv(tt.envVar)
			if !tt.expectPanic && result != tt.envValue {
				t.Errorf("GetEnv(%q) = %q, want %q", tt.envVar, result, tt.envValue)
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "hello\nworld\ntest"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := ReadFile(testFile)
	if result != testContent {
		t.Errorf("ReadFile(%q) = %q, want %q", testFile, result, testContent)
	}
}

func TestReadFileNotExists(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("ReadFile should panic for non-existent file")
		}
	}()

	ReadFile("/non/existent/file.txt")
}

func TestIdentsByFuncName(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		funcName string
		expected []string
	}{
		{
			name:     "single function call with indentation",
			body:     "line1\n    {{ ReadFileIndent \"file.txt\" }}\nline3",
			funcName: "ReadFileIndent",
			expected: []string{"    "},
		},
		{
			name:     "multiple function calls",
			body:     "line1\n  {{ ReadFileIndent \"file1.txt\" }}\nline3\n      {{ ReadFileIndent \"file2.txt\" }}",
			funcName: "ReadFileIndent",
			expected: []string{"  ", "      "},
		},
		{
			name:     "no function calls",
			body:     "line1\nline2\nline3",
			funcName: "ReadFileIndent",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IdentsByFuncName(tt.body, tt.funcName)
			if len(result) != len(tt.expected) {
				t.Errorf("IdentsByFuncName() returned %d results, want %d", len(result), len(tt.expected))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("IdentsByFuncName()[%d] = %q, want %q", i, result[i], expected)
				}
			}
		})
	}
}

func TestExtractIndentation(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "spaces only",
			line:     "    hello",
			expected: "    ",
		},
		{
			name:     "tabs only",
			line:     "\t\thello",
			expected: "\t\t",
		},
		{
			name:     "mixed spaces and tabs",
			line:     "  \t hello",
			expected: "  \t ",
		},
		{
			name:     "no indentation",
			line:     "hello",
			expected: "",
		},
		{
			name:     "only whitespace",
			line:     "    ",
			expected: "    ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIndentation(tt.line)
			if result != tt.expected {
				t.Errorf("extractIndentation(%q) = %q, want %q", tt.line, result, tt.expected)
			}
		})
	}
}

func TestCommaSplit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single item",
			input:    "item1",
			expected: []string{"item1"},
		},
		{
			name:     "multiple items",
			input:    "item1,item2,item3",
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "spaces around commas",
			input:    "item1, item2 , item3",
			expected: []string{"item1", " item2 ", " item3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CommaSplit(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("CommaSplit(%q) returned %d items, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("CommaSplit(%q)[%d] = %q, want %q", tt.input, i, result[i], expected)
				}
			}
		})
	}
}

func TestCmdTemplReadFileIndent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := CmdTempl{
		Idents: []string{"  ", "    "},
	}

	result := cmd.ReadFileIndent(testFile)
	expected := "line1\n  line2\n  line3"

	if result != expected {
		t.Errorf("ReadFileIndent() = %q, want %q", result, expected)
	}

	if len(cmd.Idents) != 1 {
		t.Errorf("ReadFileIndent should consume one indent, got %d remaining", len(cmd.Idents))
	}
}

func TestFindLineNumber(t *testing.T) {
	lines := []string{"line0", "line1", "line2"}
	tests := []struct {
		name     string
		pos      int
		expected int
	}{
		{
			name:     "first line",
			pos:      0,
			expected: 0,
		},
		{
			name:     "second line start",
			pos:      6,
			expected: 1,
		},
		{
			name:     "third line",
			pos:      12,
			expected: 2,
		},
		{
			name:     "beyond end",
			pos:      100,
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findLineNumber(lines, tt.pos)
			if result != tt.expected {
				t.Errorf("findLineNumber(lines, %d) = %d, want %d", tt.pos, result, tt.expected)
			}
		})
	}
}

func TestIsErrFileNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "file not found error",
			err:      os.ErrNotExist,
			expected: true,
		},
		{
			name:     "other error",
			err:      os.ErrPermission,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsErrFileNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("IsErrFileNotFound(%v) = %t, want %t", tt.err, result, tt.expected)
			}
		})
	}
}