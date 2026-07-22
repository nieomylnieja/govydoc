// Package pkg hosts library code that's intended to be used by external applications.
package pkg

import "github.com/x-github-account-name/x-repo-name/internal"

// Demo returns a demo string.
func Demo() string {
	return "this is just a " + internal.GetDemo()
}
