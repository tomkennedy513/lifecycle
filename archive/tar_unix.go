//go:build unix

package archive

import (
	"archive/tar"
	"os"

	"golang.org/x/sys/unix"
)

// noFollow makes os.OpenFile refuse to follow a final-component symlink.
const noFollow = unix.O_NOFOLLOW

func setUmask(newMask int) (oldMask int) {
	return unix.Umask(newMask)
}

func createSymlink(hdr *tar.Header) error {
	return os.Symlink(hdr.Linkname, hdr.Name)
}

func addSysAttributes(hdr *tar.Header, fi os.FileInfo) {
}
