package farseek

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitDiscoverer_DiscoverChangedResources(t *testing.T) {
	// Create a temporary directory for the git repo
	dir := t.TempDir()

	// Initialize git repo
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "you@example.com")
	runGit(t, dir, "config", "user.name", "Your Name")

	// Create a main.tf file and commit it
	mainTf := filepath.Join(dir, "main.tf")
	err := os.WriteFile(mainTf, []byte(`resource "test_instance" "foo" {}`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "main.tf")
	runGit(t, dir, "commit", "-m", "Initial commit")

	// Get the initial commit SHA
	baseSHA := getHeadSHA(t, dir)

	// Create .farseek_sha
	err = WriteSHA(dir, baseSHA)
	if err != nil {
		t.Fatal(err)
	}

	g := GitDiscoverer{}

	// Case 1: No changes
	resources, err := g.DiscoverChangedResources(dir, baseSHA, false)
	if err != nil {
		t.Fatalf("DiscoverChangedResources failed: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("Expected 0 changes, got %d", len(resources))
	}

	// Case 2: Uncommitted change
	// Modify main.tf
	err = os.WriteFile(mainTf, []byte(`resource "test_instance" "foo" { count = 2 }`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Check without uncommitted flag -> should still be 0 because we compare against HEAD
	resources, err = g.DiscoverChangedResources(dir, baseSHA, false)
	if err != nil {
		t.Fatalf("DiscoverChangedResources failed: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("Expected 0 changes without uncommitted flag, got %d", len(resources))
	}

	// Check WITH uncommitted flag -> should be 1
	resources, err = g.DiscoverChangedResources(dir, baseSHA, true)
	if err != nil {
		t.Fatalf("DiscoverChangedResources failed: %v", err)
	}
	if len(resources) != 1 {
		t.Errorf("Expected 1 change with uncommitted flag, got %d", len(resources))
	}
	if len(resources) > 0 && resources[0].Address != "test_instance.foo" {
		t.Errorf("Expected address test_instance.foo, got %s", resources[0].Address)
	}

	// Case 3: Committed change
	runGit(t, dir, "add", "main.tf")
	runGit(t, dir, "commit", "-m", "Update main.tf")

	// Now baseSHA is behind HEAD.
	// Without uncommitted flag (default behavior), it should see the change since it is committed.
	resources, err = g.DiscoverChangedResources(dir, baseSHA, false)
	if err != nil {
		t.Fatalf("DiscoverChangedResources failed: %v", err)
	}
	if len(resources) != 1 {
		t.Errorf("Expected 1 change after commit, got %d", len(resources))
	}

	// With uncommitted flag, it should ALS0 see the change (diff baseSHA..WORKDIR includes applied commits)
	resources, err = g.DiscoverChangedResources(dir, baseSHA, true)
	if err != nil {
		t.Fatalf("DiscoverChangedResources failed: %v", err)
	}
	if len(resources) != 1 {
		t.Errorf("Expected 1 change after commit with uncommitted flag, got %d", len(resources))
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
	}
}

func getHeadSHA(t *testing.T, dir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out[:len(out)-1]) // TRIM newline
}
