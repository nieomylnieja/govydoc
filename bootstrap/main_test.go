package main

import (
	_ "embed"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/survivorbat/huhtest"
)

//go:embed testdata/expected-justfile-with-binary
var expectedJustfileWithBinary string

//go:embed testdata/expected-justfile-no-binary
var expectedJustfileNoBinary string

const (
	testAccountName = "test-account"
	testRepoName    = "test-repo"
)

func TestBootstrap_DefaultBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, testAccountName, testRepoName)
	require.NoError(t, err, "Bootstrap failed. Output: %s", output)

	t.Run("removes bootstrap directory", func(t *testing.T) {
		assert.NoDirExists(t, filepath.Join(tmpDir, "bootstrap"))
	})

	t.Run("removes test directory", func(t *testing.T) {
		assert.NoDirExists(t, filepath.Join(tmpDir, "test"))
	})
	t.Run("renames cmd directory", func(t *testing.T) {
		oldPath := filepath.Join(tmpDir, "cmd", "x-repo-name")
		newPath := filepath.Join(tmpDir, "cmd", testRepoName)

		assert.NoDirExists(t, oldPath)
		assert.DirExists(t, newPath)
	})
	t.Run("replaces repo and account names in files", func(t *testing.T) {
		goMod := readFile(t, filepath.Join(tmpDir, "go.mod"))
		assert.NotContains(t, goMod, "x-github-account-name")
		assert.NotContains(t, goMod, "x-repo-name")
		assert.Contains(t, goMod, testAccountName)
		assert.Contains(t, goMod, testRepoName)

		justfile := readFile(t, filepath.Join(tmpDir, "justfile"))
		assert.NotContains(t, justfile, "x-repo-name")

		golangciYml := readFile(t, filepath.Join(tmpDir, ".golangci.yml"))
		assert.NotContains(t, golangciYml, "x-github-account-name")
		assert.NotContains(t, golangciYml, "x-repo-name")
		assert.Contains(t, golangciYml, testAccountName)
		assert.Contains(t, golangciYml, testRepoName)

		featureRequest := readFile(t, filepath.Join(tmpDir, ".github", "ISSUE_TEMPLATE", "feature_request.md"))
		assert.NotContains(t, featureRequest, "x-repo-name")
		assert.Contains(t, featureRequest, testRepoName)
	})
	t.Run("creates new README.md", func(t *testing.T) {
		readme := readFile(t, filepath.Join(tmpDir, "README.md"))
		expectedContent := "# " + testRepoName + "\n\nTODO\n"
		assert.Equal(t, expectedContent, readme, "README.md content incorrect")
	})
	t.Run("keeps binary-related files", func(t *testing.T) {
		assert.FileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release.yml"))
	})
	t.Run("keeps versioning-related files", func(t *testing.T) {
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "scripts", "release-notes.bash"))
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "workflows", "pr-autolabeler.yml"))
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "workflows", "pr-check.yml"))
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release-drafter.yml"))

		prCheck := readFile(t, filepath.Join(tmpDir, ".github", "workflows", "pr-check.yml"))
		assert.Contains(t, prCheck, "  pr-title-check:")
		assert.Contains(t, prCheck, "name: Check PR title")
		assert.Contains(t, prCheck, "readonly PR_TITLE_PATTERN='^(feat|fix|sec|infra|test|chore|doc): .{5,}$'")
		assert.NotContains(t, prCheck, "Slashgear/action-check-pr-title")
		assert.Contains(t, prCheck, "  release-notes-check:")
	})

	t.Run("keeps build, install, and release recipes in justfile", func(t *testing.T) {
		actualJustfile := readFile(t, filepath.Join(tmpDir, "justfile"))
		expectedJustfile := getExpectedJustfile(t, true)
		assert.Equal(t, expectedJustfile, actualJustfile, "justfile content differs from expected")
	})
}

func TestBootstrap_NoBinaryFlag(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, "--no-binary", testAccountName, testRepoName)
	require.NoError(t, err, "Bootstrap failed. Output: %s", output)

	t.Run("removes binary-related files", func(t *testing.T) {
		assert.NoFileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release.yml"))
	})
	t.Run("removes build, install, and release recipes from justfile", func(t *testing.T) {
		actualJustfile := readFile(t, filepath.Join(tmpDir, "justfile"))
		expectedJustfile := getExpectedJustfile(t, false)
		assert.Equal(t, expectedJustfile, actualJustfile, "justfile content differs from expected")
	})
	t.Run("does not rename cmd directory", func(t *testing.T) {
		oldPath := filepath.Join(tmpDir, "cmd", "x-repo-name")
		newPath := filepath.Join(tmpDir, "cmd", testRepoName)

		assert.NoDirExists(t, newPath)
		assert.DirExists(t, oldPath)
	})
	t.Run("still replaces placeholders", func(t *testing.T) {
		goMod := readFile(t, filepath.Join(tmpDir, "go.mod"))
		assert.NotContains(t, goMod, "x-github-account-name")
		assert.NotContains(t, goMod, "x-repo-name")
	})
	t.Run("output shows binary support disabled", func(t *testing.T) {
		assert.Contains(t, output, "Binary support: false")
	})
}

func TestBootstrap_NoVersioningFlag(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, "--no-versioning", testAccountName, testRepoName)
	require.NoError(t, err, "Bootstrap failed. Output: %s", output)

	t.Run("removes versioning-related files", func(t *testing.T) {
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "scripts", "release-notes.bash"))
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "workflows", "pr-autolabeler.yml"))
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release-drafter.yml"))
	})
	t.Run("keeps pull request title check only", func(t *testing.T) {
		prCheck := readFile(t, filepath.Join(tmpDir, ".github", "workflows", "pr-check.yml"))

		assert.Contains(t, prCheck, "  pr-title-check:")
		assert.Contains(t, prCheck, "name: Check PR title")
		assert.Contains(t, prCheck, "readonly PR_TITLE_PATTERN='^(feat|fix|sec|infra|test|chore|doc): .{5,}$'")
		assert.NotContains(t, prCheck, "Slashgear/action-check-pr-title")
		assert.NotContains(t, prCheck, "  release-notes-check:")
	})
	t.Run("keeps binary-related files", func(t *testing.T) {
		assert.FileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release.yml"))
	})
	t.Run("output shows versioning support disabled", func(t *testing.T) {
		assert.Contains(t, output, "Versioning support: false")
	})
}

func TestBootstrap_BothFlags(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, "--no-binary", "--no-versioning", testAccountName, testRepoName)
	require.NoError(t, err, "Bootstrap failed. Output: %s", output)

	t.Run("removes binary-related files", func(t *testing.T) {
		assert.NoFileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
	})
	t.Run("removes versioning-related files", func(t *testing.T) {
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
	})
	t.Run("output indicates both options were processed", func(t *testing.T) {
		assert.Contains(t, output, "Binary support: false")
		assert.Contains(t, output, "Versioning support: false")
	})
}

func TestBootstrap_FlagsInDifferentOrder(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	_, err := runBootstrap(t, tmpDir, "--no-versioning", "--no-binary", testAccountName, testRepoName)
	require.NoError(t, err, "Bootstrap failed with flags in different order")

	assert.NoFileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
	assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
}

func TestBootstrap_MissingArguments(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	tests := []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "no arguments",
			args:        []string{},
			expectedErr: "interactive mode requires a TTY",
		},
		{
			name:        "only account name",
			args:        []string{testAccountName},
			expectedErr: "BOOTSTRAP_REPO environment variable is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			copyProject(t, tmpDir)

			output, err := runBootstrap(t, tmpDir, tt.args...)
			assert.Error(t, err, "Expected bootstrap to fail with missing arguments")
			assert.Contains(t, output, tt.expectedErr)
		})
	}
}

func TestBootstrap_FlagAfterPositionalArgs(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	_, err := runBootstrap(t, tmpDir, testAccountName, testRepoName, "--no-binary")
	require.NoError(t, err, "Bootstrap should handle flags after positional args")
	assert.NoFileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
}

func TestLoadConfigInteractive_BothEnabled(t *testing.T) {
	cfg, err := runLoadConfigInteractive(t, true, true)
	require.NoError(t, err)
	assert.Equal(t, "my-account", cfg.accountName)
	assert.Equal(t, "my-repo", cfg.repoName)
	assert.True(t, cfg.includeBinary)
	assert.True(t, cfg.includeVersion)
	assert.False(t, cfg.setupSecrets)
}

func TestLoadConfigInteractive_BinaryDisabled(t *testing.T) {
	cfg, err := runLoadConfigInteractive(t, false, true)
	require.NoError(t, err)
	assert.False(t, cfg.includeBinary)
	assert.True(t, cfg.includeVersion)
}

func TestLoadConfigInteractive_VersioningDisabled(t *testing.T) {
	cfg, err := runLoadConfigInteractive(t, true, false)
	require.NoError(t, err)
	assert.True(t, cfg.includeBinary)
	assert.False(t, cfg.includeVersion)
}

func TestLoadConfigInteractive_BothDisabled(t *testing.T) {
	cfg, err := runLoadConfigInteractive(t, false, false)
	require.NoError(t, err)
	assert.False(t, cfg.includeBinary)
	assert.False(t, cfg.includeVersion)
}

func TestLoadConfigInteractive_WhitespaceTrimmed(t *testing.T) {
	disableRepositoryDetection(t)

	stdin, stdout, cancel := huhtest.NewResponder().
		AddResponse("GitHub Account Name", "  my-account  ").
		AddResponse("Repository Name", "  my-repo  ").
		AddConfirm("Include Binary Support?", huhtest.ConfirmAffirm).
		AddConfirm("Include Versioning Support?", huhtest.ConfirmAffirm).
		AddConfirm("Set GitHub Release Secrets?", huhtest.ConfirmNegative).
		Start(t, 30*time.Second)
	defer cancel()

	cfg, err := loadConfigInteractive(stdin, stdout)
	require.NoError(t, err)
	assert.Equal(t, "my-account", cfg.accountName)
	assert.Equal(t, "my-repo", cfg.repoName)
}

func TestLoadConfigInteractive_SetupSecrets(t *testing.T) {
	disableRepositoryDetection(t)

	stdin, stdout, cancel := huhtest.NewResponder().
		AddResponse("GitHub Account Name", "my-account").
		AddResponse("Repository Name", "my-repo").
		AddConfirm("Include Binary Support?", huhtest.ConfirmAffirm).
		AddConfirm("Include Versioning Support?", huhtest.ConfirmAffirm).
		AddConfirm("Set GitHub Release Secrets?", huhtest.ConfirmAffirm).
		AddResponse("GoReleaser Token", "  goreleaser-token  ").
		AddResponse("Release Drafter Token", "  release-drafter-token  ").
		Start(t, 30*time.Second)
	defer cancel()

	cfg, err := loadConfigInteractive(stdin, stdout)
	require.NoError(t, err)
	assert.True(t, cfg.setupSecrets)
	assert.Equal(t, "goreleaser-token", cfg.goreleaserToken)
	assert.Equal(t, "release-drafter-token", cfg.releaseDrafterToken)
}

func TestReleaseTokenDescriptionsIncludeSecretNames(t *testing.T) {
	assert.Equal(
		t,
		"Personal access token for GoReleaser; creates GitHub Actions secret GORELEASER_TOKEN; input is hidden",
		goreleaserTokenDescription,
	)
	assert.Equal(
		t,
		"Personal access token for Release Drafter; creates GitHub Actions secret RELEASE_DRAFTER_TOKEN; input is hidden",
		releaseDrafterTokenDescription,
	)
}

func TestReleaseSecretsDescription(t *testing.T) {
	tests := map[string]struct {
		cfg  *config
		want string
	}{
		"binary only": {
			cfg: &config{
				includeBinary: true,
			},
			want: "Creates GitHub Actions secret GORELEASER_TOKEN",
		},
		"both release features": {
			cfg: &config{
				includeBinary:  true,
				includeVersion: true,
			},
			want: "Creates GitHub Actions secrets GORELEASER_TOKEN and RELEASE_DRAFTER_TOKEN",
		},
		"no release features": {
			cfg:  &config{},
			want: "No GitHub Actions release secrets will be created",
		},
		"versioning only": {
			cfg: &config{
				includeVersion: true,
			},
			want: "Creates GitHub Actions secret RELEASE_DRAFTER_TOKEN",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, releaseSecretsDescription(tt.cfg))
		})
	}
}

func TestLoadConfigInteractive_SkipsSecretsPromptWithoutReleaseFeatures(t *testing.T) {
	cfg, err := runLoadConfigInteractive(t, false, false)
	require.NoError(t, err)
	assert.False(t, cfg.setupSecrets)
	assert.Empty(t, cfg.goreleaserToken)
	assert.Empty(t, cfg.releaseDrafterToken)
}

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name                    string
		account                 string
		repo                    string
		noBinary                string
		noVersioning            string
		setupSecrets            string
		goreleaserToken         string
		releaseDrafterToken     string
		wantErr                 bool
		wantAccount             string
		wantRepo                string
		wantBinary              bool
		wantVersioning          bool
		wantSetup               bool
		wantGoreleaserToken     string
		wantReleaseDrafterToken string
	}{
		{
			name:    "missing BOOTSTRAP_REPO returns error",
			account: "my-account",
			repo:    "",
			wantErr: true,
		},
		{
			name:    "whitespace-only BOOTSTRAP_REPO returns error",
			account: "my-account",
			repo:    "   ",
			wantErr: true,
		},
		{
			name:    "whitespace-only BOOTSTRAP_ACCOUNT returns error",
			account: "   ",
			repo:    "my-repo",
			wantErr: true,
		},
		{
			name:           "whitespace is trimmed from account and repo",
			account:        "  my-account  ",
			repo:           "  my-repo  ",
			wantAccount:    "my-account",
			wantRepo:       "my-repo",
			wantBinary:     true,
			wantVersioning: true,
		},
		{
			name:           "BOOTSTRAP_NO_BINARY=true disables binary",
			account:        "my-account",
			repo:           "my-repo",
			noBinary:       "true",
			wantAccount:    "my-account",
			wantRepo:       "my-repo",
			wantBinary:     false,
			wantVersioning: true,
		},
		{
			name:           "BOOTSTRAP_NO_BINARY not set enables binary",
			account:        "my-account",
			repo:           "my-repo",
			wantAccount:    "my-account",
			wantRepo:       "my-repo",
			wantBinary:     true,
			wantVersioning: true,
		},
		{
			name:           "BOOTSTRAP_NO_VERSIONING=true disables versioning",
			account:        "my-account",
			repo:           "my-repo",
			noVersioning:   "true",
			wantAccount:    "my-account",
			wantRepo:       "my-repo",
			wantBinary:     true,
			wantVersioning: false,
		},
		{
			name:           "BOOTSTRAP_NO_VERSIONING not set enables versioning",
			account:        "my-account",
			repo:           "my-repo",
			wantAccount:    "my-account",
			wantRepo:       "my-repo",
			wantBinary:     true,
			wantVersioning: true,
		},
		{
			name:                    "BOOTSTRAP_SETUP_SECRETS=true stores trimmed tokens",
			account:                 "my-account",
			repo:                    "my-repo",
			setupSecrets:            "true",
			goreleaserToken:         "  goreleaser-token  ",
			releaseDrafterToken:     "  release-drafter-token  ",
			wantAccount:             "my-account",
			wantRepo:                "my-repo",
			wantBinary:              true,
			wantVersioning:          true,
			wantSetup:               true,
			wantGoreleaserToken:     "goreleaser-token",
			wantReleaseDrafterToken: "release-drafter-token",
		},
		{
			name:                    "BOOTSTRAP_SETUP_SECRETS=true with binary only stores GoReleaser token",
			account:                 "my-account",
			repo:                    "my-repo",
			noVersioning:            "true",
			setupSecrets:            "true",
			goreleaserToken:         "  goreleaser-token  ",
			wantAccount:             "my-account",
			wantRepo:                "my-repo",
			wantBinary:              true,
			wantVersioning:          false,
			wantSetup:               true,
			wantGoreleaserToken:     "goreleaser-token",
			wantReleaseDrafterToken: "",
		},
		{
			name:                    "BOOTSTRAP_SETUP_SECRETS=true with versioning only stores Release Drafter token",
			account:                 "my-account",
			repo:                    "my-repo",
			noBinary:                "true",
			setupSecrets:            "true",
			releaseDrafterToken:     "  release-drafter-token  ",
			wantAccount:             "my-account",
			wantRepo:                "my-repo",
			wantBinary:              false,
			wantVersioning:          true,
			wantSetup:               true,
			wantGoreleaserToken:     "",
			wantReleaseDrafterToken: "release-drafter-token",
		},
		{
			name:           "BOOTSTRAP_SETUP_SECRETS=true is ignored without release features",
			account:        "my-account",
			repo:           "my-repo",
			noBinary:       "true",
			noVersioning:   "true",
			setupSecrets:   "true",
			wantAccount:    "my-account",
			wantRepo:       "my-repo",
			wantBinary:     false,
			wantVersioning: false,
			wantSetup:      false,
		},
		{
			name:                "BOOTSTRAP_SETUP_SECRETS=true requires GoReleaser token",
			account:             "my-account",
			repo:                "my-repo",
			setupSecrets:        "true",
			releaseDrafterToken: "release-drafter-token",
			wantErr:             true,
		},
		{
			name:            "BOOTSTRAP_SETUP_SECRETS=true requires Release Drafter token",
			account:         "my-account",
			repo:            "my-repo",
			setupSecrets:    "true",
			goreleaserToken: "goreleaser-token",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BOOTSTRAP_ACCOUNT", tt.account)
			t.Setenv("BOOTSTRAP_REPO", tt.repo)
			t.Setenv("BOOTSTRAP_NO_BINARY", tt.noBinary)
			t.Setenv("BOOTSTRAP_NO_VERSIONING", tt.noVersioning)
			t.Setenv("BOOTSTRAP_SETUP_SECRETS", tt.setupSecrets)
			t.Setenv("BOOTSTRAP_GORELEASER_TOKEN", tt.goreleaserToken)
			t.Setenv("BOOTSTRAP_RELEASE_DRAFTER_TOKEN", tt.releaseDrafterToken)

			cfg, err := loadConfigFromEnv(tt.account)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantAccount, cfg.accountName)
			assert.Equal(t, tt.wantRepo, cfg.repoName)
			assert.Equal(t, tt.wantBinary, cfg.includeBinary)
			assert.Equal(t, tt.wantVersioning, cfg.includeVersion)
			assert.Equal(t, tt.wantSetup, cfg.setupSecrets)
			assert.Equal(t, tt.wantGoreleaserToken, cfg.goreleaserToken)
			assert.Equal(t, tt.wantReleaseDrafterToken, cfg.releaseDrafterToken)
		})
	}
}

func TestDetectRepositoryIdentity(t *testing.T) {
	tests := map[string]struct {
		gitOutput          string
		gitErr             error
		ghOutput           string
		ghErr              error
		wantAccount        string
		wantRepo           string
		wantGitCommands    [][]string
		wantGitHubCommands [][]string
	}{
		"git remote": {
			gitOutput:       "https://github.com/my-account/my-repo.git\n",
			wantAccount:     "my-account",
			wantRepo:        "my-repo",
			wantGitCommands: [][]string{{"git", "remote", "get-url", "origin"}},
		},
		"gh fallback": {
			gitErr:          errors.New("exit status 2"),
			ghOutput:        "my-account/my-repo\n",
			wantAccount:     "my-account",
			wantRepo:        "my-repo",
			wantGitCommands: [][]string{{"git", "remote", "get-url", "origin"}},
			wantGitHubCommands: [][]string{
				{"gh", "repo", "view", "--json", "owner,name", "--jq", ".owner.login + \"/\" + .name"},
			},
		},
		"no repository detected": {
			gitErr:          errors.New("exit status 2"),
			ghErr:           errors.New("exit status 1"),
			wantGitCommands: [][]string{{"git", "remote", "get-url", "origin"}},
			wantGitHubCommands: [][]string{
				{"gh", "repo", "view", "--json", "owner,name", "--jq", ".owner.login + \"/\" + .name"},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var gitCommands [][]string
			var gitHubCommands [][]string
			restoreCommandHooks(t)
			runGitCommand = func(stdin, name string, args ...string) ([]byte, error) {
				assert.Empty(t, stdin)
				command := append([]string{name}, args...)
				gitCommands = append(gitCommands, command)
				return []byte(tt.gitOutput), tt.gitErr
			}
			runGitHubCommand = func(stdin, name string, args ...string) ([]byte, error) {
				assert.Empty(t, stdin)
				command := append([]string{name}, args...)
				gitHubCommands = append(gitHubCommands, command)
				return []byte(tt.ghOutput), tt.ghErr
			}

			accountName, repoName := detectRepositoryIdentity()

			assert.Equal(t, tt.wantAccount, accountName)
			assert.Equal(t, tt.wantRepo, repoName)
			assert.Equal(t, tt.wantGitCommands, gitCommands)
			assert.Equal(t, tt.wantGitHubCommands, gitHubCommands)
		})
	}
}

func TestParseGitHubRepository(t *testing.T) {
	tests := map[string]struct {
		remoteURL   string
		wantAccount string
		wantRepo    string
	}{
		"https remote": {
			remoteURL:   "https://github.com/my-account/my-repo.git",
			wantAccount: "my-account",
			wantRepo:    "my-repo",
		},
		"http remote": {
			remoteURL:   "http://github.com/my-account/my-repo.git",
			wantAccount: "my-account",
			wantRepo:    "my-repo",
		},
		"scp-like ssh remote": {
			remoteURL:   "git@github.com:my-account/my-repo.git",
			wantAccount: "my-account",
			wantRepo:    "my-repo",
		},
		"ssh URL remote": {
			remoteURL:   "ssh://git@github.com/my-account/my-repo.git",
			wantAccount: "my-account",
			wantRepo:    "my-repo",
		},
		"host path remote": {
			remoteURL:   "github.com/my-account/my-repo",
			wantAccount: "my-account",
			wantRepo:    "my-repo",
		},
		"non-GitHub remote": {
			remoteURL: "git@example.com:my-account/my-repo.git",
		},
		"nested path": {
			remoteURL: "https://github.com/my-account/my-repo/extra.git",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			accountName, repoName := parseGitHubRepository(tt.remoteURL)
			assert.Equal(t, tt.wantAccount, accountName)
			assert.Equal(t, tt.wantRepo, repoName)
		})
	}
}

func TestSetupReleaseSecrets(t *testing.T) {
	tests := map[string]struct {
		cfg          *config
		lookPathErr  error
		commandErrs  map[string]error
		wantCommands [][]string
		wantStdin    []string
		wantErr      string
	}{
		"binary only": {
			cfg: &config{
				accountName:     "my-account",
				repoName:        "my-repo",
				includeBinary:   true,
				goreleaserToken: "goreleaser-token",
			},
			wantCommands: [][]string{
				{"gh", "auth", "status"},
				{"gh", "secret", "set", goreleaserSecretName, "--repo", "my-account/my-repo"},
			},
			wantStdin: []string{"", "goreleaser-token"},
		},
		"versioning only": {
			cfg: &config{
				accountName:         "my-account",
				repoName:            "my-repo",
				includeVersion:      true,
				releaseDrafterToken: "release-drafter-token",
			},
			wantCommands: [][]string{
				{"gh", "auth", "status"},
				{"gh", "secret", "set", releaseDrafterSecretName, "--repo", "my-account/my-repo"},
			},
			wantStdin: []string{"", "release-drafter-token"},
		},
		"both release features": {
			cfg: &config{
				accountName:         "my-account",
				repoName:            "my-repo",
				includeBinary:       true,
				includeVersion:      true,
				goreleaserToken:     "goreleaser-token",
				releaseDrafterToken: "release-drafter-token",
			},
			wantCommands: [][]string{
				{"gh", "auth", "status"},
				{"gh", "secret", "set", goreleaserSecretName, "--repo", "my-account/my-repo"},
				{"gh", "secret", "set", releaseDrafterSecretName, "--repo", "my-account/my-repo"},
			},
			wantStdin: []string{"", "goreleaser-token", "release-drafter-token"},
		},
		"no release features": {
			cfg: &config{
				accountName: "my-account",
				repoName:    "my-repo",
			},
		},
		"missing gh": {
			cfg: &config{
				accountName:     "my-account",
				repoName:        "my-repo",
				includeBinary:   true,
				goreleaserToken: "goreleaser-token",
			},
			lookPathErr: errors.New("executable file not found"),
			wantErr:     "GitHub CLI (gh) is required",
		},
		"auth failure": {
			cfg: &config{
				accountName:     "my-account",
				repoName:        "my-repo",
				includeBinary:   true,
				goreleaserToken: "goreleaser-token",
			},
			commandErrs: map[string]error{
				"gh auth status": errors.New("exit status 1"),
			},
			wantCommands: [][]string{
				{"gh", "auth", "status"},
			},
			wantStdin: []string{""},
			wantErr:   "gh auth status failed",
		},
		"secret set failure": {
			cfg: &config{
				accountName:     "my-account",
				repoName:        "my-repo",
				includeBinary:   true,
				goreleaserToken: "goreleaser-token",
			},
			commandErrs: map[string]error{
				"gh secret set GORELEASER_TOKEN --repo my-account/my-repo": errors.New("exit status 1"),
			},
			wantCommands: [][]string{
				{"gh", "auth", "status"},
				{"gh", "secret", "set", goreleaserSecretName, "--repo", "my-account/my-repo"},
			},
			wantStdin: []string{"", "goreleaser-token"},
			wantErr:   "gh secret set failed for GORELEASER_TOKEN",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var commands [][]string
			var stdins []string
			restoreCommandHooks(t)
			findExecutable = func(file string) (string, error) {
				assert.Equal(t, "gh", file)
				if tt.lookPathErr != nil {
					return "", tt.lookPathErr
				}
				return "/usr/bin/gh", nil
			}
			runGitHubCommand = func(stdin, name string, args ...string) ([]byte, error) {
				command := append([]string{name}, args...)
				commands = append(commands, command)
				stdins = append(stdins, stdin)
				if err := tt.commandErrs[strings.Join(command, " ")]; err != nil {
					return []byte("command failed"), err
				}
				return []byte("ok"), nil
			}

			err := setupReleaseSecrets(tt.cfg)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.NotContains(t, err.Error(), "goreleaser-token")
				assert.NotContains(t, err.Error(), "release-drafter-token")
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantCommands, commands)
			assert.Equal(t, tt.wantStdin, stdins)
		})
	}
}

func getExpectedJustfile(t *testing.T, includeBinary bool) string {
	t.Helper()

	if includeBinary {
		return expectedJustfileWithBinary
	}
	return expectedJustfileNoBinary
}

func restoreCommandHooks(t *testing.T) {
	t.Helper()

	originalFindExecutable := findExecutable
	originalRunGitCommand := runGitCommand
	originalRunGitHubCommand := runGitHubCommand
	t.Cleanup(func() {
		findExecutable = originalFindExecutable
		runGitCommand = originalRunGitCommand
		runGitHubCommand = originalRunGitHubCommand
	})
}

func disableRepositoryDetection(t *testing.T) {
	t.Helper()

	restoreCommandHooks(t)
	runGitCommand = func(string, string, ...string) ([]byte, error) {
		return nil, errors.New("git repository detection disabled")
	}
	runGitHubCommand = func(string, string, ...string) ([]byte, error) {
		return nil, errors.New("GitHub repository detection disabled")
	}
}

func runBootstrap(t *testing.T, tmpDir string, args ...string) (string, error) {
	t.Helper()

	// Build the Go bootstrap binary
	bootstrapDir := filepath.Join(tmpDir, "bootstrap")
	buildCmd := exec.Command("go", "build", "-o", "bootstrap-bin")
	buildCmd.Dir = bootstrapDir
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build bootstrap binary. Output: %s", output)

	// Parse arguments to extract flags and set environment variables
	accountName := ""
	repoName := ""
	noBinary := "false"
	noVersioning := "false"

	for _, arg := range args {
		switch arg {
		case "--no-binary":
			noBinary = "true"
		case "--no-versioning":
			noVersioning = "true"
		default:
			if accountName == "" {
				accountName = arg
			} else if repoName == "" {
				repoName = arg
			}
		}
	}

	// Run the bootstrap binary from the bootstrap directory (like the old script)
	// The tool expects to run from bootstrap/ and will cd .. to the project root
	bootstrapBin := filepath.Join(bootstrapDir, "bootstrap-bin")
	//nolint:gosec // G204: bootstrapBin is a test binary path, not user input
	cmd := exec.Command(bootstrapBin)
	cmd.Dir = bootstrapDir // Run from bootstrap/ directory

	// Only set environment variables if values are provided
	// This allows testing the error case when required args are missing
	env := os.Environ()
	if accountName != "" || repoName != "" {
		env = append(env,
			"BOOTSTRAP_ACCOUNT="+accountName,
			"BOOTSTRAP_REPO="+repoName,
			"BOOTSTRAP_NO_BINARY="+noBinary,
			"BOOTSTRAP_NO_VERSIONING="+noVersioning,
		)
	}
	cmd.Env = env

	output, err = cmd.CombinedOutput()
	return string(output), err
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	//nolint:gosec // G304: path is a test file path within t.TempDir()
	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file %s", path)
	return string(content)
}

func runLoadConfigInteractive(t *testing.T, includeBinary, includeVersion bool) (*config, error) {
	t.Helper()
	disableRepositoryDetection(t)

	includeBinaryResponse := huhtest.ConfirmNegative
	if includeBinary {
		includeBinaryResponse = huhtest.ConfirmAffirm
	}
	includeVersionResponse := huhtest.ConfirmNegative
	if includeVersion {
		includeVersionResponse = huhtest.ConfirmAffirm
	}

	responder := huhtest.NewResponder().
		AddResponse("GitHub Account Name", "my-account").
		AddResponse("Repository Name", "my-repo").
		AddConfirm("Include Binary Support?", includeBinaryResponse).
		AddConfirm("Include Versioning Support?", includeVersionResponse)
	if includeBinary || includeVersion {
		responder = responder.AddConfirm("Set GitHub Release Secrets?", huhtest.ConfirmNegative)
	}

	stdin, stdout, cancel := responder.Start(t, 30*time.Second)
	defer cancel()

	return loadConfigInteractive(stdin, stdout)
}

func copyProject(t *testing.T, dst string) {
	t.Helper()
	// We're in the bootstrap directory, so copy the parent (project root)
	src := filepath.Join(findModuleRoot(t), "..")
	src, err := filepath.Abs(src)
	require.NoError(t, err)
	err = os.CopyFS(dst, os.DirFS(src))
	require.NoError(t, err, "Failed to copy directory %s to %s", src, dst)
}

var (
	moduleRoot         string
	findModuleRootOnce sync.Once
)

func findModuleRoot(t *testing.T) string {
	t.Helper()
	findModuleRootOnce.Do(func() {
		// We're in bootstrap/, so just return current directory
		dir, err := os.Getwd()
		require.NoError(t, err, "Failed to get working directory")
		moduleRoot = filepath.Clean(dir)
	})
	return moduleRoot
}
