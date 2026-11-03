//go:build !windows

package reliable

import (
	"context"
	"errors"
)

func pickProject(context.Context) (string, error) {
	return "", errors.New("folder picker is available on Windows; paste the project folder path")
}
