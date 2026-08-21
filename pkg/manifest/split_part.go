package manifest

import (
	"fmt"

	"github.com/osbuild/image-builder/pkg/artifact"
	"github.com/osbuild/image-builder/pkg/disk"
	"github.com/osbuild/image-builder/pkg/osbuild"
)

// SplitPartImage extracts a single partition from a raw disk image using dd.
type SplitPartImage struct {
	Base
	filename    string
	imgPipeline FilePipeline

	// Mountpoint identifies which partition to extract.
	Mountpoint string

	// PartitionTable is used to resolve the mountpoint to an offset and size.
	PartitionTable *disk.PartitionTable
}

func NewSplitPartImage(buildPipeline Build, imgPipeline FilePipeline, mountpoint string, pt *disk.PartitionTable, name string) *SplitPartImage {
	p := &SplitPartImage{
		Base:           NewBase(name, buildPipeline),
		imgPipeline:    imgPipeline,
		filename:       fmt.Sprintf("%s.img", name),
		Mountpoint:     mountpoint,
		PartitionTable: pt,
	}
	buildPipeline.addDependent(p)
	return p
}

func (p *SplitPartImage) Filename() string {
	return p.filename
}

func (p *SplitPartImage) SetFilename(filename string) {
	p.filename = filename
}

func (p *SplitPartImage) serialize() (osbuild.Pipeline, error) {
	pipeline, err := p.Base.serialize()
	if err != nil {
		return osbuild.Pipeline{}, err
	}

	part, err := FindPartitionByMountpoint(p.PartitionTable, p.Mountpoint)
	if err != nil {
		return osbuild.Pipeline{}, err
	}

	srcOffset := int(part.Start)
	opts := &osbuild.DDStageOptions{
		Src:       fmt.Sprintf("input://image/%s", p.imgPipeline.Filename()),
		Dst:       p.filename,
		SrcOffset: &srcOffset,
		Count:     int(part.Size),
	}

	inputs := osbuild.NewPipelineTreeInputs("image", p.imgPipeline.Name())
	pipeline.AddStage(osbuild.NewDDStage(opts, inputs))

	truncateOpts := &osbuild.TruncateStageOptions{
		Filename: p.filename,
		Size:     fmt.Sprintf("%d", part.Size),
	}
	pipeline.AddStage(osbuild.NewTruncateStage(truncateOpts))

	return pipeline, nil
}

func (p *SplitPartImage) getBuildPackages(Distro) ([]string, error) {
	return nil, nil
}

func (p *SplitPartImage) Export() *artifact.Artifact {
	p.Base.export = true
	return artifact.New(p.Name(), p.Filename(), nil)
}

func FindPartitionByMountpoint(pt *disk.PartitionTable, mountpoint string) (*disk.Partition, error) {
	var found *disk.Partition
	err := pt.ForEachMountable(func(mnt disk.Mountable, path []disk.Entity) error {
		if mnt.GetMountpoint() == mountpoint {
			for _, ent := range path {
				if part, ok := ent.(*disk.Partition); ok {
					found = part
					return nil
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, fmt.Errorf("partition with mountpoint %q not found", mountpoint)
	}
	return found, nil
}
