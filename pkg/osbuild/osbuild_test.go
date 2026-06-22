// Package osbuild provides primitives for representing and (un)marshalling
// OSBuild types.
package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipeline_AddStage(t *testing.T) {
	expectedPipeline := &Pipeline{
		Build: "name:build",
		Stages: []*Stage{
			{
				Type: "org.osbuild.rpm",
			},
		},
	}
	actualPipeline := &Pipeline{
		Build: "name:build",
	}
	actualPipeline.AddStage(&Stage{
		Type: "org.osbuild.rpm",
	})
	assert.Equal(t, expectedPipeline, actualPipeline)
	assert.Equal(t, 1, len(actualPipeline.Stages))
}

func TestPipeline_SetDefaultMounts(t *testing.T) {
	pipeline := &Pipeline{}
	pipeline.SetDefaultMounts(
		*NewExt4Mount("root", "root-dev", "/"),
		*NewXfsMount("data", "data-dev", "/data"),
	)

	pipeline.AddStage(&Stage{Type: "org.osbuild.rpm"})
	pipeline.AddStage(&Stage{Type: "org.osbuild.locale"})

	assert.Len(t, pipeline.Stages, 2)
	for _, stage := range pipeline.Stages {
		assert.Len(t, stage.Mounts, 2)
		assert.Equal(t, "root", stage.Mounts[0].Name)
		assert.Equal(t, "data", stage.Mounts[1].Name)
	}
}

func TestPipeline_SetDefaultMountsAppendsToStageMounts(t *testing.T) {
	pipeline := &Pipeline{}
	pipeline.SetDefaultMounts(*NewExt4Mount("root", "root-dev", "/"))

	stage := &Stage{
		Type:   "org.osbuild.rpm",
		Mounts: []Mount{*NewXfsMount("existing", "existing-dev", "/existing")},
	}
	pipeline.AddStage(stage)

	assert.Len(t, pipeline.Stages[0].Mounts, 2)
	assert.Equal(t, "existing", pipeline.Stages[0].Mounts[0].Name)
	assert.Equal(t, "root", pipeline.Stages[0].Mounts[1].Name)
}

func TestPipeline_AddStageNilSkipsMounts(t *testing.T) {
	pipeline := &Pipeline{}
	pipeline.SetDefaultMounts(*NewExt4Mount("root", "root-dev", "/"))
	pipeline.AddStage(nil)

	assert.Len(t, pipeline.Stages, 0)
}

func TestPipeline_AddStagesWithMounts(t *testing.T) {
	pipeline := &Pipeline{}
	pipeline.SetDefaultMounts(*NewFATMount("efi", "efi-dev", "/boot/efi"))

	stages := []*Stage{
		{Type: "org.osbuild.rpm"},
		{Type: "org.osbuild.locale"},
		{Type: "org.osbuild.hostname"},
	}
	pipeline.AddStages(stages...)

	assert.Len(t, pipeline.Stages, 3)
	for _, stage := range pipeline.Stages {
		assert.Len(t, stage.Mounts, 1)
		assert.Equal(t, "efi", stage.Mounts[0].Name)
	}
}

func TestPipeline_NoMountsDoesNotAffectStages(t *testing.T) {
	pipeline := &Pipeline{}
	pipeline.AddStage(&Stage{Type: "org.osbuild.rpm"})

	assert.Len(t, pipeline.Stages[0].Mounts, 0)
}

func TestPipeline_SetDefaultDevices(t *testing.T) {
	pipeline := &Pipeline{}
	pipeline.SetDefaultDevices(map[string]Device{
		"root": {Type: "org.osbuild.loopback"},
		"boot": {Type: "org.osbuild.loopback"},
	})

	pipeline.AddStage(&Stage{Type: "org.osbuild.rpm"})
	pipeline.AddStage(&Stage{Type: "org.osbuild.locale"})

	assert.Len(t, pipeline.Stages, 2)
	for _, stage := range pipeline.Stages {
		assert.Len(t, stage.Devices, 2)
		assert.Equal(t, "org.osbuild.loopback", stage.Devices["root"].Type)
		assert.Equal(t, "org.osbuild.loopback", stage.Devices["boot"].Type)
	}
}

func TestPipeline_SetDefaultDevicesPreservesExisting(t *testing.T) {
	pipeline := &Pipeline{}
	pipeline.SetDefaultDevices(map[string]Device{
		"root": {Type: "org.osbuild.loopback"},
	})

	stage := &Stage{
		Type: "org.osbuild.rpm",
		Devices: map[string]Device{
			"root": {Type: "org.osbuild.luks2"},
		},
	}
	pipeline.AddStage(stage)

	assert.Len(t, pipeline.Stages[0].Devices, 1)
	assert.Equal(t, "org.osbuild.luks2", pipeline.Stages[0].Devices["root"].Type)
}

func TestPipeline_NoDevicesDoesNotAffectStages(t *testing.T) {
	pipeline := &Pipeline{}
	pipeline.AddStage(&Stage{Type: "org.osbuild.rpm"})

	assert.Nil(t, pipeline.Stages[0].Devices)
}

func TestPipeline_DefaultMountsAndDevicesTogether(t *testing.T) {
	pipeline := &Pipeline{}
	pipeline.SetDefaultMounts(*NewExt4Mount("root", "root-dev", "/"))
	pipeline.SetDefaultDevices(map[string]Device{
		"root-dev": {Type: "org.osbuild.loopback"},
	})

	pipeline.AddStage(&Stage{Type: "org.osbuild.rpm"})

	assert.Len(t, pipeline.Stages[0].Mounts, 1)
	assert.Len(t, pipeline.Stages[0].Devices, 1)
}

var fakeOsbuildManifestWithIdentifiers = []byte(`{
  "version": "2",
  "pipelines": [
    {
       "name": "build",
       "stages": [
         {
			"id": "1234",
            "type": "org.osbuild.rpm"
         },
         {
			"id": "5678",
            "type": "org.osbuild.mkdir"
         }
       ]
    }
  ]
}`)

func TestManifestFromBytes(t *testing.T) {
	manifest, err := NewManifestFromBytes(fakeOsbuildManifestWithIdentifiers)
	assert.NoError(t, err)

	assert.Equal(t, manifest.Pipelines[0].Stages[0].ID, "1234")
	assert.Equal(t, manifest.Pipelines[0].Stages[1].ID, "5678")

	pID, err := manifest.Pipelines[0].GetID()
	assert.NoError(t, err)

	assert.Equal(t, pID, "5678")
}
