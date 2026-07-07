package archive

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

type PathMode struct {
	Path string
	Mode os.FileMode
}

var (
	umaskLock      sync.Mutex
	extractCounter int
	originalUmask  int
)

func setUmaskIfNeeded() {
	umaskLock.Lock()
	defer umaskLock.Unlock()
	extractCounter++
	if extractCounter == 1 {
		originalUmask = setUmask(0)
	}
}

func unsetUmaskIfNeeded() {
	umaskLock.Lock()
	defer umaskLock.Unlock()
	extractCounter--
	if extractCounter == 0 {
		_ = setUmask(originalUmask)
	}
}

// Extract reads all entries from TarReader and extracts them to the filesystem.
func Extract(tr TarReader, destRoot string) error {
	setUmaskIfNeeded()
	defer unsetUmaskIfNeeded()

	buf := make([]byte, 32*32*1024)
	dirsFound := make(map[string]bool)
	symlinksCreated := make(map[string]bool)

	var pathModes []PathMode
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			for _, pathMode := range pathModes { // directories that are newly created and for which there is a header in the tar should have the right permissions
				if err := os.Chmod(pathMode.Path, pathMode.Mode); err != nil {
					return err
				}
			}
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "error extracting from archive")
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := checkPathIsSafe(hdr.Name, destRoot, symlinksCreated, true); err != nil {
				return errors.Wrapf(err, "refusing to extract directory %q", hdr.Name)
			}
			if _, err := os.Stat(hdr.Name); os.IsNotExist(err) {
				pathMode := PathMode{hdr.Name, hdr.FileInfo().Mode()}
				pathModes = append(pathModes, pathMode)
			}
			if err := os.MkdirAll(hdr.Name, os.ModePerm); err != nil {
				return errors.Wrapf(err, "failed to create directory %q", hdr.Name)
			}
			dirsFound[hdr.Name] = true

		case tar.TypeReg:
			if err := checkPathIsSafe(hdr.Name, destRoot, symlinksCreated, false); err != nil {
				return errors.Wrapf(err, "refusing to extract file %q", hdr.Name)
			}
			dirPath := filepath.Dir(hdr.Name)
			if !dirsFound[dirPath] {
				if _, err := os.Stat(dirPath); os.IsNotExist(err) {
					if err := os.MkdirAll(dirPath, applyUmask(os.ModePerm, originalUmask)); err != nil { // if there is no header for the parent directory in the tar, apply the provided umask
						return errors.Wrapf(err, "failed to create parent dir %q for file %q", dirPath, hdr.Name)
					}
					dirsFound[dirPath] = true
				}
			}

			if err := writeFile(tr, hdr.Name, hdr.FileInfo().Mode(), buf); err != nil {
				return errors.Wrapf(err, "failed to write file %q", hdr.Name)
			}
		case tar.TypeSymlink:
			if err := checkPathIsSafe(hdr.Name, destRoot, symlinksCreated, false); err != nil {
				return errors.Wrapf(err, "refusing to create symlink %q", hdr.Name)
			}
			if err := validateSymlinkTarget(hdr.Name, hdr.Linkname, destRoot); err != nil {
				return errors.Wrapf(err, "failed to validate symlink target %q for symlink %q", hdr.Linkname, hdr.Name)
			}
			if err := createSymlink(hdr); err != nil {
				return errors.Wrapf(err, "failed to create symlink %q with target %q", hdr.Name, hdr.Linkname)
			}
			symlinksCreated[hdr.Name] = true
		case tar.TypeXGlobalHeader:
			// ignore PAX Global Extended Headers
			continue
		default:
			return fmt.Errorf("unknown file type in tar %d", hdr.Typeflag)
		}
	}
}

func applyUmask(mode os.FileMode, umask int) os.FileMode {
	return os.FileMode(int(mode) &^ umask)
}

func writeFile(in io.Reader, path string, mode os.FileMode, buf []byte) (err error) {
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|noFollow, mode)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := fh.Close(); err == nil {
			err = closeErr
		}
	}()
	_, err = io.CopyBuffer(fh, in, buf)
	return err
}

func validateSymlinkTarget(name, linkname, destRoot string) error {
	target := linkname
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(name), target)
	}
	target = filepath.Clean(target)
	root := filepath.Clean(destRoot)
	rel, err := filepath.Rel(root, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("symlink %q -> %q escapes destination root", name, linkname)
	}

	return nil
}

func checkPathIsSafe(path, destRoot string, symlinksCreated map[string]bool, isDir bool) error {
	rel, err := filepath.Rel(destRoot, path)
	if err != nil {
		return fmt.Errorf("failed to determine path %q relative to destination root %q: %w", path, destRoot, err)
	}
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		// Permit strict ancestors of destRoot, but only for directory entries
		// (parent TypeDir entries emitted by layers.DirLayer). A non-dir entry at
		// an ancestor path is never legitimate and stays rejected here.
		if isDir {
			cleanPath := filepath.Clean(path) + string(filepath.Separator)
			if strings.HasPrefix(destRoot+string(filepath.Separator), cleanPath) {
				return nil
			}
		}
		return fmt.Errorf("path %q escapes destination root %q", path, destRoot)
	}

	current := destRoot
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}

		current = filepath.Join(current, part)
		if symlinksCreated[current] {
			return fmt.Errorf("path component %q of %q is a symlink created during extraction", current, path)
		}
	}

	return nil
}
