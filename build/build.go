// Package build provides Go package building.
package build

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/skrashevich/gobinaries"
)

// environMap returns a map of environment variables.
var environMap map[string]string

// init initializes the environment variable map.
func init() {
	environMap = make(map[string]string)
	for _, v := range os.Environ() {
		parts := strings.Split(v, "=")
		environMap[parts[0]] = parts[1]
	}
}

// environWhitelist is a list of environment variables to include in Go sub-commands.
var environWhitelist = []string{
	"PATH",
	"HOME",
	"PWD",
	"GOPATH",
	"GOLANG_VERSION",
	"TMPDIR",
}

// ErrNotExecutable is returned when the package path provided does not produce a binary.
var ErrNotExecutable = errors.New("not executable")

// Error represents a build error.
type Error struct {
	err    error
	stderr string
}

// Error implementation.
func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.err.Error(), e.stderr)
}

// Write a package binary to w.
func Write(w io.Writer, bin gobinaries.Binary) error {
	dir, err := os.UserHomeDir()
	dir = filepath.Join(dir, ".cache", "gobinaries", bin.Module)
	if err != nil {
		return fmt.Errorf("user home dir: %w", err)
	}

	err = install(bin, dir)
	if err != nil {
		return fmt.Errorf("tidy module: %w", err)
	}
	var dst string
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Mode().Perm()&0111 != 0 { // check if file is executable
			dst = path
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}

	// check permissions and copy it to w
	f, err := os.Open(dst)
	if err != nil {
		return fmt.Errorf("opening: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stating: %w", err)
	}

	if !isExecutable(info.Mode()) {
		return ErrNotExecutable
	}

	_, err = io.Copy(w, f)
	if err != nil {
		return fmt.Errorf("copying: %w", err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("closing: %w", err)
	}

	err = os.RemoveAll(dir)
	if err != nil {
		return fmt.Errorf("cleaning: %w", err)
	}
	return nil
}

// ClearCache removes the module cache.
func ClearCache() error {
	cmd := exec.Command("go", "clean", "--modcache")
	return cmd.Run()
}

// isExecutable returns true if the exec bit is set for u/g/o.
func isExecutable(mode os.FileMode) bool {
	return mode.Perm()&0111 == 0111
}

// addModule initializes a new go module in the given dir. This is apparently
// necessary to build using Go modules since `go build` does not support
// semver, awkward UX but oh well.
func addModule(dir string) error {
	cmd := exec.Command("go", "mod", "init", "github.com/gobinary")
	cmd.Env = environ()
	cmd.Env = append(cmd.Env, "GO111MODULE=on")
	cmd.Dir = dir
	return command(cmd)
}

func install(bin gobinaries.Binary, dir string) error {
	ldflags := fmt.Sprintf("-s -w -X main.version=%s", bin.Version)
	cmd := exec.Command("go", "install", "-trimpath", "-ldflags", ldflags, bin.Module+"/...@"+bin.Version)
	cmd.Env = environ()
	cmd.Env = append(cmd.Env, "GOPATH="+dir)
	cmd.Env = append(cmd.Env, "CGO_ENABLED="+bin.CGO)
	cmd.Env = append(cmd.Env, "GOOS="+bin.OS)
	if strings.HasPrefix(bin.Arch, "armv") {
		cmd.Env = append(cmd.Env, "GOARCH=arm")
		cmd.Env = append(cmd.Env, "GOARM="+strings.TrimPrefix(bin.Arch, "armv"))
	} else {
		cmd.Env = append(cmd.Env, "GOARCH="+bin.Arch)
	}
	cmd.Dir, _ = os.UserHomeDir()
	return command(cmd)
}

// getMajorVersion tries to detect the major version of the package.
func getMajorVersion(tag string) (int, error) {
	major := strings.Split(tag, ".")[0]
	if len(major) < 1 {
		return 0, fmt.Errorf("invalid major version")
	}
	return strconv.Atoi(major[1:])
}

// normalizeModuleDep returns a normalized module dependency.
func normalizeModuleDep(bin gobinaries.Binary) string {
	mod := bin.Module
	version := bin.Version
	var dep string
	major, err := getMajorVersion(version)
	if err == nil && major > 1 {
		dep = fmt.Sprintf("%s/v%d@%s", mod, major, version)
	} else {
		dep = fmt.Sprintf("%s@%s", mod, version)
	}
	return dep
}

// addModuleDep creates a module dependency.
func addModuleDep(dir, dep string) error {
	cmd := exec.Command("go", "mod", "edit", "-require", dep)
	cmd.Env = environ()
	cmd.Env = append(cmd.Env, "GO111MODULE=on")
	cmd.Dir = dir
	return command(cmd)
}

// command executes a command and capture stderr.
func command(cmd *exec.Cmd) error {
	var w strings.Builder
	cmd.Stderr = &w
	err := cmd.Run()
	if err != nil {
		return Error{
			err:    err,
			stderr: strings.TrimSpace(w.String()),
		}
	}
	return nil
}

// tempFilename returns a new temporary file name.
func tempFilename() (string, error) {
	f, err := ioutil.TempFile(os.TempDir(), "gobinary")
	if err != nil {
		return "", err
	}
	defer f.Close()
	defer os.Remove(f.Name())
	return f.Name(), nil
}

// environ returns the environment variables for Go sub-commands.
func environ() (env []string) {
	for _, name := range environWhitelist {
		env = append(env, name+"="+environMap[name])
	}
	return
}
