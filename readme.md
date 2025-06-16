# got

A Go template engine that combines environment variables, file inclusion, and smart indentation for generating configuration files and code templates.

## Features

- **Template Processing**: Uses Go's `text/template` with custom functions
- **Environment Variable Support**: Read and validate environment variables in templates
- **File Inclusion**: Include external files with automatic indentation
- **Smart Indentation**: Automatically indent included content to match template structure
- **Configuration Management**: Support for `.env.default.properties` and `.env.properties` files with flexible configuration paths
- **Recursive Template Processing**: All included files are processed as templates, allowing nested use of custom functions and variables
- **Cycle Dependency Detection**: Prevents infinite loops by detecting circular dependencies between template includes. The system tracks which files have been included in the current processing chain and throws an error if a file tries to include itself or create a circular reference

## Installation

Make sure you have Go installed, then run:

```bash
go install github.com/yury-egorenkov/got@latest
```

This will download the source and compile the executable into `$GOPATH/bin/got`. Make sure `$GOPATH/bin` is in your `$PATH` so the shell can discover the `got` command. For example, add this to your `~/.profile`:

```bash
export GOPATH="$HOME/go"
export PATH="$GOPATH/bin:$PATH"
```

Alternatively, you can run the executable using the full path:

```bash
~/go/bin/got
```

## Usage

### Basic Command

```bash
got run -t template.tmpl -o output.txt
```

### Options

- `-t`: Path to template file (required)
- `-o`: Output file path (optional, defaults to stdout)

### Template Functions

The following functions are available in templates:

#### `GetEnv`
Read environment variables with validation:
```go
{{ GetEnv "DATABASE_URL" }}
```
Panics if the environment variable is not set.

#### `ReadFile`
Include file contents:
```go
{{ ReadFile "config/database.yml" }}
```

#### `ReadFileIndent`
Include file contents with automatic indentation matching the template:
```go
services:
  app:
    {{ ReadFileIndent "docker/app-service.yml" }}
```

#### `Indent`
Manually indent text by specified spaces:
```go
{{ Indent 4 "some text" }}
```

## Configuration

### Environment Files

The tool automatically loads environment variables from `.env.properties` files in this order:

1. Files specified in `CONF` environment variable (comma-separated paths)
2. `.env.default.properties` in current directory for default values
3. `.env.properties` in current or specified by `CONF` directory to override defaults

Example:
```bash
# Load from custom config
CONF=conf/prod got run -t templ.tmpl

# Load from multiple configs
CONF=conf/base,conf/prod got run -t templ.tmpl
```

### Template Example

```go
# Generated configuration
database:
  url: {{ GetEnv "DATABASE_URL" }}
  
services:
  {{ ReadFileIndent "services.yml" }}

logging:
{{ Indent 2 (ReadFile "logging.conf") }}
```

### Error Handling

The tool includes built-in protection against common template issues:

- **Circular Dependencies**: If template A includes template B, and template B tries to include template A (directly or through a chain), the tool will detect this and throw an error like: `cyclic dependency file "template-b.tmpl" reads "template-a.tmpl"`
- **Missing Environment Variables**: Using `GetEnv` with undefined variables will cause the template processing to fail with a clear error message
- **Missing Files**: Attempting to include non-existent files will result in a file not found error

## Dependencies

- [github.com/mitranim/cmd](https://github.com/mitranim/cmd) - Command line argument parsing
- [github.com/mitranim/gg](https://github.com/mitranim/gg) - General utilities
- [github.com/joho/godotenv](https://github.com/joho/godotenv) - Environment file loading

## License

See project license file.