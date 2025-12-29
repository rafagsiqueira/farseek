// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package farseek

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/rafagsiqueira/farseek/internal/configs"
	"github.com/zclconf/go-cty/cty"
)

// ResourceDiscoverer is an interface for finding which resources have changed.
type ResourceDiscoverer interface {
	DiscoverChangedResources(dir, baseSHA string, includeUncommitted bool) ([]DiscoveredResource, error)
	DiscoverAllResources(dir string, includeUncommitted bool) ([]DiscoveredResource, error)
	GetResourceAttributeFromSHA(dir, sha, filename, address, attribute string) (string, error)
	GetCurrentSHA(dir string) (string, error)
}

type DiscoveredResource struct {
	Address  string
	Filename string
	Config   hcl.Body // Might be nil if only in old state or if we couldn't parse it
	IsNew    bool     // True if not in base search (e.g. not in Git history)
}

// GitDiscoverer implements ResourceDiscoverer using Git.
type GitDiscoverer struct{}

func (g GitDiscoverer) DiscoverChangedResources(dir, baseSHA string, includeUncommitted bool) ([]DiscoveredResource, error) {
	var files []string
	var err error

	if baseSHA == "" {
		// If no base SHA, we consider all .tf files as "changed" (new)
		log.Printf("[INFO] Farseek: No .farseek_sha found, discovering all resources in %s", dir)
		files, err = g.getAllTfFiles(dir)
	} else {
		files, err = g.getChangedFiles(dir, baseSHA, includeUncommitted)
	}

	if err != nil {
		return nil, err
	}

	// Map to verify which resources are in changed files
	isChangedFile := make(map[string]bool)
	for _, f := range files {
		isChangedFile[f] = true
	}

	// Load historical resources if baseSHA is present
	historicalResources := make(map[string]bool)
	if baseSHA != "" {
		// Actually we only care about resources that were in the project at baseSHA.
		// Let's just get all resources at that SHA.
		histRes, err := g.discoverAllResourcesAtSHA(dir, baseSHA)
		if err == nil {
			for _, dr := range histRes {
				historicalResources[dr.Address] = true
			}
		}
	}

	// Load current configuration
	parser := configs.NewParser(nil)
	allFiles, err := g.getAllTfFiles(dir)
	if err != nil {
		return nil, err
	}

	var results []DiscoveredResource
	currentAddresses := make(map[string]bool)

	for _, f := range allFiles {
		path := filepath.Join(dir, f)
		file, diags := parser.LoadConfigFile(path)
		if diags.HasErrors() {
			continue
		}

		processResource := func(addr string, body hcl.Body) {
			currentAddresses[addr] = true
			if isChangedFile[f] {
				_, existed := historicalResources[addr]
				results = append(results, DiscoveredResource{
					Address:  addr,
					Filename: f,
					Config:   body,
					IsNew:    !existed,
				})
			}
		}

		for _, r := range file.ManagedResources {
			addr := r.Type + "." + r.Name
			processResource(addr, r.Config)
		}
		for _, d := range file.DataResources {
			addr := "data." + d.Type + "." + d.Name
			processResource(addr, d.Config)
		}
	}

	// Handle Deletions (resources that were in historicalResources but not in currentAddresses)
	for addr, _ := range historicalResources {
		if _, exists := currentAddresses[addr]; !exists {
			// Find which file it was in.
			// We can get this from discoverAllResourcesAtSHA if we keep the filename there.
		}
	}

	// Wait, the previous deletion logic was iteration over changed files and checking history of EACH.
	// That was better for finding WHICH file was deleted.

	// Let's stick closer to previous logic but use the historicalResources cache.
	if baseSHA != "" {
		histRes, _ := g.discoverAllResourcesAtSHA(dir, baseSHA)
		for _, dr := range histRes {
			if _, exists := currentAddresses[dr.Address]; !exists {
				// It was deleted!
				results = append(results, DiscoveredResource{
					Address:  dr.Address,
					Filename: dr.Filename,
					Config:   nil,
					IsNew:    false, // It existed before, so it's not "new" in the additive sense
				})
			}
		}
	}

	return results, nil
}

func (g GitDiscoverer) DiscoverAllResources(dir string, includeUncommitted bool) ([]DiscoveredResource, error) {
	if includeUncommitted {
		return g.discoverAllResourcesInWorkingDir(dir)
	}
	return g.discoverAllResourcesAtSHA(dir, "HEAD")
}

func (g GitDiscoverer) discoverAllResourcesInWorkingDir(dir string) ([]DiscoveredResource, error) {
	// Consider all .tf files in working directory
	files, err := g.getAllTfFiles(dir)
	if err != nil {
		return nil, err
	}

	parser := configs.NewParser(nil)
	var results []DiscoveredResource

	for _, f := range files {
		path := filepath.Join(dir, f)
		file, diags := parser.LoadConfigFile(path)
		if diags.HasErrors() {
			continue
		}

		for _, r := range file.ManagedResources {
			addr := r.Type + "." + r.Name
			results = append(results, DiscoveredResource{
				Address:  addr,
				Filename: f,
				Config:   r.Config,
			})
		}
		for _, d := range file.DataResources {
			addr := "data." + d.Type + "." + d.Name
			results = append(results, DiscoveredResource{
				Address:  addr,
				Filename: f,
				Config:   d.Config,
			})
		}
	}

	return results, nil
}

func (g GitDiscoverer) discoverAllResourcesAtSHA(dir, sha string) ([]DiscoveredResource, error) {
	// List all files at SHA
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", sha)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var results []DiscoveredResource

	for _, f := range lines {
		if f == "" || !(strings.HasSuffix(f, ".tf") || strings.HasSuffix(f, ".tf.json")) {
			continue
		}

		content, err := g.getFileContentAtSHA(dir, sha, f)
		if err != nil {
			continue
		}

		// Parse strictly to find addresses
		synFile, diags := hclsyntax.ParseConfig(content, f, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() || synFile == nil {
			continue
		}

		body, ok := synFile.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}

		for _, block := range body.Blocks {
			if block.Type == "resource" && len(block.Labels) == 2 {
				addr := block.Labels[0] + "." + block.Labels[1]
				results = append(results, DiscoveredResource{
					Address:  addr,
					Filename: f,
					Config:   nil, // We don't have a current config for it if it's from history
				})
			}
			if block.Type == "data" && len(block.Labels) == 2 {
				addr := "data." + block.Labels[0] + "." + block.Labels[1]
				results = append(results, DiscoveredResource{
					Address:  addr,
					Filename: f,
					Config:   nil,
				})
			}
		}
	}

	return results, nil
}

func (g GitDiscoverer) getAllTfFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".tf") || strings.HasSuffix(path, ".tf.json")) {
			rel, err := filepath.Rel(dir, path)
			if err == nil {
				files = append(files, rel)
			}
		}
		// Don't descend into subdirectories for now, Tofu usually handles them separately
		// unless they are explicitly targeted.
		if info.IsDir() && path != dir {
			return filepath.SkipDir
		}
		return nil
	})
	return files, err
}

func (g GitDiscoverer) getFileContentAtSHA(dir, sha, path string) ([]byte, error) {
	// Using ./path with git show ensures it's relative to the current directory
	// even if we are not at the repo root.
	cmd := exec.Command("git", "show", sha+":./"+path)
	cmd.Dir = dir
	return cmd.Output()
}

func (g GitDiscoverer) getChangedFiles(dir, baseSHA string, includeUncommitted bool) ([]string, error) {
	args := []string{"diff", "--name-only", "--relative", baseSHA}
	if !includeUncommitted {
		args = append(args, "HEAD")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" && (strings.HasSuffix(line, ".tf") || strings.HasSuffix(line, ".tf.json")) {
			files = append(files, line)
		}
	}
	return files, nil
}

func (g GitDiscoverer) GetCurrentSHA(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetResourceAttributeFromSHA extracts a specific attribute (e.g. "name") from a resource block
// in a file at a specific SHA. It uses best-effort parsing.
func (g GitDiscoverer) GetResourceAttributeFromSHA(dir, sha, filename, address, attribute string) (string, error) {
	content, err := g.getFileContentAtSHA(dir, sha, filename)
	if err != nil {
		return "", err
	}

	// Parse the file
	file, diags := hclsyntax.ParseConfig(content, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return "", diags
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return "", nil
	}

	// Find the resource block
	parts := strings.Split(address, ".")
	if len(parts) < 2 {
		return "", nil
	}
	// address: type.name OR module...type.name
	// We handle root resources for now.
	resType := parts[0]
	resName := parts[1]
	if parts[0] == "data" && len(parts) >= 3 {
		resType = "data"
		resName = parts[1] // Actually data.type.name means type=type, name=name.
		// But in HCL: data "type" "name"
		if len(parts) > 2 {
			// type is parts[1], name is parts[2]
			// But wait, parts[0] is "data".
			// So resType should be parts[1]?
			// No, finding block by type "data", labels: [parts[1], parts[2]]
		}
	}

	for _, block := range body.Blocks {
		if parts[0] == "data" {
			if block.Type == "data" && len(block.Labels) == 2 {
				if block.Labels[0] == parts[1] && block.Labels[1] == parts[2] {
					goto Found
				}
			}
		} else {
			if block.Type == "resource" && len(block.Labels) == 2 {
				if block.Labels[0] == resType && block.Labels[1] == resName {
					goto Found
				}
			}
		}
		continue

	Found:
		attr, ok := block.Body.Attributes[attribute]
		if ok {
			// Extract value
			// We only support literal strings for now
			val, diags := attr.Expr.Value(nil)
			if !diags.HasErrors() && val.Type() == cty.String {
				return val.AsString(), nil
			}
		}
		return "", nil
	}
	return "", nil
}

// Global discoverer that can be overridden in tests.
var Discovery ResourceDiscoverer = GitDiscoverer{}
