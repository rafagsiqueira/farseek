// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"

	"github.com/rafagsiqueira/farseek/internal/addrs"
	"github.com/rafagsiqueira/farseek/internal/checks"
	"github.com/rafagsiqueira/farseek/internal/configs/configschema"
	"github.com/rafagsiqueira/farseek/internal/encryption"
	farseek "github.com/rafagsiqueira/farseek/internal/farseek"
	"github.com/rafagsiqueira/farseek/internal/plans"
	"github.com/rafagsiqueira/farseek/internal/plans/planfile"
	"github.com/rafagsiqueira/farseek/internal/providers"
	"github.com/rafagsiqueira/farseek/internal/states"
	"github.com/rafagsiqueira/farseek/internal/states/statefile"
	"github.com/rafagsiqueira/farseek/internal/tfdiags"
)

func TestPlan(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}
}
func TestPlan_conditionalSensitive(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-plan-conditional-sensitive"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{}
	code := c.Run(args)
	output := done(t).Stderr()
	if code != 1 {
		t.Fatalf("bad status code: %d\n\n%s", code, output)
	}

	if strings.Count(output, "Output refers to sensitive values") != 9 {
		t.Fatal("Not all outputs have issue with refer to sensitive value", output)
	}
}

func TestPlan_plan(t *testing.T) {
	testCwdTemp(t)

	planPath := testPlanFileNoop(t)

	p := testProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{planPath}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("wrong exit status %d; want 1\nstderr: %s", code, output.Stderr())
	}
}

func TestPlan_destroy(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
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
	outPath := testTempFile(t)
	statePath := testStateFile(t, originalState)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-destroy",
		"-out", outPath,
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	plan := testReadPlan(t, outPath)
	for _, rc := range plan.Changes.Resources {
		if got, want := rc.Action, plans.Delete; got != want {
			t.Fatalf("wrong action %s for %s; want %s\nplanned change: %s", got, rc.Addr, want, spew.Sdump(rc))
		}
	}
}

func TestPlan_noState(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Verify that refresh was called
	if p.ReadResourceCalled {
		t.Fatal("ReadResource should not be called")
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.NullVal(p.GetProviderSchemaResponse.ResourceTypes["test_instance"].Block.ImpliedType())
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestPlan_noTestVars(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-no-test-vars"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	code := c.Run([]string{})
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Verify values defined in the 'tests' folder are not used during a plan action
	expectedResult := `ami = "ValueFROMmain/tfvars"`
	result := output.All()
	if !strings.Contains(result, expectedResult) {
		t.Fatalf("Expected output to contain '%s', got: %s", expectedResult, result)
	}

	expectedToNotExist := "ValueFROMtests/tfvars"
	if strings.Contains(result, expectedToNotExist) {
		t.Fatalf("Expected output to not contain '%s', got: %s", expectedToNotExist, result)
	}
}

func TestPlan_generatedConfigPath(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-import-config-gen"), td)
	t.Chdir(td)

	genPath := filepath.Join(td, "generated.tf")

	p := planFixtureProvider()
	view, done := testView(t)

	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("bar"),
				}),
				Private: nil,
			},
		},
	}

	args := []string{
		"-generate-config-out", genPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	testFileEquals(t, genPath, filepath.Join(td, "generated.tf.expected"))
}

func TestPlan_outPath(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	outPath := filepath.Join(td, "test.plan")

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.NullVal(cty.EmptyObject),
	}

	args := []string{
		"-out", outPath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	testReadPlan(t, outPath) // will call t.Fatal itself if the file cannot be read
}

func TestPlan_outPathNoChange(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				// Aside from "id" (which is computed) the values here must
				// exactly match the values in the "plan" test fixture in order
				// to produce the empty plan we need for this test.
				AttrsJSON: []byte(`{"id":"bar","ami":"bar","network_interface":[{"description":"Main network interface","device_index":"0"}]}`),
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

	outPath := filepath.Join(td, "test.plan")

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-out", outPath,
		"-state", statePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	plan := testReadPlan(t, outPath)
	if !plan.Changes.Empty() {
		t.Fatalf("Expected empty plan to be written to plan file, got: %s", spew.Sdump(plan))
	}
}

func TestPlan_outPathWithError(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-fail-condition"), td)
	t.Chdir(td)

	outPath := filepath.Join(td, "test.plan")

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	p.PlanResourceChangeResponse = &providers.PlanResourceChangeResponse{
		PlannedState: cty.NullVal(cty.EmptyObject),
	}

	args := []string{
		"-out", outPath,
	}
	code := c.Run(args)
	output := done(t)
	if code == 0 {
		t.Fatal("expected non-zero exit status", output)
	}

	plan := testReadPlan(t, outPath) // will call t.Fatal itself if the file cannot be read
	if !plan.Errored {
		t.Fatal("plan should be marked with Errored")
	}

	if plan.Checks == nil {
		t.Fatal("plan contains no checks")
	}

	// the checks should only contain one failure
	results := plan.Checks.ConfigResults.Elements()
	if len(results) != 1 {
		t.Fatal("incorrect number of check results", len(results))
	}
	if results[0].Value.Status != checks.StatusFail {
		t.Errorf("incorrect status, got %s", results[0].Value.Status)
	}
}

func TestPlan_refreshFalse(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-existing-state"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-refresh=false",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if p.ReadResourceCalled {
		t.Fatal("ReadResource should not have been called")
	}
}

func TestPlan_refreshTrue(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-existing-state"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-refresh=true",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if !p.ReadResourceCalled {
		t.Fatalf("ReadResource should have been called")
	}
}

// A consumer relies on the fact that running
// farseek plan -refresh=false -refresh=true gives the same result as
// farseek plan -refresh=true.
// While the flag logic itself is handled by the stdlib flags package (and code
// in main() that is tested elsewhere), we verify the overall plan command
// behaviour here in case we accidentally break this with additional logic.
func TestPlan_refreshFalseRefreshTrue(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-existing-state"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-refresh=false",
		"-refresh=true",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should have been called")
	}
}

func TestPlan_state(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	originalState := testState()
	statePath := testStateFile(t, originalState)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
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
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("bar"),
		"ami": cty.NullVal(cty.String),
		"network_interface": cty.ListValEmpty(cty.Object(map[string]cty.Type{
			"device_index": cty.String,
			"description":  cty.String,
		})),
	})
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestPlan_stateDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	// Generate state and move it to the default path
	originalState := testState()
	statePath := testStateFile(t, originalState)
	if err := os.Rename(statePath, path.Join(td, "farseek.tfstate")); err != nil {
		t.Fatal(err)
	}

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// Verify that the provider was called with the existing state
	actual := p.PlanResourceChangeRequest.PriorState
	expected := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("bar"),
		"ami": cty.NullVal(cty.String),
		"network_interface": cty.ListValEmpty(cty.Object(map[string]cty.Type{
			"device_index": cty.String,
			"description":  cty.String,
		})),
	})
	if !expected.RawEquals(actual) {
		t.Fatalf("wrong prior state\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestPlan_validate(t *testing.T) {
	// This is triggered by not asking for input so we have to set this to false
	test = false
	defer func() { test = true }()

	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-invalid"), td)
	t.Chdir(td)

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true, Computed: true},
					},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{"-no-color"}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	actual := output.Stderr()
	if want := "Error: Invalid count argument"; !strings.Contains(actual, want) {
		t.Fatalf("unexpected error output\ngot:\n%s\n\nshould contain: %s", actual, want)
	}
	if want := "9:   count = timestamp()"; !strings.Contains(actual, want) {
		t.Fatalf("unexpected error output\ngot:\n%s\n\nshould contain: %s", actual, want)
	}
}

func TestPlan_vars(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-vars"), td)
	t.Chdir(td)

	p := planVarsFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	actual := ""
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		actual = req.ProposedNewState.GetAttr("value").AsString()
		resp.PlannedState = req.ProposedNewState
		return
	}

	args := []string{
		"-var", "foo=bar",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestPlan_varsInvalid(t *testing.T) {
	testCases := []struct {
		args    []string
		wantErr string
	}{
		{
			[]string{"-var", "foo"},
			`The given -var option "foo" is not correctly specified.`,
		},
		{
			[]string{"-var", "foo = bar"},
			`Variable name "foo " is invalid due to trailing space.`,
		},
	}

	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-vars"), td)
	t.Chdir(td)

	for _, tc := range testCases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			p := planVarsFixtureProvider()
			view, done := testView(t)
			c := &PlanCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					View:             view,
				},
			}

			code := c.Run(tc.args)
			output := done(t)
			if code != 1 {
				t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
			}

			got := output.Stderr()
			if !strings.Contains(got, tc.wantErr) {
				t.Fatalf("bad error output, want %q, got:\n%s", tc.wantErr, got)
			}
		})
	}
}

func TestPlan_varsUnset(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-vars"), td)
	t.Chdir(td)

	// The plan command will prompt for interactive input of var.foo.
	// We'll answer "bar" to that prompt, which should then allow this
	// configuration to apply even though var.foo doesn't have a
	// default value and there are no -var arguments on our command line.

	// This will (helpfully) panic if more than one variable is requested during plan:
	// https://github.com/hashicorp/terraform/issues/26027
	close := testInteractiveInput(t, []string{"bar"})
	defer close()

	p := planVarsFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}
}

// This test adds a required argument to the test provider to validate
// processing of user input:
// https://github.com/hashicorp/terraform/issues/26035
func TestPlan_providerArgumentUnset(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	// The plan command will prompt for interactive input of provider.test.region
	defaultInputReader = bytes.NewBufferString("us-east-1\n")

	p := planFixtureProvider()
	// override the planFixtureProvider schema to include a required provider argument
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"region": {Type: cty.String, Required: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Optional: true, Computed: true},
						"ami": {Type: cty.String, Optional: true, Computed: true},
					},
					BlockTypes: map[string]*configschema.NestedBlock{
						"network_interface": {
							Nesting: configschema.NestingList,
							Block: configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"device_index": {Type: cty.String, Optional: true},
									"description":  {Type: cty.String, Optional: true},
								},
							},
						},
					},
				},
			},
		},
		DataSources: map[string]providers.Schema{
			"test_data_source": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
						"valid": {
							Type:     cty.Bool,
							Computed: true,
						},
					},
				},
			},
		},
	}
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}
}

// based on the TestPlan_varsUnset test
// this is to check if we use the static variable in a resource
// does Plan ask for input
func TestPlan_resource_variable_inputs(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-vars-unset"), td)
	t.Chdir(td)

	// The plan command will prompt for interactive input of var.src.
	// We'll answer "mod" to that prompt, which should then allow this
	// configuration to apply even though var.src doesn't have a
	// default value and there are no -var arguments on our command line.

	// This will (helpfully) panic if more than one variable is requested during plan:
	// https://github.com/hashicorp/terraform/issues/26027
	inputClose := testInteractiveInput(t, []string{"./mod"})
	defer inputClose()

	p := planVarsFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}
}

// This test deals with a particularly-narrow bad interaction between different
// components in Farseek:
//
//   - The main language runtime uses static references in the configuration as
//     part of a heuristic to produce a set of "relevant attributes", whose
//     changes outside of Farseek (if any) might be interesting to call out
//     in the plan diff UI.
//   - The "try" and "can" functions allow there to be references to resources
//     with attribute paths that aren't actually correct for the resource
//     instance's value, which the "relevant attributes" analysis does not check
//     because it's analyzing based only on static traversal syntax.
//   - The human-oriented plan renderer tries to correlate changes described in
//     the "resource drift" part of the plan with paths appearing in
//     "relevant attributes" to limit the rendering of "Changes outside of
//     Farseek" only to changes that seem like they might have contributed to
//     the set of planned changes. Because of the previous two points, the
//     "relevant attributes" attribute path data cannot be trusted to
//     definitely conform to the schema of the indicated resource type and so
//     the renderer must tolerate inconsistencies without crashing or returning
//     an error.
//
// Because this potential problem is in the collaboration between three separate
// subsystems, we test it here so we can exercise all three in a relatively
// realistic way. This is therefore an integration test of these three
// components working together, rather than a test of of the plan command
// specifically.
func TestPlan_withInvalidReferencesInTry(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-invalid-reference-try"), td)
	t.Chdir(td)

	provider := &farseek.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{
				Block: &configschema.Block{},
			},
			ResourceTypes: map[string]providers.Schema{
				"test": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"values": {
								// Accepting any type here is important because
								// it forces references to nested values to
								// be type-checked only at runtime.
								// This is similar to how providers handle
								// situations where the schema is decided
								// dynamically by the remote server, as with
								// "kubernetes_manifest" and "helm_release"
								// in those respective providers.
								Type:     cty.DynamicPseudoType,
								Required: true,
							},
						},
					},
				},
			},
		},
		ReadResourceFn: func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
			// The following intentionally changes the "phase" attribute of
			// the values map, if present, to create a situation that the
			// language runtime would recognize as "changes outside of Farseek",
			// reported as "resource_drift" in the plan JSON.
			//
			// Because test.b.values is derived from test.a.values, that "drift"
			// is considered to be a relevant change for the plan UI to render.
			given := req.PriorState.GetAttr("values")
			updated := req.PriorState
			if given.HasIndex(cty.StringVal("phase")).True() {
				values := given.AsValueMap()
				values["phase"] = cty.StringVal("drifted")
				updated = cty.ObjectVal(map[string]cty.Value{
					"values": cty.MapVal(values),
				})
			}
			return providers.ReadResourceResponse{
				NewState: updated,
			}
		},
	}

	rsrcA := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test",
		Name: "a",
	}.Absolute(addrs.RootModuleInstance)
	instA := rsrcA.Instance(addrs.NoKey)
	rsrcB := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test",
		Name: "b",
	}.Absolute(addrs.RootModuleInstance)
	instB := rsrcB.Instance(addrs.NoKey)
	providerConfig := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("test"),
	}

	// We need a prior state where both resource instances already exist
	// so that we can detect the "drift" and try to report it.
	priorState := states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceProvider(rsrcA, providerConfig)
		ss.SetResourceInstanceCurrent(
			instA,
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"values":{"type":["map","string"],"value":{"phase":"initial"}}}`),
			},
			providerConfig, addrs.NoKey,
		)
		ss.SetResourceProvider(rsrcB, providerConfig)
		ss.SetResourceInstanceCurrent(
			instB,
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"values":{"type":["map","string"],"value":{"a_phase":"initial"}}}`),
			},
			providerConfig, addrs.NoKey,
		)
	})
	f, err := os.Create(DefaultStateFilename)
	if err != nil {
		t.Fatal(err)
	}
	err = statefile.Write(
		&statefile.File{
			Serial:  1,
			Lineage: "...",
			State:   priorState,
		},
		f,
		encryption.StateEncryptionDisabled(),
	)
	if err != nil {
		t.Fatal(err)
	}

	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider),
			View:             view,
		},
	}

	code := c.Run([]string{"-out=tfplan"})
	output := done(t)
	if code != 0 {
		t.Fatalf("unexpected error: %d\n\n%s", code, output.Stderr())
	}

	// If we get here without the plan phase failing then we've passed the
	// test, but we'll also inspect the saved plan file to make sure that
	// this plan meets the conditions we were intending to test, since
	// the exact heuristic used to decide which attributes are "relevant"
	// may change over time in a way that requires this test to be set up
	// a little differently.
	planReader, err := planfile.Open("tfplan", encryption.PlanEncryptionDisabled())
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planReader.ReadPlan()
	if err != nil {
		t.Fatal(err)
	}

	// We need to have detected that test.a changed during refresh.
	foundDrifted := false
	for _, change := range plan.DriftedResources {
		if change.Addr.Equal(instA) {
			foundDrifted = true
		}
	}
	if !foundDrifted {
		t.Errorf("plan does not report %s under DriftedResources", instA)
	}

	// We need to have detected both of the references in test.b as
	// "relevant", so that the plan renderer would've tried to correlate
	// them both with the before and after values in the DriftedResources
	// entry for test.a.
	foundRefWithIndex := false
	foundRefWithoutIndex := false
	for _, attrRef := range plan.RelevantAttributes {
		if !attrRef.Resource.Equal(instA) {
			continue // we only care about references to test.a
		}
		// For ease of comparison we'll use the diagnostic string representation
		// of the path. If the details of this string rendering intentionally
		// change in future then it's okay to update the following to match
		// those changes as long as it still describes the same cty.Path content.
		gotPath := tfdiags.FormatCtyPath(attrRef.Attr)
		if gotPath == `.values[0].phase` {
			// (this is the one that doesn't match the value and so could
			// cause problems if the plan renderer does not handle that.)
			foundRefWithIndex = true
		}
		if gotPath == `.values.phase` {
			foundRefWithoutIndex = true
		}
	}
	if !foundRefWithIndex {
		t.Errorf("plan does not report test.a.values[0].phase as a relevant attribute")
	}
	if !foundRefWithoutIndex {
		t.Errorf("plan does not report test.a.values.phase as a relevant attribute")
	}
}

// Test that farseek properly merges provider configuration that's split
// between config files and interactive input variables.
// https://github.com/hashicorp/terraform/issues/28956
func TestPlan_providerConfigMerge(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-provider-input"), td)
	t.Chdir(td)

	// Disable test mode so input would be asked
	test = false
	defer func() { test = true }()

	// The plan command will prompt for interactive input of provider.test.region
	defaultInputReader = bytes.NewBufferString("us-east-1\n")

	p := planFixtureProvider()
	// override the planFixtureProvider schema to include a required provider argument and a nested block
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"region": {Type: cty.String, Required: true},
					"url":    {Type: cty.String, Required: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"auth": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"user":     {Type: cty.String, Required: true},
								"password": {Type: cty.String, Required: true},
							},
						},
					},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true, Computed: true},
					},
				},
			},
		},
	}

	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if !p.ConfigureProviderCalled {
		t.Fatal("configure provider not called")
	}

	// For this test, we want to confirm that we've sent the expected config
	// value *to* the provider.
	got := p.ConfigureProviderRequest.Config
	want := cty.ObjectVal(map[string]cty.Value{
		"auth": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"user":     cty.StringVal("one"),
				"password": cty.StringVal("onepw"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"user":     cty.StringVal("two"),
				"password": cty.StringVal("twopw"),
			}),
		}),
		"region": cty.StringVal("us-east-1"),
		"url":    cty.StringVal("example.com"),
	})

	if !got.RawEquals(want) {
		t.Fatal("wrong provider config")
	}

}

func TestPlan_varFile(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-vars"), td)
	t.Chdir(td)

	varFilePath := testTempFile(t)
	if err := os.WriteFile(varFilePath, []byte(planVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	p := planVarsFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	actual := ""
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		actual = req.ProposedNewState.GetAttr("value").AsString()
		resp.PlannedState = req.ProposedNewState
		return
	}

	args := []string{
		"-var-file", varFilePath,
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestPlan_varFileDefault(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-vars"), td)
	t.Chdir(td)

	varFilePath := filepath.Join(td, "terraform.tfvars")
	if err := os.WriteFile(varFilePath, []byte(planVarFile), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	p := planVarsFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	actual := ""
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		actual = req.ProposedNewState.GetAttr("value").AsString()
		resp.PlannedState = req.ProposedNewState
		return
	}

	args := []string{}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if actual != "bar" {
		t.Fatal("didn't work")
	}
}

func TestPlan_varFileWithDecls(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-vars"), td)
	t.Chdir(td)

	varFilePath := testTempFile(t)
	if err := os.WriteFile(varFilePath, []byte(planVarFileWithDecl), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	p := planVarsFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-var-file", varFilePath,
	}
	code := c.Run(args)
	output := done(t)
	if code == 0 {
		t.Fatalf("succeeded; want failure\n\n%s", output.Stdout())
	}

	msg := output.Stderr()
	if got, want := msg, "Variable declaration in .tfvars file"; !strings.Contains(got, want) {
		t.Fatalf("missing expected error message\nwant message containing %q\ngot:\n%s", want, got)
	}
}

func TestPlan_detailedExitcode(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	t.Run("return 1", func(t *testing.T) {
		view, done := testView(t)
		c := &PlanCommand{
			Meta: Meta{
				// Running plan without setting testingOverrides is similar to plan without init
				View: view,
			},
		}
		code := c.Run([]string{"-detailed-exitcode"})
		output := done(t)
		if code != 1 {
			t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
		}
	})

	t.Run("return 2", func(t *testing.T) {
		p := planFixtureProvider()
		view, done := testView(t)
		c := &PlanCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		}

		code := c.Run([]string{"-detailed-exitcode"})
		output := done(t)
		if code != 2 {
			t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
		}
	})
}

func TestPlan_detailedExitcode_emptyDiff(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-emptydiff"), td)
	t.Chdir(td)

	p := testProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{"-detailed-exitcode"}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}
}

func TestPlan_shutdown(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-shutdown"), td)
	t.Chdir(td)

	cancelled := make(chan struct{})
	shutdownCh := make(chan struct{})

	p := testProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
			ShutdownCh:       shutdownCh,
		},
	}

	p.StopFn = func() error {
		close(cancelled)
		return nil
	}

	var once sync.Once

	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		once.Do(func() {
			shutdownCh <- struct{}{}
		})

		// Because of the internal lock in the MockProvider, we can't
		// coordinate directly with the calling of Stop, and making the
		// MockProvider concurrent is disruptive to a lot of existing tests.
		// Wait here a moment to help make sure the main goroutine gets to the
		// Stop call before we exit, or the plan may finish before it can be
		// canceled.
		time.Sleep(200 * time.Millisecond)

		s := req.ProposedNewState.AsValueMap()
		s["ami"] = cty.StringVal("bar")
		resp.PlannedState = cty.ObjectVal(s)
		return
	}

	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"ami": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	code := c.Run([]string{})
	output := done(t)
	if code != 1 {
		t.Errorf("wrong exit code %d; want 1\noutput:\n%s", code, output.Stdout())
	}

	select {
	case <-cancelled:
	default:
		t.Error("command not cancelled")
	}
}

func TestPlan_init_required(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			// Running plan without setting testingOverrides is similar to plan without init
			View: view,
		},
	}

	args := []string{"-no-color"}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("expected error, got success")
	}
	got := output.Stderr()
	if !strings.Contains(got, "farseek init") || !strings.Contains(got, "provider registry.opentofu.org/hashicorp/test: required by this configuration but no version is selected") {
		t.Fatal("wrong error message in output:", got)
	}
}

func TestPlan_replace(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-replace"), td)
	t.Chdir(td)

	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "a",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"hello"}`),
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
						"id": {Type: cty.String, Computed: true},
					},
				},
			},
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}

	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-no-color",
		"-replace", "test_instance.a",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("wrong exit code %d\n\n%s", code, output.Stderr())
	}

	stdout := output.Stdout()
	if got, want := stdout, "1 to add, 0 to change, 1 to destroy"; !strings.Contains(got, want) {
		t.Errorf("wrong plan summary\ngot output:\n%s\n\nwant substring: %s", got, want)
	}
	if got, want := stdout, "test_instance.a will be replaced, as requested"; !strings.Contains(got, want) {
		t.Errorf("missing replace explanation\ngot output:\n%s\n\nwant substring: %s", got, want)
	}
}

// Verify that the parallelism flag allows no more than the desired number of
// concurrent calls to PlanResourceChange.
func TestPlan_parallelism(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("parallelism"), td)
	t.Chdir(td)

	par := 4

	// started is a semaphore that we use to ensure that we never have more
	// than "par" plan operations happening concurrently
	started := make(chan struct{}, par)

	// beginCtx is used as a starting gate to hold back PlanResourceChange
	// calls until we reach the desired concurrency. The cancel func "begin" is
	// called once we reach the desired concurrency, allowing all apply calls
	// to proceed in unison.
	beginCtx, begin := context.WithCancel(context.Background())
	// Ensure cancel is fired regardless of test
	defer begin()

	// Since our mock provider has its own mutex preventing concurrent calls
	// to ApplyResourceChange, we need to use a number of separate providers
	// here. They will all have the same mock implementation function assigned
	// but crucially they will each have their own mutex.
	providerFactories := map[addrs.Provider]providers.Factory{}
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("test%d", i)
		provider := &farseek.MockProvider{}
		provider.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
			ResourceTypes: map[string]providers.Schema{
				name + "_instance": {Block: &configschema.Block{}},
			},
		}
		provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			// If we ever have more than our intended parallelism number of
			// plan operations running concurrently, the semaphore will fail.
			select {
			case started <- struct{}{}:
				defer func() {
					<-started
				}()
			default:
				t.Fatal("too many concurrent apply operations")
			}

			// If we never reach our intended parallelism, the context will
			// never be canceled and the test will time out.
			if len(started) >= par {
				begin()
			}
			<-beginCtx.Done()

			// do some "work"
			// Not required for correctness, but makes it easier to spot a
			// failure when there is more overlap.
			time.Sleep(10 * time.Millisecond)
			return providers.PlanResourceChangeResponse{
				PlannedState: req.ProposedNewState,
			}
		}
		providerFactories[addrs.NewDefaultProvider(name)] = providers.FactoryFixed(provider)
	}
	testingOverrides := &testingOverrides{
		Providers: providerFactories,
	}

	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: testingOverrides,
			View:             view,
		},
	}

	args := []string{
		fmt.Sprintf("-parallelism=%d", par),
	}

	res := c.Run(args)
	output := done(t)
	if res != 0 {
		t.Fatal(output.Stdout())
	}
}

func TestPlan_warnings(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	t.Run("full warnings", func(t *testing.T) {
		p := planWarningsFixtureProvider()
		view, done := testView(t)
		c := &PlanCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		}
		code := c.Run([]string{})
		output := done(t)
		if code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
		}
		// the output should contain 3 warnings (returned by planWarningsFixtureProvider())
		wantWarnings := []string{
			"warning 1",
			"warning 2",
			"warning 3",
		}
		for _, want := range wantWarnings {
			if !strings.Contains(output.Stdout(), want) {
				t.Errorf("missing warning %s", want)
			}
		}
	})

	t.Run("compact warnings", func(t *testing.T) {
		p := planWarningsFixtureProvider()
		view, done := testView(t)
		c := &PlanCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(p),
				View:             view,
			},
		}
		code := c.Run([]string{"-compact-warnings"})
		output := done(t)
		if code != 0 {
			t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
		}
		// the output should contain 3 warnings (returned by planWarningsFixtureProvider())
		// and the message that plan was run with -compact-warnings
		wantWarnings := []string{
			"warning 1",
			"warning 2",
			"warning 3",
			"To see the full warning notes, run Farseek without -compact-warnings.",
		}
		for _, want := range wantWarnings {
			if !strings.Contains(output.Stdout(), want) {
				t.Errorf("missing warning %s", want)
			}
		}
	})
}

func TestPlan_jsonGoldenReference(t *testing.T) {
	// Create a temporary working directory that is empty
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-json",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	checkGoldenReference(t, output, "plan")
}

// planFixtureSchema returns a schema suitable for processing the
// configuration in testdata/plan . This schema should be
// assigned to a mock provider named "test".
func planFixtureSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Optional: true, Computed: true},
						"ami": {Type: cty.String, Optional: true},
					},
					BlockTypes: map[string]*configschema.NestedBlock{
						"network_interface": {
							Nesting: configschema.NestingList,
							Block: configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"device_index": {Type: cty.String, Optional: true},
									"description":  {Type: cty.String, Optional: true},
								},
							},
						},
					},
				},
			},
		},
		DataSources: map[string]providers.Schema{
			"test_data_source": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
						"valid": {
							Type:     cty.Bool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func TestPlan_showSensitiveArg(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-sensitive-output"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-show-sensitive",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad status code: \n%s", output.Stderr())
	}

	if got, want := output.Stdout(), "sensitive    = \"Hello world\""; !strings.Contains(got, want) {
		t.Fatalf("got incorrect output, want %q, got:\n%s", want, got)
	}
}

func TestPlan_withoutShowSensitiveArg(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-sensitive-output"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad status code: \n%s", output.Stderr())
	}

	if got, want := output.Stdout(), "sensitive    = (sensitive value)"; !strings.Contains(got, want) {
		t.Fatalf("got incorrect output, want %q, got:\n%s", want, got)
	}
}

func TestPlan_concise(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{"-concise"}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad status code: \n%s", output.Stderr())
	}

	if got, want := output.Stdout(), "Reading..."; strings.Contains(got, want) {
		t.Fatalf("got incorrect output, want %q, got:\n%s", want, got)
	}
}

// planFixtureProvider returns a mock provider that is configured for basic
// operation with the configuration in testdata/plan. This mock has
// GetSchemaResponse and PlanResourceChangeFn populated, with the plan
// step just passing through the new object proposed by Farseek Core.
func planFixtureProvider() *farseek.MockProvider {
	p := testProvider()
	p.GetProviderSchemaResponse = planFixtureSchema()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		return providers.ReadDataSourceResponse{
			State: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("zzzzz"),
				"valid": cty.BoolVal(true),
			}),
		}
	}
	return p
}

// planVarsFixtureSchema returns a schema suitable for processing the
// configuration in testdata/plan-vars . This schema should be
// assigned to a mock provider named "test".
func planVarsFixtureSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":    {Type: cty.String, Optional: true, Computed: true},
						"value": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
}

// planVarsFixtureProvider returns a mock provider that is configured for basic
// operation with the configuration in testdata/plan-vars. This mock has
// GetSchemaResponse and PlanResourceChangeFn populated, with the plan
// step just passing through the new object proposed by Farseek Core.
func planVarsFixtureProvider() *farseek.MockProvider {
	p := testProvider()
	p.GetProviderSchemaResponse = planVarsFixtureSchema()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			PlannedState: req.ProposedNewState,
		}
	}
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		return providers.ReadDataSourceResponse{
			State: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("zzzzz"),
				"valid": cty.BoolVal(true),
			}),
		}
	}
	return p
}

// planFixtureProvider returns a mock provider that is configured for basic
// operation with the configuration in testdata/plan. This mock has
// GetSchemaResponse and PlanResourceChangeFn populated, returning 3 warnings.
func planWarningsFixtureProvider() *farseek.MockProvider {
	p := testProvider()
	p.GetProviderSchemaResponse = planFixtureSchema()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		return providers.PlanResourceChangeResponse{
			Diagnostics: tfdiags.Diagnostics{
				tfdiags.SimpleWarning("warning 1"),
				tfdiags.SimpleWarning("warning 2"),
				tfdiags.SimpleWarning("warning 3"),
			},
			PlannedState: req.ProposedNewState,
		}
	}
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		return providers.ReadDataSourceResponse{
			State: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("zzzzz"),
				"valid": cty.BoolVal(true),
			}),
		}
	}
	return p
}

const planVarFile = `
foo = "bar"
`

const planVarFileWithDecl = `
foo = "bar"

variable "nope" {
}
`

func TestPlan_farseekDrift(t *testing.T) {
	// Case 1: Nothing changed in Git -> 0 ReadResource calls
	t.Run("nothing-changed", func(t *testing.T) {
		readResources := runFarseekTest(t, "plan-existing-state", nil)
		if len(readResources) > 0 {
			t.Errorf("expected 0 resources to be refreshed if nothing changed in Git, but got %d", len(readResources))
		}
	})

	// Case 2: One resource changed in Git -> 1 ReadResource call for that resource
	t.Run("one-changed", func(t *testing.T) {
		changed := []farseek.DiscoveredResource{{Address: "test_instance.foo"}}
		readResources := runFarseekTest(t, "plan-existing-state", changed)

		// This will fail currently as involuntary targeting from Git is not implemented.
		// If it's implemented, we expect test_instance.foo to have been refreshed.

		if !readResources["test_instance.mock-value"] {
			t.Errorf("expected test_instance.mock-value to be refreshed. Got: %v", readResources)
		}
		if len(readResources) != 1 {
			t.Errorf("expected exactly 1 resource (test_instance.foo) to be refreshed, but got %d: %v", len(readResources), readResources)
		}
	})
}

// runFarseekTest is a helper for creative reuse of existing fixtures in Farseek mode.
func runFarseekTest(t *testing.T, fixture string, changedResources []farseek.DiscoveredResource) map[string]bool {
	// 1. Setup a temporary directory using the fixture
	td := t.TempDir()
	testCopyDir(t, testFixturePath(fixture), td)
	t.Chdir(td)

	// Clean up existing state to ensure state-less mode is active
	os.Remove(filepath.Join(td, "farseek.tfstate"))

	// 2. Create .farseek_sha to trigger Farseek mode
	err := os.WriteFile(filepath.Join(td, ".farseek_sha"), []byte("previous-sha"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Force FarseekMode even with testingOverrides
	t.Setenv("FARSEEK_TEST_FORCE_MODE", "true")

	// 3. Setup Mock Provider and Track Refresh calls
	p := planFixtureProvider()
	readResources := make(map[string]bool)
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		id := "null"
		if !req.PriorState.IsNull() {
			val := req.PriorState
			if val.Type().IsObjectType() && val.Type().HasAttribute("id") {
				idAttr := val.GetAttr("id")
				if !idAttr.IsNull() && idAttr.Type() == cty.String {
					id = idAttr.AsString()
				}
			}
		}
		readResources[req.TypeName+"."+id] = true
		return providers.ReadResourceResponse{
			NewState: req.PriorState,
		}
	}

	// 4. Perform the plan using PlanCommand
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	// 5. Mock Git Discovery logic
	oldDiscovery := farseek.Discovery
	defer func() { farseek.Discovery = oldDiscovery }()
	farseek.Discovery = mockDiscoverer{resources: changedResources}

	args := []string{}
	code := c.Run(args)
	done(t)

	if code != 0 {
		t.Fatalf("bad: %d", code)
	}

	return readResources
}

type mockDiscoverer struct {
	resources []farseek.DiscoveredResource
}

func (m mockDiscoverer) DiscoverChangedResources(dir, baseSHA string, includeUncommitted bool) ([]farseek.DiscoveredResource, error) {
	return m.resources, nil
}

func (m mockDiscoverer) DiscoverAllResources(dir string, includeUncommitted bool) ([]farseek.DiscoveredResource, error) {
	return m.resources, nil
}

func (m mockDiscoverer) GetResourceAttributeFromSHA(dir, sha, filename, address, attribute string) (string, error) {
	return "mock-value", nil
}

func (m mockDiscoverer) GetCurrentSHA(dir string) (string, error) {
	return "mock-sha", nil
}
