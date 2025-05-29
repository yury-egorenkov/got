# got

A Go template engine that combines environment variables, file inclusion, and smart indentation for generating configuration files and code templates.

## Features

- **Template Processing**: Uses Go's `text/template` with custom functions
- **Environment Variable Support**: Read and validate environment variables in templates
- **File Inclusion**: Include external files with automatic indentation
- **Smart Indentation**: Automatically indent included content to match template structure
- **Configuration Management**: Support for `.env.properties` files with flexible configuration paths

## Installation

Make sure you have Go installed, then run:

```bash
go install github.com/yury_egorenkov/got@latest
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
2. `.env.properties` in current directory

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

## Dependencies

- [github.com/mitranim/cmd](https://github.com/mitranim/cmd) - Command line argument parsing
- [github.com/mitranim/gg](https://github.com/mitranim/gg) - General utilities
- [github.com/joho/godotenv](https://github.com/joho/godotenv) - Environment file loading

## License

See project license file.