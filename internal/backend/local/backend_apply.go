// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package local

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/rafagsiqueira/farseek/internal/addrs"
	"github.com/rafagsiqueira/farseek/internal/backend"
	"github.com/rafagsiqueira/farseek/internal/command/views"

	farseek "github.com/rafagsiqueira/farseek/internal/farseek"
	"github.com/rafagsiqueira/farseek/internal/logging"
	"github.com/rafagsiqueira/farseek/internal/plans"
	"github.com/rafagsiqueira/farseek/internal/states"
	"github.com/rafagsiqueira/farseek/internal/states/statefile"
	"github.com/rafagsiqueira/farseek/internal/states/statemgr"
	"github.com/rafagsiqueira/farseek/internal/tfdiags"
)

// test hook called between plan+apply during opApply
var testHookStopPlanApply func()

const (
	defaultPersistInterval                 = 20 // arbitrary interval that's hopefully a sweet spot
	persistIntervalEnvironmentVariableName = "TF_STATE_PERSIST_INTERVAL"
)

func getEnvAsInt(envName string, defaultValue int) int {
	if val, exists := os.LookupEnv(envName); exists {
		parsedVal, err := strconv.Atoi(val)
		if err == nil {
			return parsedVal
		}
		panic(fmt.Sprintf("Can't parse value '%s' of environment variable '%s'", val, envName))
	}
	return defaultValue
}

func (b *Local) opApply(
	stopCtx context.Context,
	cancelCtx context.Context,
	op *backend.Operation,
	runningOp *backend.RunningOperation) {

	// TEMP: Opt-in support for testing with the new experimental language
	// runtime. Refer to backend_temp_new_runtime.go for more information.
	if experimentalRuntimeEnabled() {
		b.opApplyWithExperimentalRuntime(stopCtx, cancelCtx, op, runningOp)
		return
	}

	log.Printf("[INFO] backend/local: starting Apply operation")

	var diags, moreDiags tfdiags.Diagnostics

	// For the moment we have a bit of a tangled mess of context.Context here, for
	// historical reasons. Hopefully we'll clean this up one day, but here's the
	// guide for now:
	// - ctx is used only for its values, and should be connected to the top-level ctx
	//   from "package main" so that we can obtain telemetry objects, etc from it.
	// - stopCtx is cancelled to trigger a graceful shutdown.
	// - cancelCtx is cancelled for a graceless shutdown.
	ctx := context.WithoutCancel(stopCtx)

	// If we have a nil module at this point, then set it to an empty tree
	// to avoid any potential crashes.
	if op.PlanFile == nil && op.PlanMode != plans.DestroyMode && !op.HasConfig() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No configuration files",
			"Apply requires configuration to be present. Applying without a configuration "+
				"would mark everything for destruction, which is normally not what is desired. "+
				"If you would like to destroy everything, run 'farseek destroy' instead.",
		))
		op.ReportResult(runningOp, diags)
		return
	}

	stateHook := new(StateHook)
	if !op.FarseekMode {
		op.Hooks = append(op.Hooks, stateHook)
	}

	// Get our context
	lr, _, opState, contextDiags := b.localRun(ctx, op)
	diags = diags.Append(contextDiags)
	if contextDiags.HasErrors() {
		op.ReportResult(runningOp, diags)
		return
	}
	// the state was locked during successful context creation; unlock the state
	// when the operation completes
	defer func() {
		diags := op.StateLocker.Unlock()
		if diags.HasErrors() {
			op.View.Diagnostics(diags)
			runningOp.Result = backend.OperationFailure
		}
	}()

	if op.FarseekMode {
		log.Printf("[DEBUG] backend/local: Farseek stateless mode enabled. Discovered: %d", len(op.DiscoveredResources))
		// We explicitly ignore the local state to ensure it's truly stateless.
		lr.InputState = states.NewState()

		// Inject relevant discovered resources into the state.
		for _, dr := range op.DiscoveredResources {
			if dr.IsNew {
				// Truly new resources should NOT be in the state, so Farseek plans to create them.
				log.Printf("[DEBUG] backend/local: Farseek skipping injection for new resource %s", dr.Address)
				continue
			}

			// For resources that existed in history, we inject them into state so Farseek refreshes/destroys them.
			addr, diags := addrs.ParseAbsResourceInstanceStr(dr.Address)
			if diags.HasErrors() {
				log.Printf("[WARN] backend/local: Farseek failed to parse address %s: %s", dr.Address, diags.Err())
				continue
			}

			log.Printf("[DEBUG] backend/local: Farseek injecting resource %s into input state", addr)
			mod := lr.InputState.EnsureModule(addr.Module)
			providerAddr := addrs.AbsProviderConfig{
				Module:   addr.Module.Module(),
				Provider: addrs.ImpliedProviderForUnqualifiedType(addr.Resource.Resource.ImpliedProvider()),
			}

			// Try to recover attributes (id/name) from history to help the provider identify the resource.
			jsonAttrs := "{}"
			if op.FarseekBaseSHA != "" {
				nameVal, _ := farseek.Discovery.GetResourceAttributeFromSHA(op.ConfigDir, op.FarseekBaseSHA, dr.Filename, dr.Address, "name")
				idVal, _ := farseek.Discovery.GetResourceAttributeFromSHA(op.ConfigDir, op.FarseekBaseSHA, dr.Filename, dr.Address, "id")

				id := idVal
				if id == "" {
					id = nameVal
				}

				if id != "" {
					if nameVal != "" {
						jsonAttrs = fmt.Sprintf(`{"id":%q, "name":%q}`, id, nameVal)
					} else {
						jsonAttrs = fmt.Sprintf(`{"id":%q}`, id)
					}
				}
			}

			src := &states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(jsonAttrs),
			}
			mod.SetResourceInstanceCurrent(addr.Resource, src, providerAddr, addrs.NoKey)
		}
	}
	runningOp.State = lr.InputState

	schemas, moreDiags := lr.Core.Schemas(ctx, lr.Config, lr.InputState)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		op.ReportResult(runningOp, diags)
		return
	}
	// stateHook uses schemas for when it periodically persists state to the
	// persistent storage backend.
	stateHook.Schemas = schemas
	persistInterval := getEnvAsInt(persistIntervalEnvironmentVariableName, defaultPersistInterval)
	if persistInterval < defaultPersistInterval {
		panic(fmt.Sprintf("Can't use value lower than %d for env variable %s, got %d",
			defaultPersistInterval, persistIntervalEnvironmentVariableName, persistInterval))
	}
	stateHook.PersistInterval = time.Duration(persistInterval) * time.Second

	var plan *plans.Plan
	// If we weren't given a plan, then we refresh/plan
	if op.PlanFile == nil {

		// Perform the plan
		log.Printf("[INFO] backend/local: apply calling Plan")
		plan, moreDiags = lr.Core.Plan(ctx, lr.Config, lr.InputState, lr.PlanOpts)

		// FarseekMode: Suppress updates to attributes not present in the configuration
		b.filterPlanChanges(ctx, op, lr, plan)

		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			// If Farseek Core generated a partial plan despite the errors
			// then we'll make the best effort to render it. Farseek Core
			// promises that if it returns a non-nil plan along with errors
			// then the plan won't necessarily contain all the needed
			// actions but that any it does include will be properly-formed.
			// plan.Errored will be true in this case, which our plan
			// renderer can rely on to tailor its messaging.
			if plan != nil && (len(plan.Changes.Resources) != 0 || len(plan.Changes.Outputs) != 0) {
				op.View.Plan(plan, schemas)
			}
			op.ReportResult(runningOp, diags)
			return
		}

		trivialPlan := !plan.CanApply()
		hasUI := op.UIOut != nil && op.UIIn != nil
		mustConfirm := hasUI && !op.AutoApprove && !trivialPlan
		op.View.Plan(plan, schemas)

		if testHookStopPlanApply != nil {
			testHookStopPlanApply()
		}

		// Check if we've been stopped before going through confirmation, or
		// skipping confirmation in the case of -auto-approve.
		// This can currently happen if a single stop request was received
		// during the final batch of resource plan calls, so no operations were
		// forced to abort, and no errors were returned from Plan.
		if stopCtx.Err() != nil {
			diags = diags.Append(errors.New("execution halted"))
			runningOp.Result = backend.OperationFailure
			op.ReportResult(runningOp, diags)
			return
		}

		if mustConfirm {
			var desc, query string
			switch op.PlanMode {
			case plans.DestroyMode:
				if op.Workspace != "default" {
					query = "Do you really want to destroy all resources in workspace \"" + op.Workspace + "\"?"
				} else {
					query = "Do you really want to destroy all resources?"
				}
				desc = "Farseek will destroy all your managed infrastructure, as shown above.\n" +
					"There is no undo. Only 'yes' will be accepted to confirm."
			case plans.RefreshOnlyMode:
				if op.Workspace != "default" {
					query = "Would you like to update the Farseek state for \"" + op.Workspace + "\" to reflect these detected changes?"
				} else {
					query = "Would you like to update the Farseek state to reflect these detected changes?"
				}
				desc = "Farseek will write these changes to the state without modifying any real infrastructure.\n" +
					"There is no undo. Only 'yes' will be accepted to confirm."
			default:
				if op.Workspace != "default" {
					query = "Do you want to perform these actions in workspace \"" + op.Workspace + "\"?"
				} else {
					query = "Do you want to perform these actions?"
				}
				desc = "Farseek will perform the actions described above.\n" +
					"Only 'yes' will be accepted to approve."
			}

			// We'll show any accumulated warnings before we display the prompt,
			// so the user can consider them when deciding how to answer.
			if len(diags) > 0 {
				op.View.Diagnostics(diags)
				diags = nil // reset so we won't show the same diagnostics again later
			}

			v, err := op.UIIn.Input(stopCtx, &farseek.InputOpts{
				Id:          "approve",
				Query:       "\n" + query,
				Description: desc,
			})
			if err != nil {
				diags = diags.Append(fmt.Errorf("error asking for approval: %w", err))
				op.ReportResult(runningOp, diags)
				return
			}
			if v != "yes" {
				op.View.Cancelled(op.PlanMode)
				runningOp.Result = backend.OperationFailure
				return
			}
		} else {
			// If we didn't ask for confirmation from the user, and they have
			// included any failing checks in their configuration, then they
			// will see a very confusing output after the apply operation
			// completes. This is because all the diagnostics from the plan
			// operation will now be shown alongside the diagnostics from the
			// apply operation. For check diagnostics, the plan output is
			// irrelevant and simple noise after the same set of checks have
			// been executed again during the apply stage. As such, we are going
			// to remove all diagnostics marked as check diagnostics at this
			// stage, so we will only show the user the check results from the
			// apply operation.
			//
			// Note, if we did ask for approval then we would have displayed the
			// plan check results at that point which is useful as the user can
			// use them to make a decision about whether to apply the changes.
			// It's just that if we didn't ask for approval then showing the
			// user the checks from the plan alongside the checks from the apply
			// is needlessly confusing.
			var filteredDiags tfdiags.Diagnostics
			for _, diag := range diags {
				if rule, ok := addrs.DiagnosticOriginatesFromCheckRule(diag); ok && rule.Container.CheckableKind() == addrs.CheckableCheck {
					continue
				}
				filteredDiags = filteredDiags.Append(diag)
			}
			diags = filteredDiags
		}
	} else {
		plan = lr.Plan
		if plan.Errored {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Cannot apply incomplete plan",
				"Farseek encountered an error when generating this plan, so it cannot be applied.",
			))
			op.ReportResult(runningOp, diags)
			return
		}
		for _, change := range plan.Changes.Resources {
			if change.Action != plans.NoOp {
				op.View.PlannedChange(change)
			}
		}
	}

	// Set up our hook for continuous state updates
	stateHook.StateMgr = opState

	// Start to apply in a goroutine so that we can be interrupted.
	var applyState *states.State
	var applyDiags tfdiags.Diagnostics
	doneCh := make(chan struct{})
	panicHandler := logging.PanicHandlerWithTraceFn()
	go func() {
		defer panicHandler()
		defer close(doneCh)
		log.Printf("[INFO] backend/local: apply calling Apply")
		applyState, applyDiags = lr.Core.Apply(ctx, plan, lr.Config, lr.ApplyOpts)
	}()

	if b.opWait(doneCh, stopCtx, cancelCtx, lr.Core, opState, op.View) {
		return
	}
	diags = diags.Append(applyDiags)

	// Even on error with an empty state, the state value should not be nil.
	// Return early here to prevent corrupting any existing state.
	if diags.HasErrors() && applyState == nil {
		log.Printf("[ERROR] backend/local: apply returned nil state")
		op.ReportResult(runningOp, diags)
		return
	}

	// Store the final state
	runningOp.State = applyState
	if true {
		err := statemgr.WriteAndPersist(context.TODO(), opState, applyState, schemas)
		if err != nil {
			// Export the state file from the state manager and assign the new
			// state. This is needed to preserve the existing serial and lineage.
			stateFile := statemgr.Export(opState)
			if stateFile == nil {
				stateFile = &statefile.File{}
			}
			stateFile.State = applyState

			diags = diags.Append(b.backupStateForError(stateFile, err, op.View))
			op.ReportResult(runningOp, diags)
			return
		}
	}

	if applyDiags.HasErrors() {
		op.ReportResult(runningOp, diags)
		return
	}

	// If we've accumulated any warnings along the way then we'll show them
	// here just before we show the summary and next steps. If we encountered
	// errors then we would've returned early at some other point above.
	op.View.Diagnostics(diags)
}

// backupStateForError is called in a scenario where we're unable to persist the
// state for some reason, and will attempt to save a backup copy of the state
// to local disk to help the user recover. This is a "last ditch effort" sort
// of thing, so we really don't want to end up in this codepath; we should do
// everything we possibly can to get the state saved _somewhere_.
func (b *Local) backupStateForError(stateFile *statefile.File, err error, view views.Operation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Failed to save state",
		fmt.Sprintf("Error saving state: %s", err),
	))

	local := statemgr.NewFilesystem("errored.tfstate", b.encryption)
	writeErr := local.WriteStateForMigration(stateFile, true)
	if writeErr != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to create local state file",
			fmt.Sprintf("Error creating local state file for recovery: %s", writeErr),
		))

		// To avoid leaving the user with no state at all, our last resort
		// is to print the JSON state out onto the terminal. This is an awful
		// UX, so we should definitely avoid doing this if at all possible,
		// but at least the user has _some_ path to recover if we end up
		// here for some reason.
		if dumpErr := view.EmergencyDumpState(stateFile, b.encryption); dumpErr != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to serialize state",
				fmt.Sprintf(stateWriteFatalErrorFmt, dumpErr),
			))
		}

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to persist state to backend",
			stateWriteConsoleFallbackError,
		))
		return diags
	}

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Failed to persist state to backend",
		stateWriteBackedUpError,
	))

	return diags
}

const stateWriteBackedUpError = `The error shown above has prevented Farseek from writing the updated state to the configured backend. To allow for recovery, the state has been written to the file "errored.tfstate" in the current working directory.

Running "farseek apply" again at this point will create a forked state, making it harder to recover.

To retry writing this state, use the following command:
    farseek state push errored.tfstate
`

const stateWriteConsoleFallbackError = `The errors shown above prevented Farseek from writing the updated state to
the configured backend and from creating a local backup file. As a fallback,
the raw state data is printed above as a JSON object.

To retry writing this state, copy the state data (from the first { to the last } inclusive) and save it into a local file called errored.tfstate, then run the following command:
    farseek state push errored.tfstate
`

const stateWriteFatalErrorFmt = `Failed to save state after apply.

Error serializing state: %s

A catastrophic error has prevented Farseek from persisting the state file or creating a backup. Unfortunately this means that the record of any resources created during this apply has been lost, and such resources may exist outside of Farseek's management.

For resources that support import, it is possible to recover by manually importing each resource using its id from the target system.

This is a serious bug in Farseek and should be reported.
`
