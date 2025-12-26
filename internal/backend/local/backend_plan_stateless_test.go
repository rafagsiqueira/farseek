package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/rafagsiqueira/farseek/internal/addrs"
	"github.com/rafagsiqueira/farseek/internal/backend"
	"github.com/rafagsiqueira/farseek/internal/configs/configschema"
	"github.com/rafagsiqueira/farseek/internal/plans"
	"github.com/rafagsiqueira/farseek/internal/providers"
)

func TestLocal_planFarseekMode_ChecksExistence(t *testing.T) {
	// 1. Setup specific configuration for this test
	td := t.TempDir()
	tfConfig := `
resource "test_instance" "foo" {
  name = "existing_resource"
}
`
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(tfConfig), 0644); err != nil {
		t.Fatal(err)
	}

	b := TestLocal(t)
	schema := planFixtureSchema()
	// Add 'name' attribute to schema so config is valid
	schema.ResourceTypes["test_instance"].Block.Attributes["name"] = &configschema.Attribute{
		Type: cty.String, Optional: true,
	}

	// 2. Mock Provider with Import Support
	p := TestLocalProvider(t, b, "test", schema)

	p.ImportResourceStateFn = func(req providers.ImportResourceStateRequest) (resp providers.ImportResourceStateResponse) {
		if req.ID == "existing_resource" {
			return providers.ImportResourceStateResponse{
				ImportedResources: []providers.ImportedResource{
					{
						TypeName: req.TypeName,
						State: cty.ObjectVal(map[string]cty.Value{
							"name": cty.StringVal("existing_resource"),
							"ami":  cty.NullVal(cty.String),
							"network_interface": cty.ListValEmpty(cty.Object(map[string]cty.Type{
								"device_index": cty.Number,
								"description":  cty.String,
							})),
						}),
					},
				},
			}
		}
		return providers.ImportResourceStateResponse{}
	}

	// 3. Setup Operation
	op, done := testOperationPlan(t, td)
	op.PlanRefresh = true // Enable refresh
	op.FarseekMode = true

	// Target the specific resource
	targetAddr, _ := addrs.ParseAbsResourceInstanceStr("test_instance.foo")
	op.Targets = []addrs.Targetable{targetAddr}

	// Setup plan output to verification
	outDir := t.TempDir()
	planPath := filepath.Join(outDir, "plan.tfplan")
	op.PlanOutPath = planPath
	cfg := cty.ObjectVal(map[string]cty.Value{
		"path": cty.StringVal(b.StatePath),
	})
	cfgRaw, err := plans.NewDynamicValue(cfg, cfg.Type())
	if err != nil {
		t.Fatal(err)
	}
	op.PlanOutBackend = &plans.Backend{
		Type:   "local",
		Config: cfgRaw,
	}

	// 4. Run Plan
	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	<-run.Done() // Operation returns a separate wait channel

	output := done(t) // detailed output checking if needed

	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed. Output:\n%s", output.Stderr())
	}

	// 5. Assertions
	if !p.ImportResourceStateCalled {
		t.Fatal("ImportResourceState was NOT called! The check-first logic failed to trigger.")
	}

	// Read the plan back
	plan := testReadPlan(t, planPath)

	// Verify the plan does NOT contain a Create
	changes := plan.Changes
	resChanges := changes.ResourceInstance(targetAddr)
	if resChanges == nil {
		// No changes is also acceptable (matches state exactly)
	} else {
		if resChanges.Action == plans.Create {
			t.Fatalf("Plan action is CREATE, but should be NoOp or Update because resource exists! Action: %s", resChanges.Action)
		}
	}
}

func TestLocal_planFarseekMode_UpdatesOnlyConfigured(t *testing.T) {
	// 1. Setup specific configuration for this test
	// Config ONLY has 'name'. 'ami' is missing.
	td := t.TempDir()
	tfConfig := `
resource "test_instance" "foo" {
  name = "existing_resource"
  lifecycle {
    create_before_destroy = true
  }
}
`
	if err := os.WriteFile(filepath.Join(td, "main.tf"), []byte(tfConfig), 0644); err != nil {
		t.Fatal(err)
	}

	b := TestLocal(t)
	schema := planFixtureSchema()
	// Add 'name' attribute to schema so config is valid
	schema.ResourceTypes["test_instance"].Block.Attributes["name"] = &configschema.Attribute{
		Type: cty.String, Optional: true,
	}

	// 2. Mock Provider with Import Support
	p := TestLocalProvider(t, b, "test", schema)

	p.ImportResourceStateFn = func(req providers.ImportResourceStateRequest) (resp providers.ImportResourceStateResponse) {
		if req.ID == "existing_resource" {
			return providers.ImportResourceStateResponse{
				ImportedResources: []providers.ImportedResource{
					{
						TypeName: req.TypeName,
						State: cty.ObjectVal(map[string]cty.Value{
							"name": cty.StringVal("existing_resource"),
							"ami":  cty.StringVal("ami-12345"), // State has value!
							"network_interface": cty.ListValEmpty(cty.Object(map[string]cty.Type{
								"device_index": cty.Number,
								"description":  cty.String,
							})),
						}),
					},
				},
			}
		}
		return providers.ImportResourceStateResponse{}
	}

	// 3. Setup Operation
	op, done := testOperationPlan(t, td)
	op.PlanRefresh = true
	op.FarseekMode = true

	// Target the specific resource
	targetAddr, _ := addrs.ParseAbsResourceInstanceStr("test_instance.foo")
	op.Targets = []addrs.Targetable{targetAddr}

	// Setup plan output
	outDir := t.TempDir()
	planPath := filepath.Join(outDir, "plan.tfplan")
	op.PlanOutPath = planPath
	cfg := cty.ObjectVal(map[string]cty.Value{
		"path": cty.StringVal(b.StatePath),
	})
	cfgRaw, err := plans.NewDynamicValue(cfg, cfg.Type())
	if err != nil {
		t.Fatal(err)
	}
	op.PlanOutBackend = &plans.Backend{
		Type:   "local",
		Config: cfgRaw,
	}

	// 4. Run Plan
	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	<-run.Done()

	output := done(t)

	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed. Output:\n%s", output.Stderr())
	}

	// Read the plan back
	plan := testReadPlan(t, planPath)

	// Verify the plan does NOT contain Update for AMI
	// Since 'ami' is in state ("ami-12345") but NOT in config (null),
	// standard Terraform behavior would be an UPDATE (ami: "ami-12345" -> null).
	// usage of 'FarseekMode' should suppress this because 'ami' is not in config.
	changes := plan.Changes
	resChanges := changes.ResourceInstance(targetAddr)

	if resChanges != nil {
		// If check-first logic works, it's not Create.
		// If suppression works, it should be NoOp (nil) or eventually empty update?
		// Note: If ALL changes are suppressed, resChanges might be nil if the plan purely consists of NoOps?
		// Actually if it's NoOp, resChanges might be non-nil but Action is NoOp.
		if resChanges.Action == plans.Update {
			// Check if AMI is being changed
			// This assertion is tricky without inspecting the specialized change object,
			// but if the ONLY change was AMI, then Action should have been NoOp if suppressed.
			t.Fatalf("Plan action is UPDATE, expected NoOp (update suppression). Plan:\n%s", output.Stdout())
		}
		if resChanges.Action == plans.Create {
			t.Fatalf("Plan action is CREATE, expected NoOp (resource exists).")
		}
	}
}
