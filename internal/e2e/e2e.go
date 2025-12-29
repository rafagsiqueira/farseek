// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package e2e

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rafagsiqueira/farseek/internal/encryption"
	"github.com/rafagsiqueira/farseek/internal/plans"
	"github.com/rafagsiqueira/farseek/internal/plans/planfile"
	"github.com/rafagsiqueira/farseek/internal/states"
	"github.com/rafagsiqueira/farseek/internal/states/statefile"
)

// Type Binary represents the combination of a compiled binary
// and a temporary working directory to run it in.
type Binary struct {
	binPath string
	workDir string
	env     []string
}

// NewBinary prepares a temporary directory containing the files from the
// given fixture and returns an instance of type binary that can run
// the generated binary in that directory.
//
// If the temporary directory cannot be created, a fixture of the given name
// cannot be found, or if an error occurs while _copying_ the fixture files,
// this function will panic. Tests should be written to assume that this
// function always succeeds.
func NewBinary(t *testing.T, binaryPath, workingDir string) *Binary {
	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		panic(err)
	}

	// For our purposes here we do a very simplistic file copy that doesn't
	// attempt to preserve file permissions, attributes, alternate data
	// streams, etc. Since we only have to deal with our own fixtures in
	// the testdata subdir, we know we don't need to deal with anything
	// of this nature.
	err = filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == workingDir {
			// nothing to do at the root
			return nil
		}

		if filepath.Base(path) == ".exists" {
			// We use this file just to let git know the "empty" fixture
			// exists. It is not used by any test.
			return nil
		}

		srcFn := path

		path, err = filepath.Rel(workingDir, path)
		if err != nil {
			return err
		}

		dstFn := filepath.Join(tmpDir, path)

		if info.IsDir() {
			return os.Mkdir(dstFn, os.ModePerm)
		}

		src, err := os.Open(srcFn)
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(dstFn, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return err
		}

		_, err = io.Copy(dst, src)
		if err != nil {
			return err
		}

		if err := src.Close(); err != nil {
			return err
		}
		if err := dst.Close(); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	return &Binary{
		binPath: binaryPath,
		workDir: tmpDir,
	}
}

// AddEnv appends an entry to the environment variable table passed to any
// commands subsequently run.
func (b *Binary) AddEnv(entry string) {
	b.env = append(b.env, entry)
}

// Cmd returns an exec.Cmd pre-configured to run the generated Farseek
// binary with the given arguments in the temporary working directory.
//
// The returned object can be mutated by the caller to customize how the
// process will be run, before calling Run.
func (b *Binary) Cmd(args ...string) *exec.Cmd {
	cmd := exec.Command(b.binPath, args...)
	cmd.Dir = b.workDir
	cmd.Env = os.Environ()

	cmd.Env = append(cmd.Env, b.env...)

	return cmd
}

// Run executes the generated Farseek binary with the given arguments
// and returns the bytes that it wrote to both stdout and stderr.
//
// This is a simple way to run Farseek for non-interactive commands
// that don't need any special environment variables. For more complex
// situations, use Cmd and customize the command before running it.
func (b *Binary) Run(args ...string) (stdout, stderr string, err error) {
	cmd := b.Cmd(args...)
	cmd.Stdin = nil
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	err = cmd.Run()
	stdout = cmd.Stdout.(*bytes.Buffer).String()
	stderr = cmd.Stderr.(*bytes.Buffer).String()
	return
}

// Path returns a file path within the temporary working directory by
// appending the given arguments as path segments.
func (b *Binary) Path(parts ...string) string {
	args := make([]string, 0, len(parts)+1)
	args = append(args, b.workDir)
	args = append(args, parts...)
	return filepath.Join(args...)
}

// OpenFile is a helper for easily opening a file from the working directory
// for reading.
func (b *Binary) OpenFile(path ...string) (*os.File, error) {
	flatPath := b.Path(path...)
	return os.Open(flatPath)
}

// ReadFile is a helper for easily reading a whole file from the working
// directory.
func (b *Binary) ReadFile(path ...string) ([]byte, error) {
	flatPath := b.Path(path...)
	return os.ReadFile(flatPath)
}

// WriteFile is a helper for easily writing a whole file to the working
// directory.
func (b *Binary) WriteFile(filename string, content string) error {
	path := b.Path(filename)
	return os.WriteFile(path, []byte(content), os.ModePerm)
}

// FileExists is a helper for easily testing whether a particular file
// exists in the working directory.
func (b *Binary) FileExists(path ...string) bool {
	flatPath := b.Path(path...)
	_, err := os.Stat(flatPath)
	return !os.IsNotExist(err)
}

// LocalState is a helper for easily reading the local backend's state file
// farseek.tfstate from the working directory.
func (b *Binary) LocalState() (*states.State, error) {
	return b.StateFromFile("farseek.tfstate")
}

// StateFromFile is a helper for easily reading a state snapshot from a file
// on disk relative to the working directory.
func (b *Binary) StateFromFile(filename string) (*states.State, error) {
	f, err := b.OpenFile(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stateFile, err := statefile.Read(f, encryption.StateEncryptionDisabled())
	if err != nil {
		return nil, fmt.Errorf("Error reading statefile: %w", err)
	}
	return stateFile.State, nil
}

// Plan is a helper for easily reading a plan file from the working directory.
func (b *Binary) Plan(path string) (*plans.Plan, error) {
	path = b.Path(path)
	pr, err := planfile.Open(path, encryption.PlanEncryptionDisabled())
	if err != nil {
		return nil, err
	}
	plan, err := pr.ReadPlan()
	if err != nil {
		return nil, err
	}
	return plan, nil
}

// SetLocalState is a helper for easily writing to the file the local backend
// uses for state in the working directory. This does not go through the
// actual local backend code, so processing such as management of serials
// does not apply and the given state will simply be written verbatim.
func (b *Binary) SetLocalState(state *states.State) error {
	path := b.Path("farseek.tfstate")
	f, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create temporary state file %s: %w", path, err)
	}
	defer f.Close()

	sf := &statefile.File{
		Serial:  0,
		Lineage: "fake-for-testing",
		State:   state,
	}
	return statefile.Write(sf, f, encryption.StateEncryptionDisabled())
}

func GoBuild(pkgPath, tmpPrefix string) string {
	if runtime.GOOS == "windows" {
		tmpPrefix += ".exe"
	}

	dir, prefix := filepath.Split(tmpPrefix)
	tmpFile, err := os.CreateTemp(dir, prefix)
	if err != nil {
		panic(err)
	}
	tmpFilename := tmpFile.Name()
	if err = tmpFile.Close(); err != nil {
		panic(err)
	}

	args := []string{
		"go",
		"build",
	}

	if len(os.Getenv("GOCOVERDIR")) != 0 {
		args = append(args,
			"-cover",
			"-coverpkg=github.com/rafagsiqueira/farseek/...",
		)
	}

	args = append(args,
		"-o", tmpFilename,
		pkgPath,
	)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err = cmd.Run()
	if err != nil {
		// The go compiler will have already produced some error messages
		// on stderr by the time we get here.
		panic(fmt.Sprintf("failed to build executable: %s", err))
	}

	return tmpFilename
}

// WorkDir() returns the binary workdir
func (b *Binary) WorkDir() string {
	return b.workDir
}
