// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package e2etest

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/rafagsiqueira/farseek/internal/e2e"
	"github.com/rafagsiqueira/farseek/internal/grpcwrap"
	tfplugin "github.com/rafagsiqueira/farseek/internal/plugin6"
	simple "github.com/rafagsiqueira/farseek/internal/provider-simple-v6"
	proto "github.com/rafagsiqueira/farseek/internal/tfplugin6"
)

func setupGitScenario(t *testing.T) (*e2e.Binary, *providerServer, func()) {
	fixturePath := filepath.Join("testdata", "test-provider")
	tf := e2e.NewBinary(t, farseekBin, fixturePath)

	reattachCh := make(chan *plugin.ReattachConfig)
	closeCh := make(chan struct{})
	provider := &providerServer{
		ProviderServer: grpcwrap.Provider6(simple.Provider()),
	}
	ctx, cancel := context.WithCancel(context.Background())

	go plugin.Serve(&plugin.ServeConfig{
		Logger: hclog.New(&hclog.LoggerOptions{
			Name:   "plugintest",
			Level:  hclog.Trace,
			Output: io.Discard,
		}),
		Test: &plugin.ServeTestConfig{
			Context:          ctx,
			ReattachConfigCh: reattachCh,
			CloseCh:          closeCh,
		},
		GRPCServer: plugin.DefaultGRPCServer,
		VersionedPlugins: map[int]plugin.PluginSet{
			6: {
				"provider": &tfplugin.GRPCProviderPlugin{
					GRPCProvider: func() proto.ProviderServer {
						return provider
					},
				},
			},
		},
	})

	config := <-reattachCh
	if config == nil {
		cancel()
		t.Fatalf("no reattach config received")
	}

	reattachStr, err := json.Marshal(map[string]reattachConfig{
		"hashicorp/simple": {
			Protocol:        string(config.Protocol),
			ProtocolVersion: 6,
			Pid:             config.Pid,
			Test:            true,
			Addr: reattachConfigAddr{
				Network: config.Addr.Network(),
				String:  config.Addr.String(),
			},
		},
	})
	if err != nil {
		cancel()
		t.Fatal(err)
	}

	tf.AddEnv("TF_REATTACH_PROVIDERS=" + string(reattachStr))
	tf.AddEnv("FARSEEK_LOG=TRACE")

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tf.WorkDir()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s\nOutput: %s", args, err, string(out))
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "test")

	return tf, provider, func() {
		cancel()
		<-closeCh
	}
}

func runGit(t *testing.T, workDir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s\nOutput: %s", args, err, string(out))
	}
}

// 1) A new .tf file is created with a new resource since the last tracked commit
func TestGitScenario_NewFile(t *testing.T) {
	tf, _, teardown := setupGitScenario(t)
	defer teardown()

	// Initial commit
	tf.WriteFile("main.tf", `resource "simple_resource" "base" {}`)
	runGit(t, tf.WorkDir(), "add", "main.tf")
	runGit(t, tf.WorkDir(), "commit", "-m", "base")

	// Sync mock cloud
	tf.Run("init")
	tf.Run("apply", "-auto-approve")

	// Create new file
	tf.WriteFile("new.tf", `resource "simple_resource" "new" {}`)
	runGit(t, tf.WorkDir(), "add", "new.tf")
	runGit(t, tf.WorkDir(), "commit", "-m", "add new.tf")

	stdout, stderr, err := tf.Run("plan", "-no-color")
	if err != nil {
		t.Fatalf("plan failed: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "1 to add, 0 to change, 0 to destroy") {
		t.Errorf("plan should show 1 to add:\n%s", stdout)
	}
	if !strings.Contains(stdout, "simple_resource.new will be created") {
		t.Errorf("plan should mention simple_resource.new:\n%s", stdout)
	}
}

// 2) A .tf file is deleted since the last tracked commit
func TestGitScenario_DeleteFile(t *testing.T) {
	tf, _, teardown := setupGitScenario(t)
	defer teardown()

	// Initial commit with two files
	tf.WriteFile("main.tf", `resource "simple_resource" "base" {}`)
	tf.WriteFile("delete_me.tf", `resource "simple_resource" "deleted" {}`)
	runGit(t, tf.WorkDir(), "add", ".")
	runGit(t, tf.WorkDir(), "commit", "-m", "base")

	// Sync mock cloud
	tf.Run("init")
	tf.Run("apply", "-auto-approve")

	// Delete file
	os.Remove(filepath.Join(tf.WorkDir(), "delete_me.tf"))
	runGit(t, tf.WorkDir(), "add", "delete_me.tf")
	runGit(t, tf.WorkDir(), "commit", "-m", "delete delete_me.tf")

	stdout, stderr, err := tf.Run("plan", "-no-color")
	if err != nil {
		t.Fatalf("plan failed: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "0 to add, 0 to change, 1 to destroy") {
		t.Errorf("plan should show 1 to destroy:\n%s", stdout)
	}
	if !strings.Contains(stdout, "simple_resource.deleted will be destroyed") {
		t.Errorf("plan should mention simple_resource.deleted for destruction:\n%s", stdout)
	}
}

// 3) A new resource is created in an existing .tf file
func TestGitScenario_NewResourceInExistingFile(t *testing.T) {
	tf, _, teardown := setupGitScenario(t)
	defer teardown()

	// Initial commit
	tf.WriteFile("main.tf", `resource "simple_resource" "base" {}`)
	runGit(t, tf.WorkDir(), "add", "main.tf")
	runGit(t, tf.WorkDir(), "commit", "-m", "base")

	// Sync mock cloud
	tf.Run("init")
	tf.Run("apply", "-auto-approve")

	// Add resource to existing file
	tf.WriteFile("main.tf", `
resource "simple_resource" "base" {}
resource "simple_resource" "new" {}
`)
	runGit(t, tf.WorkDir(), "add", "main.tf")
	runGit(t, tf.WorkDir(), "commit", "-m", "add new resource")

	stdout, stderr, err := tf.Run("plan", "-no-color")
	if err != nil {
		t.Fatalf("plan failed: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "1 to add, 0 to change, 0 to destroy") {
		t.Errorf("plan should show 1 to add:\n%s", stdout)
	}
	if !strings.Contains(stdout, "simple_resource.new will be created") {
		t.Errorf("plan should mention simple_resource.new:\n%s", stdout)
	}
}

// 4) A resource is renamed
func TestGitScenario_RenameResource(t *testing.T) {
	tf, _, teardown := setupGitScenario(t)
	defer teardown()

	// Initial commit
	tf.WriteFile("main.tf", `resource "simple_resource" "old" {}`)
	runGit(t, tf.WorkDir(), "add", "main.tf")
	runGit(t, tf.WorkDir(), "commit", "-m", "base")

	// Sync mock cloud
	tf.Run("init")
	tf.Run("apply", "-auto-approve")

	// Rename resource
	tf.WriteFile("main.tf", `resource "simple_resource" "new" {}`)
	runGit(t, tf.WorkDir(), "add", "main.tf")
	runGit(t, tf.WorkDir(), "commit", "-m", "rename")

	stdout, stderr, err := tf.Run("plan", "-no-color")
	if err != nil {
		t.Fatalf("plan failed: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "1 to add, 0 to change, 1 to destroy") {
		t.Errorf("plan should show 1 to add and 1 to destroy:\n%s", stdout)
	}
	if !strings.Contains(stdout, "simple_resource.old will be destroyed") {
		t.Errorf("plan should mention simple_resource.old for destruction:\n%s", stdout)
	}
	if !strings.Contains(stdout, "simple_resource.new will be created") {
		t.Errorf("plan should mention simple_resource.new for creation:\n%s", stdout)
	}
}
