package examples_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestExamplesRun(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	root := filepath.Dir(source)
	examples := []string{
		"semantic-map", "custom-identity", "loading-cache", "concurrent-cache",
		"interner", "probabilistic", "ordered-workflow", "multimap-tags",
		"multiset-frequency", "concurrent-map", "counting-bloom", "cuckoo-filter",
		"xorfilter-allowlist", "sketch-stream", "hasher-composition", "cache-policies",
	}
	for _, name := range examples {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "go", "run", "./"+name)
			cmd.Dir = root
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("go run failed: %v\n%s", err, output)
			}
		})
	}
}
