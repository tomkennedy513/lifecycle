package layers_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/lifecycle/layers"
	h "github.com/buildpacks/lifecycle/testhelpers"
)

func TestExtract(t *testing.T) {
	spec.Run(t, "extract", testLayersExtract, spec.Report(report.Terminal{}))
}

func testLayersExtract(t *testing.T, when spec.G, it spec.S) {
	var layersDir, artifactsDir string

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "lifecycle.extract.layers.")
		h.AssertNil(t, err)
		artifactsDir, err = os.MkdirTemp("", "lifecycle.extract.artifacts.")
		h.AssertNil(t, err)
	})

	it.After(func() {
		_ = os.RemoveAll(layersDir)
		_ = os.RemoveAll(artifactsDir)
	})

	when("#Extract with a confinement root", func() {
		it("extracts a layer whose entries live under the root (including parent dirs)", func() {
			src := filepath.Join(layersDir, "sbom", "launch")
			h.Mkdir(t, src)
			h.Mkfile(t, "bom-data", filepath.Join(src, "some-file"))

			factory := &layers.Factory{ArtifactsDir: artifactsDir}
			layer, err := factory.DirLayer("launch.sbom", src, "")
			h.AssertNil(t, err)

			h.AssertNil(t, os.RemoveAll(filepath.Join(layersDir, "sbom")))

			rc, err := os.Open(layer.TarPath)
			h.AssertNil(t, err)
			defer rc.Close()

			h.AssertNil(t, layers.Extract(rc, layersDir))

			got := h.MustReadFile(t, filepath.Join(layersDir, "sbom", "launch", "some-file"))
			h.AssertEq(t, string(got), "bom-data")
		})
	})
}
