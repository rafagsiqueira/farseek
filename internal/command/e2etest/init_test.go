// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package e2etest

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rafagsiqueira/farseek/internal/e2e"
)

func TestInitProvidersInternal(t *testing.T) {
	t.Parallel()

	// This test should _not_ reach out anywhere because the "terraform"
	// provider is internal to the core farseek binary.

	t.Run("output in human readable format", func(t *testing.T) {
		fixturePath := filepath.Join("testdata", "tf-provider")
		tf := e2e.NewBinary(t, farseekBin, fixturePath)

		stdout, stderr, err := tf.Run("init")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if stderr != "" {
			t.Errorf("unexpected stderr output:\n%s", stderr)
		}

		if !strings.Contains(stdout, "Farseek has been successfully initialized!") {
			t.Errorf("success message is missing from output:\n%s", stdout)
		}

		if strings.Contains(stdout, "Installing hashicorp/terraform") {
			// Shouldn't have downloaded anything with this config, because the
			// provider is built in.
			t.Errorf("provider download message appeared in output:\n%s", stdout)
		}

		if strings.Contains(stdout, "Installing terraform.io/builtin/terraform") {
			// Shouldn't have downloaded anything with this config, because the
			// provider is built in.
			t.Errorf("provider download message appeared in output:\n%s", stdout)
		}
	})

	t.Run("output in machine readable format", func(t *testing.T) {
		fixturePath := filepath.Join("testdata", "tf-provider")
		tf := e2e.NewBinary(t, farseekBin, fixturePath)

		stdout, stderr, err := tf.Run("init", "-json")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if stderr != "" {
			t.Errorf("unexpected stderr output:\n%s", stderr)
		}

		// we can not check timestamp, so the sub string is not a valid json object
		if !strings.Contains(stdout, `{"@level":"info","@message":"Farseek has been successfully initialized!","@module":"farseek.ui"`) {
			t.Errorf("success message is missing from output:\n%s", stdout)
		}

		if strings.Contains(stdout, "Installing hashicorp/terraform") {
			// Shouldn't have downloaded anything with this config, because the
			// provider is built in.
			t.Errorf("provider download message appeared in output:\n%s", stdout)
		}

		if strings.Contains(stdout, "Installing terraform.io/builtin/terraform") {
			// Shouldn't have downloaded anything with this config, because the
			// provider is built in.
			t.Errorf("provider download message appeared in output:\n%s", stdout)
		}
	})

}

func TestInitProvidersLocalOnly(t *testing.T) {
	t.Parallel()

	// This test should not reach out to the network if it is behaving as
	// intended. If it _does_ try to access an upstream registry and encounter
	// an error doing so then that's a legitimate test failure that should be
	// fixed. (If it incorrectly reaches out anywhere then it's likely to be
	// to the host "example.com", which is the placeholder domain we use in
	// the test fixture.)

	t.Run("output in human readable format", func(t *testing.T) {
		fixturePath := filepath.Join("testdata", "local-only-provider")
		tf := e2e.NewBinary(t, farseekBin, fixturePath)
		// If you run this test on a workstation with a plugin-cache directory
		// configured, it will leave a bad directory behind and farseek init will
		// not work until you remove it.
		//
		// To avoid this, we will  "zero out" any existing cli config file.
		tf.AddEnv("TF_CLI_CONFIG_FILE=")

		// Our fixture dir has a generic os_arch dir, which we need to customize
		// to the actual OS/arch where this test is running in order to get the
		// desired result.
		fixtMachineDir := tf.Path("terraform.d/plugins/example.com/awesomecorp/happycloud/1.2.0/os_arch")
		wantMachineDir := tf.Path("terraform.d/plugins/example.com/awesomecorp/happycloud/1.2.0/", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
		err := os.Rename(fixtMachineDir, wantMachineDir)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		stdout, stderr, err := tf.Run("init")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if stderr != "" {
			t.Errorf("unexpected stderr output:\n%s", stderr)
		}

		if !strings.Contains(stdout, "Farseek has been successfully initialized!") {
			t.Errorf("success message is missing from output:\n%s", stdout)
		}

		if !strings.Contains(stdout, "- Installing example.com/awesomecorp/happycloud v1.2.0") {
			t.Errorf("provider download message is missing from output:\n%s", stdout)
			t.Logf("(this can happen if you have a conflicting copy of the plugin in one of the global plugin search dirs)")
		}
	})

	t.Run("output in machine readable format", func(t *testing.T) {
		fixturePath := filepath.Join("testdata", "local-only-provider")
		tf := e2e.NewBinary(t, farseekBin, fixturePath)
		// If you run this test on a workstation with a plugin-cache directory
		// configured, it will leave a bad directory behind and farseek init will
		// not work until you remove it.
		//
		// To avoid this, we will  "zero out" any existing cli config file.
		tf.AddEnv("TF_CLI_CONFIG_FILE=")

		// Our fixture dir has a generic os_arch dir, which we need to customize
		// to the actual OS/arch where this test is running in order to get the
		// desired result.
		fixtMachineDir := tf.Path("terraform.d/plugins/example.com/awesomecorp/happycloud/1.2.0/os_arch")
		wantMachineDir := tf.Path("terraform.d/plugins/example.com/awesomecorp/happycloud/1.2.0/", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
		err := os.Rename(fixtMachineDir, wantMachineDir)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		stdout, stderr, err := tf.Run("init", "-json")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if stderr != "" {
			t.Errorf("unexpected stderr output:\n%s", stderr)
		}

		// we can not check timestamp, so the sub string is not a valid json object
		if !strings.Contains(stdout, `{"@level":"info","@message":"Farseek has been successfully initialized!","@module":"farseek.ui"`) {
			t.Errorf("success message is missing from output:\n%s", stdout)
		}

		if !strings.Contains(stdout, `{"@level":"info","@message":"- Installing example.com/awesomecorp/happycloud v1.2.0...","@module":"farseek.ui"`) {
			t.Errorf("provider download message is missing from output:\n%s", stdout)
			t.Logf("(this can happen if you have a conflicting copy of the plugin in one of the global plugin search dirs)")
		}
	})

}

func TestInitProvidersCustomMethod(t *testing.T) {
	t.Parallel()

	// This test should not reach out to the network if it is behaving as
	// intended. If it _does_ try to access an upstream registry and encounter
	// an error doing so then that's a legitimate test failure that should be
	// fixed. (If it incorrectly reaches out anywhere then it's likely to be
	// to the host "example.com", which is the placeholder domain we use in
	// the test fixture.)

	for _, configFile := range []string{"cliconfig.tfrc", "cliconfig.tfrc.json"} {
		t.Run(configFile, func(t *testing.T) {
			fixturePath := filepath.Join("testdata", "custom-provider-install-method")
			tf := e2e.NewBinary(t, farseekBin, fixturePath)

			// Our fixture dir has a generic os_arch dir, which we need to customize
			// to the actual OS/arch where this test is running in order to get the
			// desired result.
			fixtMachineDir := tf.Path("fs-mirror/example.com/awesomecorp/happycloud/1.2.0/os_arch")
			wantMachineDir := tf.Path("fs-mirror/example.com/awesomecorp/happycloud/1.2.0/", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
			err := os.Rename(fixtMachineDir, wantMachineDir)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			// We'll use a local CLI configuration file taken from our fixture
			// directory so we can force a custom installation method config.
			tf.AddEnv("TF_CLI_CONFIG_FILE=" + tf.Path(configFile))

			stdout, stderr, err := tf.Run("init")
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}

			if stderr != "" {
				t.Errorf("unexpected stderr output:\n%s", stderr)
			}

			if !strings.Contains(stdout, "Farseek has been successfully initialized!") {
				t.Errorf("success message is missing from output:\n%s", stdout)
			}

			if !strings.Contains(stdout, "- Installing example.com/awesomecorp/happycloud v1.2.0") {
				t.Errorf("provider download message is missing from output:\n%s", stdout)
			}
		})
	}
}

func TestInitProviders_pluginCache(t *testing.T) {
	t.Parallel()

	// This test reaches out to registry.opentofu.org to access plugin
	// metadata, and download the null plugin, though the template plugin
	// should come from local cache.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "plugin-cache")
	tf := e2e.NewBinary(t, farseekBin, fixturePath)

	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		// template v2.1.0 and null v2.1.0 which are used in this test
		// are not available for darwin_arm64.
		t.Skip("template v2.1.0 and null v2.1.0 are not available for darwin_arm64")
	}

	// Our fixture dir has a generic os_arch dir, which we need to customize
	// to the actual OS/arch where this test is running in order to get the
	// desired result.
	fixtMachineDir := tf.Path("cache/registry.opentofu.org/hashicorp/template/2.1.0/os_arch")
	wantMachineDir := tf.Path("cache/registry.opentofu.org/hashicorp/template/2.1.0/", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
	err := os.Rename(fixtMachineDir, wantMachineDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	extension := ""
	if runtime.GOOS == "windows" {
		extension = ".exe"

		// Fix EXE path
		target := path.Join(wantMachineDir, "terraform-provider-template_v2.1.0_x4")
		err := os.Rename(target, target+extension)
		if err != nil {
			t.Fatal(err)
		}

		// TODO add .exe entry to lockfile
		t.Skip()
	}

	// convert the slashes if building for windows.
	p := filepath.FromSlash("./cache")
	tf.AddEnv("TF_PLUGIN_CACHE_DIR=" + p)
	sout, serr, err := tf.Run("init")
	if err != nil {
		t.Errorf("unexpected error: %s\nstdout: %s\nstderr: %s", err, sout, serr)
	}

	path := filepath.FromSlash(fmt.Sprintf(".terraform/providers/registry.opentofu.org/hashicorp/template/2.1.0/%s_%s/terraform-provider-template_v2.1.0_x4", runtime.GOOS, runtime.GOARCH)) + extension
	content, err := tf.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read installed plugin from %s: %s", path, err)
	}
	if strings.TrimSpace(string(content)) != "this is not a real plugin" {
		t.Errorf("template plugin was not installed from local cache")
	}

	nullLinkPath := filepath.FromSlash(fmt.Sprintf(".terraform/providers/registry.opentofu.org/hashicorp/null/2.1.0/%s_%s/terraform-provider-null", runtime.GOOS, runtime.GOARCH)) + extension
	if !tf.FileExists(nullLinkPath) {
		t.Errorf("null plugin was not installed into %s", nullLinkPath)
	}

	nullCachePath := filepath.FromSlash(fmt.Sprintf("cache/registry.opentofu.org/hashicorp/null/2.1.0/%s_%s/terraform-provider-null", runtime.GOOS, runtime.GOARCH)) + extension
	if !tf.FileExists(nullCachePath) {
		t.Errorf("null plugin is not in cache after install. expected in: %s", nullCachePath)
	}
}

func escapeStringJSON(v string) string {
	b := &strings.Builder{}

	enc := json.NewEncoder(b)

	enc.SetEscapeHTML(false)

	if err := enc.Encode(v); err != nil {
		panic("failed to escapeStringJSON: " + v)
	}

	marshaledV := b.String()

	// shouldn't happen
	if len(marshaledV) < 2 {
		return string(marshaledV)
	}

	return string(marshaledV[1 : len(marshaledV)-2])
}

// TestTelemetrySchemaConflict reproduces the issue where Farseek fails to initialize
// telemetry due to conflicting OpenTelemetry schema URLs from different semconv versions.
//
// The issue occurs because different parts of the codebase import different versions
// of go.opentelemetry.io/otel/semconv, like internal/tracing/init.go.
//
// When OTEL_* variables are set, OpenTelemetry tries to initialize and
// detects these conflicting schema URLs, causing conflicting schema errors.
// For more information, see: https://github.com/rafagsiqueira/farseek/pull/3446
func TestTelemetrySchemaConflict(t *testing.T) {
	t.Parallel()

	fixturePath := filepath.Join("testdata", "empty")
	tf := e2e.NewBinary(t, farseekBin, fixturePath)

	// Set the environment variable that triggers telemetry initialization errors
	tf.AddEnv("OTEL_TRACES_EXPORTER=otlp")
	// We're actually using an invalid endpoint because the key error is the
	// initialization error. Sending the traces themselves is not relevant to this test.
	tf.AddEnv("OTEL_EXPORTER_OTLP_ENDPOINT=http://invalid/")
	tf.AddEnv("OTEL_EXPORTER_OTLP_INSECURE=true")

	_, stderr, err := tf.Run("init")

	if err != nil {
		t.Fatalf("Expected success, got error: %s", err)
	}

	if strings.Contains(stderr, "Could not initialize telemetry") {
		t.Errorf("Expected no error message to contain 'Could not initialize telemetry', but got: %s", stderr)
	}

	if strings.Contains(stderr, "conflicting Schema URL") {
		t.Errorf("Expected no error message to contain 'conflicting Schema URL', but got: %s", stderr)
	}
}
