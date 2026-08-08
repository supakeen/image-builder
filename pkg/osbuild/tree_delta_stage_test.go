package osbuild_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/image-builder/pkg/osbuild"
)

func TestTreeDeltaStageJsonMinimal(t *testing.T) {
	expectedJson := `{
  "type": "org.osbuild.tree-delta",
  "inputs": {
    "reference": {
      "type": "org.osbuild.tree",
      "origin": "org.osbuild.pipeline",
      "references": [
        "name:base-os"
      ]
    },
    "overlay": {
      "type": "org.osbuild.tree",
      "origin": "org.osbuild.pipeline",
      "references": [
        "name:sysext"
      ]
    }
  },
  "options": {}
}`

	opts := &osbuild.TreeDeltaStageOptions{}
	stage := osbuild.NewTreeDeltaStage(opts, "base-os", "sysext")
	require.NotNil(t, stage)

	js, err := json.MarshalIndent(stage, "", "  ")
	require.Nil(t, err)
	assert.Equal(t, expectedJson, string(js))
}

func TestTreeDeltaStageJsonFull(t *testing.T) {
	expectedJson := `{
  "type": "org.osbuild.tree-delta",
  "inputs": {
    "reference": {
      "type": "org.osbuild.tree",
      "origin": "org.osbuild.pipeline",
      "references": [
        "name:base-os"
      ]
    },
    "overlay": {
      "type": "org.osbuild.tree",
      "origin": "org.osbuild.pipeline",
      "references": [
        "name:sysext"
      ]
    }
  },
  "options": {
    "paths": [
      "/usr",
      "/opt"
    ],
    "exclude_paths": [
      "/usr/share/doc"
    ]
  }
}`

	opts := &osbuild.TreeDeltaStageOptions{
		Paths:        []string{"/usr", "/opt"},
		ExcludePaths: []string{"/usr/share/doc"},
	}
	stage := osbuild.NewTreeDeltaStage(opts, "base-os", "sysext")
	require.NotNil(t, stage)

	js, err := json.MarshalIndent(stage, "", "  ")
	require.Nil(t, err)
	assert.Equal(t, expectedJson, string(js))
}
