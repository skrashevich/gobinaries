package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/skrashevich/gobinaries"
)

// ErrObjectNotFound is returned from Get when no object is found for the specified key.
var ErrObjectNotFound = errors.New("no cloud storage object")

// Local is a local filesystem object store for binaries.
type Local struct {
	// Root is the root directory of the filesystem.
	Root string

	// Prefix is an optional object key prefix.
	Prefix string
}

// Create an object representing the package's binary.
func (l *Local) Create(ctx context.Context, r io.Reader, bin gobinaries.Binary) error {
	key := l.getKey(bin)
	path := filepath.Join(l.Root, key)

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return fmt.Errorf("making directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}

	defer f.Close()

	_, err = io.Copy(f, r)
	if err != nil {
		return fmt.Errorf("copying: %w", err)
	}

	return nil
}

// Get returns an object.
func (l *Local) Get(ctx context.Context, bin gobinaries.Binary) (io.ReadCloser, error) {
	key := l.getKey(bin)
	path := filepath.Join(l.Root, key)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, gobinaries.ErrObjectNotFound
		}
		return nil, fmt.Errorf("opening file: %w", err)
	}

	return f, nil
}

// getKey returns the object key in the form <pkg>/<binary>.
func (l *Local) getKey(bin gobinaries.Binary) string {
	dir := l.Prefix + "/" + strings.Replace(bin.Path, "/", "-", -1)
	file := fmt.Sprintf("%s-%s-%s-%s", bin.Version, bin.OS, bin.Arch, bin.CGO)
	return filepath.Join(dir, file)
}

func (l *Local) SetPrefix(prefix string) {
	l.Prefix = prefix
}
