// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"os"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/rafagsiqueira/farseek/internal/addrs"
	"github.com/rafagsiqueira/farseek/internal/configs/configschema"
	"github.com/rafagsiqueira/farseek/internal/encryption"
	"github.com/rafagsiqueira/farseek/internal/providers"
	"github.com/rafagsiqueira/farseek/internal/states"
	"github.com/rafagsiqueira/farseek/internal/states/statefile"
)

func TestApply_destroy(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	t.Chdir(td)

	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
			addrs.NoKey,
		)
	})
	statePath := testStateFile(t, originalState)

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Computed: true},
						"ami": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		"-auto-approve",
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Log(output.Stdout())
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Verify a new state exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	stateFile, err := statefile.Read(f, encryption.StateEncryptionDisabled())
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if stateFile.State == nil {
		t.Fatal("state should not be nil")
	}

	actualStr := strings.TrimSpace(stateFile.State.String())
	expectedStr := strings.TrimSpace(testApplyDestroyStr)
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}

	// Should have a backup file
	f, err = os.Open(statePath + DefaultBackupExtension)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	backupStateFile, err := statefile.Read(f, encryption.StateEncryptionDisabled())
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actualStr = strings.TrimSpace(backupStateFile.State.String())
	expectedStr = strings.TrimSpace(originalState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestApply_destroyApproveNo(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	t.Chdir(td)

	// Create some existing state
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
			addrs.NoKey,
		)
	})
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()

	defer testInputMap(t, map[string]string{
		"approve": "no",
	})()

	// Do not use the NewMockUi initializer here, as we want to delay
	// the call to init until after setting up the input mocks
	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}
	if got, want := output.Stdout(), "Destroy cancelled"; !strings.Contains(got, want) {
		t.Fatalf("expected output to include %q, but was:\n%s", want, got)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}
	actualStr := strings.TrimSpace(state.String())
	expectedStr := strings.TrimSpace(originalState.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestApply_destroyApproveYes(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	t.Chdir(td)

	// Create some existing state
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
			addrs.NoKey,
		)
	})
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()

	defer testInputMap(t, map[string]string{
		"approve": "yes",
	})()

	// Do not use the NewMockUi initializer here, as we want to delay
	// the call to init until after setting up the input mocks
	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Log(output.Stdout())
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}

	actualStr := strings.TrimSpace(state.String())
	expectedStr := strings.TrimSpace(testApplyDestroyStr)
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

func TestApply_destroyPlan(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	t.Chdir(td)

	planPath := testPlanFileNoop(t)

	p := testProvider()
	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// Run the apply command pointing to our existing state
	args := []string{
		planPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}
	if !strings.Contains(output.Stderr(), "plan file") {
		t.Fatal("expected command output to refer to plan file, but got:", output.Stderr())
	}
}

func TestApply_destroyPath(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply"), td)
	t.Chdir(td)

	p := applyFixtureProvider()

	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-auto-approve",
		testFixturePath("apply"),
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}
	if !strings.Contains(output.Stderr(), "-chdir") {
		t.Fatal("expected command output to refer to -chdir flag, but got:", output.Stderr())
	}
}

func TestApply_destroySkipInConfigAndState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("skip-destroy"), td)
	t.Chdir(td)

	// Create some existing state
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:   []byte(`{"id":"baz"}`),
				Status:      states.ObjectReady,
				SkipDestroy: true,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
			addrs.NoKey,
		)
	})
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()

	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Log(output.Stdout())
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if !strings.Contains(output.Stderr(), "Farseek has not deleted some remote objects") {
		t.Fatalf("did not expect skip-destroy message in output:\n\n%s", output.Stderr())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}

	actualStr := strings.TrimSpace(state.String())
	expectedStr := strings.TrimSpace(testApplyDestroyStr)
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

// TestApply_destroySkipWithSuppressFlag tests that the -suppress-forget-errors
// flag suppresses the error when destroy mode leaves forgotten instances behind.
func TestApply_destroySkipWithSuppressFlag(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("skip-destroy"), td)
	t.Chdir(td)

	// Create some existing state with SkipDestroy set
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:   []byte(`{"id":"baz"}`),
				Status:      states.ObjectReady,
				SkipDestroy: true,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
			addrs.NoKey,
		)
	})
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()

	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// with the suppress flag, the destroy should succeed even with forgotten instances
	args := []string{
		"-suppress-forget-errors",
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Log(output.Stdout())
		t.Fatalf("expected success with -suppress-forget-errors, but got: %d\n\n%s", code, output.Stderr())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}

	// state should be empty after the destroy
	actualStr := strings.TrimSpace(state.String())
	expectedStr := strings.TrimSpace(testApplyDestroyStr)
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

// In this case, the user has removed skip-destroy from config, but it's still set in state.
// We will plan a new state first, which will remove the skip-destroy attribute from state and then proceed to destroy the resource
func TestApply_destroySkipInStateNotInConfig(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("skip-destroy/no-skip-in-config"), td)
	t.Chdir(td)

	// Create some existing state
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:   []byte(`{"id":"baz"}`),
				Status:      states.ObjectReady,
				SkipDestroy: true,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
			addrs.NoKey,
		)
	})
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()

	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Log(output.Stdout())
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}
	// We will be destroying the resource above
	if !strings.Contains(output.Stdout(), "1 destroyed") {
		t.Fatalf("resource should be destroyed, output:\n\n%s", output.Stdout())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}

	actualStr := strings.TrimSpace(state.String())
	expectedStr := strings.TrimSpace(testApplyDestroyStr)
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}

}
func TestApply_destroySkipInStateOrphaned(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("skip-destroy/empty"), td)
	t.Chdir(td)

	// Create some existing state
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:   []byte(`{"id":"baz"}`),
				Status:      states.ObjectReady,
				SkipDestroy: true,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
			addrs.NoKey,
		)
	})
	statePath := testStateFile(t, originalState)

	p := applyFixtureProvider()

	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Log(output.Stdout())
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if !strings.Contains(output.Stderr(), "Farseek has not deleted some remote objects") {
		t.Fatalf("did not expect skip-destroy message in output:\n\n%s", output.Stderr())
	}

	// Check action reason - we must clarify to user that the attribute is stored in state even if not in config
	if !strings.Contains(output.Stdout(), "lifecycle.destroy = false") {
		t.Fatalf("did not find expected lifecycle.destroy reason in output:\n\n%s", output.Stdout())
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testStateRead(t, statePath)
	if state == nil {
		t.Fatal("state should not be nil")
	}

	actualStr := strings.TrimSpace(state.String())
	expectedStr := strings.TrimSpace(testApplyDestroyStr)
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\n%s", actualStr, expectedStr)
	}
}

const testApplyDestroyStr = `
<no state>
`
