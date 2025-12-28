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
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/rafagsiqueira/farseek/internal/e2e"
	"github.com/rafagsiqueira/farseek/internal/grpcwrap"
	tfplugin "github.com/rafagsiqueira/farseek/internal/plugin6"
	simple "github.com/rafagsiqueira/farseek/internal/provider-simple-v6"
	proto "github.com/rafagsiqueira/farseek/internal/tfplugin6"
)

type reattachConfig struct {
	Protocol        string
	ProtocolVersion int
	Pid             int
	Test            bool
	Addr            reattachConfigAddr
}

type reattachConfigAddr struct {
	Network string
	String  string
}

type providerServer struct {
	sync.Mutex
	proto.ProviderServer
	planResourceChangeCalled  bool
	applyResourceChangeCalled bool
}

func (p *providerServer) PlanResourceChange(ctx context.Context, req *proto.PlanResourceChange_Request) (*proto.PlanResourceChange_Response, error) {
	p.Lock()
	defer p.Unlock()

	p.planResourceChangeCalled = true
	return p.ProviderServer.PlanResourceChange(ctx, req)
}

func (p *providerServer) ApplyResourceChange(ctx context.Context, req *proto.ApplyResourceChange_Request) (*proto.ApplyResourceChange_Response, error) {
	p.Lock()
	defer p.Unlock()

	p.applyResourceChangeCalled = true
	return p.ProviderServer.ApplyResourceChange(ctx, req)
}

func (p *providerServer) PlanResourceChangeCalled() bool {
	p.Lock()
	defer p.Unlock()

	return p.planResourceChangeCalled
}

func (p *providerServer) ApplyResourceChangeCalled() bool {
	p.Lock()
	defer p.Unlock()

	return p.applyResourceChangeCalled
}

func (p *providerServer) ResetApplyResourceChangeCalled() {
	p.Lock()
	defer p.Unlock()

	p.applyResourceChangeCalled = false
}

func TestStatelessDestroy(t *testing.T) {
	t.Parallel()

	fixturePath := filepath.Join("testdata", "test-provider")
	tf := e2e.NewBinary(t, farseekBin, fixturePath)

	reattachCh := make(chan *plugin.ReattachConfig)
	closeCh := make(chan struct{})
	provider := &providerServer{
		ProviderServer: grpcwrap.Provider6(simple.Provider()),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
		t.Fatal(err)
	}

	tf.AddEnv("TF_REATTACH_PROVIDERS=" + string(reattachStr))
	tf.AddEnv("FARSEEK_LOG=TRACE")

	// Farseek needs a git repo
	tempDir := tf.WorkDir()
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tempDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s\nOutput: %s", args, err, string(out))
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "test")
	runGit("add", ".")
	runGit("commit", "-m", "initial")

	//// INIT
	_, stderr, err := tf.Run("init")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	//// APPLY (Initialize "cloud" state)
	_, stderr, err = tf.Run("apply", "-auto-approve")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}
	provider.ResetApplyResourceChangeCalled()

	//// DESTROY (Stateless)
	// This should discover resources in HEAD and plan their deletion
	_, stderr, err = tf.Run("destroy", "-auto-approve")
	if err != nil {
		t.Fatalf("unexpected destroy error: %s\nstderr:\n%s", err, stderr)
	}

	if !provider.ApplyResourceChangeCalled() {
		t.Errorf("ApplyResourceChange (destroy) not called on in-process provider\nstderr:\n%s", stderr)
	}

	// DESTROY --uncommitted
	// Add a new resource, uncommitted
	f, err := os.OpenFile(filepath.Join(tempDir, "main.tf"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("\nresource \"simple_resource\" \"uncommitted\" {}\n")
	f.Close()

	provider.ResetApplyResourceChangeCalled()
	_, stderr, err = tf.Run("destroy", "-auto-approve", "--uncommitted")
	if err != nil {
		t.Fatalf("unexpected destroy --uncommitted error: %s\nstderr:\n%s", err, stderr)
	}

	if !provider.ApplyResourceChangeCalled() {
		t.Errorf("ApplyResourceChange (destroy --uncommitted) not called on in-process provider\nstderr:\n%s", stderr)
	}

	cancel()
	<-closeCh
}
