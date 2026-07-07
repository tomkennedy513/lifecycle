package archive_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/buildpacks/lifecycle/archive"
)

func TestCheckPathIsSafe(t *testing.T) {
	root := filepath.Join("/tmp", "abc", "layers")
	tests := []struct {
		name    string
		path    string
		isDir   bool
		wantErr string
	}{
		{"in-root file", filepath.Join(root, "sbom", "launch", "f"), false, ""},
		{"root itself", root, true, ""},
		{"ancestor dir allowed", filepath.Join("/tmp", "abc"), true, ""},
		{"ancestor top dir allowed", "/tmp", true, ""},
		{"ancestor as non-dir rejected", filepath.Join("/tmp", "abc"), false, "escapes destination root"},
		{"escaping sibling rejected", "/home/cnb/.profile", true, "escapes destination root"},
		{"prefix-not-ancestor rejected", filepath.Join("/tmp", "abc", "lay"), true, "escapes destination root"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := archive.CheckPathIsSafe(tc.path, root, map[string]bool{}, tc.isDir)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}
