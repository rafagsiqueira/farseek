// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package e2etest

import (
	"strings"
	"testing"
)

func SanitizeStderr(s string) string {
	s = stripAnsi(s)
	for _, c := range []string{"╷", "│", "╵", "├", "─"} {
		s = strings.ReplaceAll(s, c, " ")
	}
	return strings.Join(strings.Fields(s), " ")
}

type farseekResult struct {
	t      *testing.T
	stdout string
	stderr string
	err    error
}

func (r farseekResult) Success() farseekResult {
	r.t.Helper()
	if r.err != nil {
		r.t.Errorf("unexpected error: %s\nstderr:\n%s", r.err, r.stderr)
	}
	return r
}

func (r farseekResult) Failure() farseekResult {
	r.t.Helper()
	if r.err == nil {
		r.t.Errorf("expected error, got success\nstdout:\n%s", r.stdout)
	}
	return r
}

func (r farseekResult) StderrContains(sub string) farseekResult {
	r.t.Helper()
	if !strings.Contains(SanitizeStderr(r.stderr), sub) {
		r.t.Errorf("missing string in stderr: %q\ngot:\n%s", sub, r.stderr)
	}
	return r
}

func (r farseekResult) Contains(sub string) farseekResult {
	r.t.Helper()
	if !strings.Contains(r.stdout, sub) {
		r.t.Errorf("missing string in stdout: %q\ngot:\n%s", sub, r.stdout)
	}
	return r
}
