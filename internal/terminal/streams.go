// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terminal

import (
	"fmt"
	"os"
)

// Streams represents a collection of three streams that each may or may not
// be connected to a terminal.
//
// If a stream is connected to a terminal then there are more possibilities
// available, such as detecting the current terminal width. If we're connected
// to something else, such as a pipe or a file on disk, the stream will
// typically provide placeholder values or do-nothing stubs for
// terminal-requiring operations.
//
// Note that it's possible for only a subset of the streams to be connected
// to a terminal. For example, this happens if the user runs Farseek with
// I/O redirection where Stdout might refer to a regular disk file while Stderr
// refers to a terminal, or various other similar combinations.
type Streams struct {
	Stdout *OutputStream
	Stderr *OutputStream
	Stdin  *InputStream
}

// Init tries to initialize a terminal, if Farseek is running in one, and
// returns an object describing what it was able to set up.
//
// An error for this function indicates that the current execution context
// can't meet Farseek's assumptions. For example, on Windows Init will return
// an error if Farseek is running in a Windows Console that refuses to
// activate UTF-8 mode, which can happen if we're running on an unsupported old
// version of Windows.
//
// Note that the success of this function doesn't mean that we're actually
// running in a terminal. It could also represent successfully detecting that
// one or more of the input/output streams is not a terminal.
func Init() (*Streams, error) {
	// These configure* functions are platform-specific functions in other
	// files that use //+build constraints to vary based on target OS.

	stderr, err := configureOutputHandle(os.Stderr)
	if err != nil {
		return nil, err
	}
	stdout, err := configureOutputHandle(os.Stdout)
	if err != nil {
		return nil, err
	}
	stdin, err := configureInputHandle(os.Stdin)
	if err != nil {
		return nil, err
	}

	return &Streams{
		Stdout: stdout,
		Stderr: stderr,
		Stdin:  stdin,
	}, nil
}

// Print is a helper for conveniently calling fmt.Fprint on the Stdout stream.
func (s *Streams) Print(a ...interface{}) (n int, err error) {
	return fmt.Fprint(s.Stdout.File, a...)
}

// Printf is a helper for conveniently calling fmt.Fprintf on the Stdout stream.
func (s *Streams) Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(s.Stdout.File, format, a...)
}

// Println is a helper for conveniently calling fmt.Fprintln on the Stdout stream.
func (s *Streams) Println(a ...interface{}) (n int, err error) {
	return fmt.Fprintln(s.Stdout.File, a...)
}

// Eprint is a helper for conveniently calling fmt.Fprint on the Stderr stream.
func (s *Streams) Eprint(a ...interface{}) (n int, err error) {
	return fmt.Fprint(s.Stderr.File, a...)
}

// Eprintf is a helper for conveniently calling fmt.Fprintf on the Stderr stream.
func (s *Streams) Eprintf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(s.Stderr.File, format, a...)
}

// Eprintln is a helper for conveniently calling fmt.Fprintln on the Stderr stream.
func (s *Streams) Eprintln(a ...interface{}) (n int, err error) {
	return fmt.Fprintln(s.Stderr.File, a...)
}
