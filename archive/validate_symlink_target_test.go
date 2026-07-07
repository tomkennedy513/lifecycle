package archive

import (
	"testing"
)

func TestValidateSymlinkTarget(t *testing.T) {
	tests := []struct {
		testName string
		name     string
		linkname string
		destRoot string
		wantErr  bool
	}{
		{testName: "absolute path outside destRoot", name: "/dest/layers/foo", linkname: "/some/path/outside/dest", destRoot: "/dest", wantErr: true},
		{testName: "relative path escapes destRoot", name: "/dest/layers/foo", linkname: "../../some/path/outside/dest", destRoot: "/dest", wantErr: true},
		{testName: "relative path escapes from deep path", name: "/dest/a/b/c/foo", linkname: "../../../../some/path/outside/dest", destRoot: "/dest", wantErr: true},
		{testName: "absolute path within destRoot", name: "/dest/layers/foo", linkname: "/dest/other", destRoot: "/dest", wantErr: false},
		{testName: "relative path within destRoot", name: "/dest/layers/foo", linkname: "../other", destRoot: "/dest", wantErr: false},
		{testName: "relative path stays within destRoot deeply", name: "/dest/a/b/foo", linkname: "../../a/c", destRoot: "/dest", wantErr: false},
		{testName: "points exactly to destRoot", name: "/dest/foo", linkname: "/dest", destRoot: "/dest", wantErr: false},
		// this is a limitation where destRoot="/" cannot restrict absolute symlinks, adding test to show its expected behavior
		{testName: "destRoot / does not restrict absolute symlinks", name: "/layers/foo", linkname: "/home/cnb", destRoot: "/", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if err := validateSymlinkTarget(tt.name, tt.linkname, tt.destRoot); (err != nil) != tt.wantErr {
				t.Errorf("%s: want error -> %t, got error -> %t", tt.testName, tt.wantErr, err != nil)
			}
		})
	}
}
