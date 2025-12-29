// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package views

import (
	"context"
	"fmt"

	"github.com/rafagsiqueira/farseek/internal/command/arguments"
	"github.com/rafagsiqueira/farseek/internal/command/jsonconfig"
	"github.com/rafagsiqueira/farseek/internal/command/jsonformat"
	"github.com/rafagsiqueira/farseek/internal/command/jsonplan"
	"github.com/rafagsiqueira/farseek/internal/command/jsonprovider"
	"github.com/rafagsiqueira/farseek/internal/command/jsonstate"
	"github.com/rafagsiqueira/farseek/internal/configs"
	farseek "github.com/rafagsiqueira/farseek/internal/farseek"
	"github.com/rafagsiqueira/farseek/internal/plans"
	"github.com/rafagsiqueira/farseek/internal/states/statefile"
	"github.com/rafagsiqueira/farseek/internal/tfdiags"
)

type Show interface {
	// DisplayState renders the given state snapshot, returning a status code for "farseek show" to return.
	DisplayState(ctx context.Context, stateFile *statefile.File, schemas *farseek.Schemas) int

	// DisplayPlan renders the given plan, returning a status code for "farseek show" to return.
	//
	DisplayPlan(ctx context.Context, plan *plans.Plan, config *configs.Config, priorStateFile *statefile.File, schemas *farseek.Schemas) int

	// DisplayConfig renders the given configuration, returning a status code for "farseek show" to return.
	DisplayConfig(config *configs.Config, schemas *farseek.Schemas) int

	// DisplaySingleModule renders just one module, in a format that's a subset
	// of that used by [Show.DisplayConfig] which we can produce without
	// schema or child module information.
	DisplaySingleModule(module *configs.Module) int

	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewShow(vt arguments.ViewType, view *View) Show {
	switch vt {
	case arguments.ViewJSON:
		return &ShowJSON{view: view}
	case arguments.ViewHuman:
		return &ShowHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type ShowHuman struct {
	view *View
}

var _ Show = (*ShowHuman)(nil)

func (v *ShowHuman) DisplayState(_ context.Context, stateFile *statefile.File, schemas *farseek.Schemas) int {
	renderer := jsonformat.Renderer{
		Colorize:            v.view.colorize,
		Streams:             v.view.streams,
		RunningInAutomation: v.view.runningInAutomation,
		ShowSensitive:       v.view.showSensitive,
	}

	if stateFile == nil {
		v.view.streams.Println("No state.")
		return 0
	}

	root, outputs, err := jsonstate.MarshalForRenderer(stateFile, schemas)
	if err != nil {
		v.view.streams.Eprintf("Failed to marshal state to json: %s", err)
		return 1
	}

	jstate := jsonformat.State{
		StateFormatVersion:    jsonstate.FormatVersion,
		ProviderFormatVersion: jsonprovider.FormatVersion,
		RootModule:            root,
		RootModuleOutputs:     outputs,
		ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
	}

	renderer.RenderHumanState(jstate)
	return 0
}

func (v *ShowHuman) DisplayPlan(_ context.Context, plan *plans.Plan, config *configs.Config, priorStateFile *statefile.File, schemas *farseek.Schemas) int {
	renderer := jsonformat.Renderer{
		Colorize:            v.view.colorize,
		Streams:             v.view.streams,
		RunningInAutomation: v.view.runningInAutomation,
		ShowSensitive:       v.view.showSensitive,
	}

	// Prefer to display a pre-built JSON plan, if we got one; then, fall back
	// to building one ourselves.
	if plan != nil {
		outputs, changed, drift, attrs, err := jsonplan.MarshalForRenderer(plan, schemas)
		if err != nil {
			v.view.streams.Eprintf("Failed to marshal plan to json: %s", err)
			return 1
		}

		jplan := jsonformat.Plan{
			PlanFormatVersion:     jsonplan.FormatVersion,
			ProviderFormatVersion: jsonprovider.FormatVersion,
			OutputChanges:         outputs,
			ResourceChanges:       changed,
			ResourceDrift:         drift,
			ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
			RelevantAttributes:    attrs,
		}

		var opts []plans.Quality
		if !plan.CanApply() {
			opts = append(opts, plans.NoChanges)
		}
		if plan.Errored {
			opts = append(opts, plans.Errored)
		}

		renderer.RenderHumanPlan(jplan, plan.UIMode, opts...)
	} else {
		v.view.streams.Println("No plan.")
	}
	return 0
}

func (v *ShowHuman) DisplayConfig(config *configs.Config, schemas *farseek.Schemas) int {
	// The human view should never be called for configuration display
	// since we require -json for -config
	v.view.streams.Eprintf("Internal error: human view should not be used for configuration display")
	return 1
}

func (v *ShowHuman) DisplaySingleModule(_ *configs.Module) int {
	// The human view should never be called for module display
	// since we require -json for -module=DIR.
	v.view.streams.Eprintf("Internal error: human view should not be used for module display")
	return 1
}

func (v *ShowHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

type ShowJSON struct {
	view *View
}

var _ Show = (*ShowJSON)(nil)

func (v *ShowJSON) DisplayState(_ context.Context, stateFile *statefile.File, schemas *farseek.Schemas) int {
	jsonState, err := jsonstate.Marshal(stateFile, schemas)
	if err != nil {
		v.view.streams.Eprintf("Failed to marshal state to json: %s", err)
		return 1
	}
	v.view.streams.Println(string(jsonState))
	return 0
}

func (v *ShowJSON) DisplayPlan(_ context.Context, plan *plans.Plan, config *configs.Config, priorStateFile *statefile.File, schemas *farseek.Schemas) int {
	// Prefer to display a pre-built JSON plan, if we got one; then, fall back
	// to building one ourselves.
	if plan != nil {
		planJSON, err := jsonplan.Marshal(config, plan, priorStateFile, schemas)

		if err != nil {
			v.view.streams.Eprintf("Failed to marshal plan to json: %s", err)
			return 1
		}
		v.view.streams.Println(string(planJSON))
	} else {
		// Should not get here because at least one of the two plan arguments
		// should be present, but we'll tolerate this by just returning an
		// empty JSON object.
		v.view.streams.Println("{}")
	}
	return 0
}

func (v *ShowJSON) DisplayConfig(config *configs.Config, schemas *farseek.Schemas) int {
	configJSON, err := jsonconfig.Marshal(config, schemas)
	if err != nil {
		v.view.streams.Eprintf("Failed to marshal configuration to JSON: %s", err)
		return 1
	}
	v.view.streams.Println(string(configJSON))
	return 0
}

func (v *ShowJSON) DisplaySingleModule(module *configs.Module) int {
	moduleJSON, err := jsonconfig.MarshalSingleModule(module)
	if err != nil {
		v.view.streams.Eprintf("Failed to marshal module contents to JSON: %s", err)
		return 1
	}
	v.view.streams.Println(string(moduleJSON))
	return 0
}

// Diagnostics should only be called if show cannot be executed.
// In this case, we choose to render human-readable diagnostic output,
// primarily for backwards compatibility.
func (v *ShowJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
