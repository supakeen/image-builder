package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDDStage(t *testing.T) {
	offset := 512
	options := DDStageOptions{
		Src:       "input://image/disk.img",
		Dst:       "disk.img",
		SrcOffset: &offset,
		Count:     1048576,
	}
	inputs := NewPipelineTreeInputs("image", "build")

	expectedStage := &Stage{
		Type:    "org.osbuild.dd",
		Options: &options,
		Inputs:  inputs,
	}
	actualStage := NewDDStage(&options, inputs)
	assert.Equal(t, expectedStage, actualStage)
}

func TestNewDDStageNoOffset(t *testing.T) {
	options := DDStageOptions{
		Src:   "tree:///disk.img",
		Dst:   "output.img",
		Count: 4096,
	}

	expectedStage := &Stage{
		Type:    "org.osbuild.dd",
		Options: &options,
	}
	actualStage := NewDDStage(&options, nil)
	assert.Equal(t, expectedStage, actualStage)
}
