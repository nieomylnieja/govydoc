# govydoc

`govydoc` combines a typed [govy][govy] validator with Go source comments
to produce JSON-serializable documentation.
The output is a flattened list of [JSONPath](https://www.rfc-editor.org/info/rfc9535/)
properties containing type details, validation rules, field documentation,
deprecation notices, and child-path relationships.

See the [govydoc package source][package-source]
for the complete public API.

## Install

Add the package to your project:

```sh
go get github.com/nieomylnieja/govydoc/pkg/govydoc
```

## Quick start

Assume the module path is `example.com/accounts`.
Define the documented type in `model/account.go`:

```go
package model

// Account identifies a user of the service.
type Account struct {
	// Name is the account's display name.
	Name string `json:"name"`
	// Email is the account's contact address.
	Email string `json:"email"`
}
```

From `main.go`, attach validation rules to the type.
Pass the validator to `govydoc.Generate`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nobl9/govy/pkg/govy"
	"github.com/nobl9/govy/pkg/rules"

	"github.com/nieomylnieja/govydoc/pkg/govydoc"

	"example.com/accounts/model"
)

func main() {
	validator := govy.New(
		govy.For(func(account model.Account) string { return account.Name }).
			WithName("name").
			Rules(rules.StringNotEmpty()),
		govy.For(func(account model.Account) string { return account.Email }).
			WithName("email").
			Rules(rules.StringEmail()),
	).
		WithName("Account")

	doc, err := govydoc.Generate(validator)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate documentation: %v\n", err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(doc); err != nil {
		fmt.Fprintf(os.Stderr, "encode documentation: %v\n", err)
		os.Exit(1)
	}
}
```

Run the program from its module:

```sh
go run .
```

The encoded `ObjectDoc` contains a root property at `$`
and one entry for every supported tagged field reachable from the type.

## Generated data

Each generated property combines information from reflection,
the Govy validation plan,
and source documentation:

- `Path` identifies the property with JSONPath notation.
- `TypeInfo` describes the Go type name, kind, and defining package.
- `Rules`, `Values`, and `Examples` come from the Govy property plan.
- `TypeDoc` contains the property's type documentation.
- `FieldDoc` contains the comment attached to the struct field.
- `DeprecatedDoc` contains text extracted from a `Deprecated:` marker.
- `ChildrenPaths` lists paths structurally associated with the property.

For slices, `ChildrenPaths` may contain both the field path
and its wildcard element path at the same ancestor level.

Go documentation links are rendered as links to [pkg.go.dev][pkg-go-dev].

### Property paths

`govydoc` maps common Go shapes to the following paths:

| Go shape      | Generated path   |
|:--------------|:-----------------|
| Root object   | `$`              |
| Struct field  | `$.name`         |
| Nested field  | `$.address.city` |
| Slice element | `$.items[*]`     |
| Map key       | `$.labels.*~`    |
| Map value     | `$.labels.*`     |

Only exported fields with an explicit JSON name are included.
Untagged fields, `json:"-"`, and tags without a name are ignored.

## Options

Options can be composed in the same `Generate` call:

```go
doc, err := govydoc.Generate(
	validator,
	govydoc.WithFilteredPaths("$.internal"),
	govydoc.GenerateGovyOptions(govy.PlanStrictMode()),
)
```

`WithFilteredPaths` removes `PropertyDoc` entries for exactly the listed paths.
It does not remove descendants or recompute `ChildrenPaths`,
which may still refer to filtered entries.

`GenerateGovyOptions` forwards options to the validation-plan generator.
See the available [Govy plan options][govy-plan-options].

## Development

Use the checked-in [Devbox][devbox] configuration
for the development toolchain.
Start the development shell:

```sh
devbox shell
```

Format the project, then run the CI verification commands:

```sh
just format
just test
just check
```

`just test` enables the race detector and collects package coverage.
`just check` runs vet, lint, spelling, whitespace, Markdown,
generated-code, and vulnerability checks.

[devbox]: https://www.jetify.com/devbox/
[govy]: https://github.com/nobl9/govy
[govy-plan-options]: https://pkg.go.dev/github.com/nobl9/govy/pkg/govy#PlanOption
[package-source]: https://github.com/nieomylnieja/govydoc/tree/main/pkg/govydoc
[pkg-go-dev]: https://pkg.go.dev/
