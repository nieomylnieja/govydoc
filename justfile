set shell := ["bash", "-c"]

bin_dir := "./bin"
scripts_dir := "./scripts"
app_name := "govydoc"
ldflags := "-s -w"

print_step := 'printf -- "------\n%s...\n"'

# Print this help message
[private]
default:
    @just --list

# Activate developer environment using devbox, run `just install-devbox` first if you don't have devbox installed
activate:
    devbox shell

# Install devbox binary
install-devbox:
    @{{ print_step }} "Installing devbox"
    curl -fsSL https://get.jetpack.io/devbox | bash

# Update devbox managed package versions
update-devbox:
    @{{ print_step }} "Update packages managed by devbox"
    devbox update

# Run all unit tests
test:
    @{{ print_step }} "Running unit tests"
    go test -race -cover ./...

# Run benchmark tests
test-benchmark:
    @{{ print_step }} "Running benchmark tests"
    go test -bench=. -benchmem ./...

# Produce test coverage report and inspect it in browser
test-coverage:
    @{{ print_step }} "Running test coverage report"
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out

# Run all checks
check: check-vet check-lint check-spell check-trailing check-markdown check-generate check-vulnerabilities

# Run 'go vet' on the whole project
check-vet:
    @{{ print_step }} "Running go vet"
    go vet ./...

# Run golangci-lint all-in-one linter with configuration defined inside .golangci.yml
check-lint:
    @{{ print_step }} "Running golangci-lint"
    golangci-lint run

# Check spelling, rules are defined in cspell.json
check-spell:
    @{{ print_step }} "Verifying spelling"
    cspell --no-progress '**/**'

# Check for trailing whitespaces in any of the projects' files
check-trailing:
    @{{ print_step }} "Looking for trailing whitespaces"
    {{ scripts_dir }}/check-trailing-whitespaces.bash

# Check markdown files for potential issues with markdownlint
check-markdown:
    @{{ print_step }} "Verifying Markdown files"
    markdownlint '**/*.md'

# Check for potential vulnerabilities across all Go dependencies
check-vulnerabilities:
    @{{ print_step }} "Running govulncheck"
    govulncheck ./...

# Verify if the auto generated code has been committed
check-generate:
    @{{ print_step }} "Checking if generated code matches the provided definitions"
    {{ scripts_dir }}/check-generate.bash

# Auto generate files
generate: generate-go

# Generate Golang code
generate-go:
    @{{ print_step }} "Generating Golang code"
    go generate ./...

# Format files
format: format-go format-just

# Format Go files
format-go:
    @{{ print_step }} "Formatting Go files"
    golangci-lint fmt

# Format justfile
format-just:
    @{{ print_step }} "Formatting justfile"
    just --fmt --unstable
