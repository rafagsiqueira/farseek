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
	DiscoverChangedResources(dir, baseSHA string) ([]DiscoveredResource, error)
	GetResourceAttributeFromSHA(dir, sha, filename, address, attribute string) (string, error)
	GetCurrentSHA(dir string) (string, error)
}

type DiscoveredResource struct {
	Address  string
	Filename string
	Config   hcl.Body // Might be nil if only in old state or if we couldn't parse it
}

// GitDiscoverer implements ResourceDiscoverer using Git.
type GitDiscoverer struct{}

func (g GitDiscoverer) DiscoverChangedResources(dir, baseSHA string) ([]DiscoveredResource, error) {
	var files []string
	var err error

	if baseSHA == "" {
		// If no base SHA, we consider all .tf files as "changed" (new)
		log.Printf("[INFO] Farseek: No .farseek_sha found, discovering all resources in %s", dir)
		files, err = g.getAllTfFiles(dir)
	} else {
		files, err = g.getChangedFiles(dir, baseSHA)
	}

	if err != nil {
		return nil, err
	}

	// Map to verify which resources are in changed files
	isChanged := make(map[string]bool)
	for _, f := range files {
		isChanged[f] = true
	}

	// Load configuration to map resources to files
	parser := configs.NewParser(nil)
	allFiles, err := g.getAllTfFiles(dir)
	if err != nil {
		return nil, err
	}

	type resInfo struct {
		Config hcl.Body
		File   string
	}
	addressMap := make(map[string]resInfo)

	for _, f := range allFiles {
		path := filepath.Join(dir, f)
		file, diags := parser.LoadConfigFile(path)
		if diags.HasErrors() {
			continue
		}

		for _, r := range file.ManagedResources {
			addr := r.Type + "." + r.Name
			addressMap[addr] = resInfo{Config: r.Config, File: f}
		}
		for _, d := range file.DataResources {
			addr := "data." + d.Type + "." + d.Name
			addressMap[addr] = resInfo{Config: d.Config, File: f}
		}
	}

	var results []DiscoveredResource

	// Add updated/created resources
	for addr, info := range addressMap {
		if isChanged[info.File] {
			results = append(results, DiscoveredResource{
				Address:  addr,
				Filename: info.File,
				Config:   info.Config,
			})
		}
	}

	// Handle Deletions
	for _, f := range files {
		// If baseSHA is set, check old content
		if baseSHA != "" {
			ct, err := g.getFileContentAtSHA(dir, baseSHA, f)
			if err == nil {
				// Parse old file strictly to find addresses
				synFile, diags := hclsyntax.ParseConfig(ct, f, hcl.Pos{Line: 1, Column: 1})
				if !diags.HasErrors() && synFile != nil {
					if body, ok := synFile.Body.(*hclsyntax.Body); ok {
						for _, block := range body.Blocks {
							if block.Type == "resource" && len(block.Labels) == 2 {
								addr := block.Labels[0] + "." + block.Labels[1]
								// If not in current addressMap, it's a deletion!
								if _, exists := addressMap[addr]; !exists {
									results = append(results, DiscoveredResource{
										Address:  addr,
										Filename: f,
										Config:   nil, // Deleted, so no current config
									})
								}
							}
						}
					}
				}
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

func (g GitDiscoverer) getChangedFiles(dir, baseSHA string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "--relative", baseSHA, "HEAD")
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
