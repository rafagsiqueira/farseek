package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	tmpDir, err := os.MkdirTemp("", "farseek-test")
	if err != nil {
		fmt.Printf("Error creating temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("Temp Dir: %s\n", tmpDir)

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("git rev-parse failed (as expected?): %v\n", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Printf("Exit code: %d\n", exitErr.ExitCode())
		}
	} else {
		fmt.Printf("git rev-parse succeeded: %s\n", string(out))
	}
}
