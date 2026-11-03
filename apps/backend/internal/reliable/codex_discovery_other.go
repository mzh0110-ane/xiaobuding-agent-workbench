//go:build !windows

package reliable

import "os"

func codexFileRedirected(info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}
