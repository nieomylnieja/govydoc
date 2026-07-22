# go-repo-template

Repository [template](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-repository-from-a-template)
for creating new Go projects!

## Bootstrap

Click `Use this template` button and voila!
![2024-10-05_22-53](https://github.com/user-attachments/assets/ae397fc7-5fa5-49df-94c1-314572a223d8)

After you're done, run the interactive bootstrap CLI:

```shell
just bootstrap
```

The CLI will guide you through an interactive form to configure your new project:

- **GitHub Account Name**: The GitHub account or organization that owns this repository
- **Repository Name**: The name of your new repository
- **Include Binary Support?**:
  Whether to include GoReleaser configuration and binary build workflows
- **Include Versioning Support?**:
  Whether to include release drafter, automated versioning workflows,
  and release-note checks for `feat:` and `fix:` pull requests
- **Set GitHub Release Secrets?**:
  Optionally store supplied personal access tokens as the required GitHub
  Actions release secrets with the GitHub CLI.

## Devbox

This project utilizes [devbox](https://github.com/jetify-com/devbox) in order
to provide a consistent and reliable development environment.
You can however, If you choose so, install the required dependencies manually.

## Project structure

The template includes an example of
[recommended Go project layout](https://github.com/golang-standards/project-layout)
which includes `cmd`, `pkg` and `internal` directories.

## justfile

[justfile](https://github.com/casey/just) provides all the basic utilities
for the development workflow.
Feel free to extend it with additional recipes as you see fit.
The same justfile recipes are used in CI, this ensures consistent results
for both remote and local machines.

You can quickly inspect the recipes of justfile by running either:

```shell
just --list
# or simply
just
```

When writing new recipes, make sure you document them with a `#` comment
directly above the recipe, like so:

```just
# Document me!
new-recipe:
  echo "Hello"
```

## CI

Continuous integration pipelines utilize the same
[justfile](./justfile) recipes which
you run locally within reproducible `devbox` environment.
This ensures consistent behavior of the executed checks
and makes local debugging easier.

## Testing

You can run all unit tests with `just test`.
We also encourage inspecting test coverage during development, you can verify
if the paths you're interested in are covered with `just test-coverage`.

## Releasing binaries

If you decide to ship binaries with the project,
[GoReleaser](https://goreleaser.com/) will require setting up
`GORELEASER_TOKEN` secret.
Use a dedicated fine-grained personal access token scoped to the generated
repository with `Contents: read and write`.
`Metadata: read-only` is included automatically.

## Release Drafter

If you decide to keep release automation, you will need to set up
`RELEASE_DRAFTER_TOKEN` secret.
Use a dedicated fine-grained personal access token scoped to the generated
repository with `Contents: read and write` and
`Pull requests: read and write`.
`Metadata: read-only` is included automatically.

## GitHub release secrets

The bootstrap CLI does not create personal access tokens.
For the least-privilege setup, create separate tokens for GoReleaser and
Release Drafter with the permissions listed above.
When secret setup is enabled, the CLI stores:

- `GORELEASER_TOKEN` from the GoReleaser token,
  when binary support is enabled.
- `RELEASE_DRAFTER_TOKEN` from the Release Drafter token,
  when versioning support is enabled.

You can also store the secrets manually with
[GitHub CLI](https://cli.github.com/):

```shell
gh secret set GORELEASER_TOKEN --repo <github-account-name>/<repo-name>
gh secret set RELEASE_DRAFTER_TOKEN --repo <github-account-name>/<repo-name>
```

## Labels

In order for some automations to work, like
[Release Drafter](https://github.com/release-drafter/release-drafter),
we need a predefined set of labels.
If you create a new repository from this template, the labels will be
automatically transferred for you.
However, if you want to use these automations in an existing repository,
you'll need to create these labels.
There's a convenience script for that,
located [here](./bootstrap/add-labels.bash).
Run the following:

```shell
./bootstrap/add-labels.bash <github-account-name> <repo-name>
```

If you wish to update existing labels, add `--force` to the `gh label create`
invocation in the script.

## Renovate

This template includes [Renovate](https://docs.renovatebot.com/) configuration
for automated dependency updates.
Renovate will automatically create pull requests to update your dependencies
when new versions are available.

The configuration is located in [.github/renovate.json5](./.github/renovate.json5).

To enable Renovate for your repository:

1. Install the [Renovate GitHub App](https://github.com/apps/renovate).
2. Grant it access to your repository.
3. Renovate will automatically start monitoring your dependencies.

## Gitsync

The author of this repository also uses it as a staple/root
for other repositories to follow.
This means things like linter configs or CI/CD workflows in these repositories
are supposed to be kept in sync with **this** repository (with some variations).

This is achieved with a tool called [gitsync](https://github.com/nieomylnieja/gitsync).
Configuration file for the tool is [gitsync.json](./gitsync.json).
The bootstrap script removes it.

In order to see the diff between managed repositories run:

```shell
gitsync -c gitsync.json diff
```

In order to sync the changes for managed repositories run:

```shell
gitsync -c gitsync.json sync
```

## License

The repository template comes with Mozilla Public License 2.0.
Feel free to change the license to any that suits you.
