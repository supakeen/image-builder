package manifest

import (
	"errors"
	"fmt"

	"github.com/osbuild/image-builder/pkg/customizations/fsnode"
	"github.com/osbuild/image-builder/pkg/depsolvednf"
	"github.com/osbuild/image-builder/pkg/osbuild"
	"github.com/osbuild/image-builder/pkg/platform"
	"github.com/osbuild/image-builder/pkg/rpmmd"
)

type SysextExtensionRelease struct {
	ID        string
	VersionID string
}

func (er SysextExtensionRelease) serialize() string {
	var s string
	if er.ID != "" {
		s += "ID=" + er.ID + "\n"
	}
	if er.VersionID != "" {
		s += "VERSION_ID=" + er.VersionID + "\n"
	}
	return s
}

// SysextCustomizations holds options for a sysext image. It is shared across
// the various Sysext* pipeline generators and might be used in different places.
type SysextCustomizations struct {
	PackageSet       rpmmd.PackageSet
	Paths            []string
	ExcludePaths     []string
	ExtensionRelease SysextExtensionRelease
	BaseRPMOptions   *osbuild.RPMStageOptions
}

// SysextTree installs packages into a tree that serves as the overlay for a
// systemd system extension.
type SysextTree struct {
	Base

	// Necessary plumbing values
	depsolveRepos  []rpmmd.RepoConfig
	depsolveResult *depsolvednf.DepsolveResult

	platform platform.Platform

	Customizations SysextCustomizations
}

func NewSysextTree(buildPipeline Build, platform platform.Platform, repos []rpmmd.RepoConfig, name string) *SysextTree {
	pipelineName := "sysext-" + name + "-tree"
	p := &SysextTree{
		Base:          NewBase(pipelineName, buildPipeline),
		depsolveRepos: filterRepos(repos, pipelineName),
		platform:      platform,
	}
	buildPipeline.addDependent(p)
	return p
}

func (p *SysextTree) serializeStart(inputs Inputs) error {
	if p.depsolveResult != nil {
		return errors.New("SysextTree: double call to serializeStart()")
	}

	p.depsolveResult = &inputs.Depsolved

	return nil
}

func (p *SysextTree) serializeEnd() {
	if p.depsolveResult == nil {
		panic("serializeEnd() call when serialization not in progress")
	}
	p.depsolveResult = nil
}

func (p *SysextTree) serialize() (osbuild.Pipeline, error) {
	if p.depsolveResult == nil {
		return osbuild.Pipeline{}, fmt.Errorf("serialization not started")
	}

	pipeline, err := p.Base.serialize()
	if err != nil {
		return osbuild.Pipeline{}, err
	}

	// Do not copy the reference pipeline tree here; it causes issues with the RPM database and package selections.
	rpmStages, err := osbuild.GenRPMStagesFromTransactions(p.depsolveResult.Transactions, p.Customizations.BaseRPMOptions)
	if err != nil {
		return osbuild.Pipeline{}, err
	}
	pipeline.AddStages(rpmStages...)

	return pipeline, nil
}

func (p *SysextTree) getPackageSetChain(Distro) ([]rpmmd.PackageSet, error) {
	pkgSet := p.Customizations.PackageSet
	if len(pkgSet.Include) > 0 || len(pkgSet.Exclude) > 0 {
		pkgSet.Repositories = p.depsolveRepos
		pkgSet.InstallWeakDeps = false
		return []rpmmd.PackageSet{pkgSet}, nil
	}

	return nil, nil
}

// SysextPrep computes the file-level difference between a reference tree
// (typically the base OS) and an overlay tree (the SysextTree output), then
// writes the extension-release metadata. The result contains only files that
// were added or changed plus the sysext metadata.
type SysextPrep struct {
	Base

	Customizations SysextCustomizations

	inlineData []string

	name string

	referencePipeline Pipeline
	overlayPipeline   Pipeline
}

func NewSysextPrep(
	buildPipeline Build,
	referencePipeline Pipeline,
	overlayPipeline Pipeline,
	name string,
) *SysextPrep {
	p := &SysextPrep{
		Base:              NewBase("sysext-"+name+"-prep", buildPipeline),
		name:              name,
		referencePipeline: referencePipeline,
		overlayPipeline:   overlayPipeline,
	}
	buildPipeline.addDependent(p)
	return p
}

func (p *SysextPrep) serialize() (osbuild.Pipeline, error) {
	pipeline, err := p.Base.serialize()
	if err != nil {
		return osbuild.Pipeline{}, err
	}

	opts := &osbuild.TreeDeltaStageOptions{
		Paths:        p.Customizations.Paths,
		ExcludePaths: p.Customizations.ExcludePaths,
	}
	pipeline.AddStage(osbuild.NewTreeDeltaStage(opts, p.referencePipeline.Name(), p.overlayPipeline.Name()))

	pipeline, err = p.writeExtensionRelease(pipeline)
	if err != nil {
		return osbuild.Pipeline{}, err
	}

	return pipeline, nil
}

func (p *SysextPrep) addStagesForAllFilesAndInlineData(pipeline *osbuild.Pipeline, files []*fsnode.File) {
	pipeline.AddStages(osbuild.GenFileNodesStages(files)...)

	for _, file := range files {
		if file.URI() == "" {
			p.inlineData = append(p.inlineData, string(file.Data()))
		}
	}
}

func (p *SysextPrep) getInline() []string {
	return p.inlineData
}

func (p *SysextPrep) writeExtensionRelease(pipeline osbuild.Pipeline) (osbuild.Pipeline, error) {
	// create and write the sysext metadata, maybe this should be a stage?
	extensionDir, err := fsnode.NewDirectory("/usr/lib/extension-release.d", nil, nil, nil, true)
	if err != nil {
		return osbuild.Pipeline{}, fmt.Errorf("failed to create extension-release.d")
	}

	pipeline.AddStages(osbuild.GenDirectoryNodesStages([]*fsnode.Directory{extensionDir})...)

	extensionFile, err := fsnode.NewFile(
		"/usr/lib/extension-release.d/extension-release."+p.name,
		nil,
		nil,
		nil,
		[]byte(p.Customizations.ExtensionRelease.serialize()),
	)
	if err != nil {
		return osbuild.Pipeline{}, fmt.Errorf("failed to create extension-release.IMAGE")
	}

	p.addStagesForAllFilesAndInlineData(&pipeline, []*fsnode.File{extensionFile})

	return pipeline, nil
}

// SysextPipelines groups the two pipelines that together produce a sysext:
// Tree (install packages), Prep (diff against the OS + extension-release metadata).
type SysextPipelines struct {
	Tree *SysextTree
	Prep *SysextPrep
}

// NewSysextPipelines creates the full Tree -> Prep chain for a named sysext,
// diffed against referencePipeline.
func NewSysextPipelines(buildPipeline Build, platform platform.Platform, repos []rpmmd.RepoConfig, referencePipeline Pipeline, name string) SysextPipelines {
	tree := NewSysextTree(buildPipeline, platform, repos, name)
	prep := NewSysextPrep(buildPipeline, referencePipeline, tree, name)
	return SysextPipelines{
		Tree: tree,
		Prep: prep,
	}
}
