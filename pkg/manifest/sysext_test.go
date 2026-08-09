package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/image-builder/pkg/arch"
	"github.com/osbuild/image-builder/pkg/depsolvednf"
	"github.com/osbuild/image-builder/pkg/manifest"
	"github.com/osbuild/image-builder/pkg/osbuild"
	"github.com/osbuild/image-builder/pkg/platform"
	"github.com/osbuild/image-builder/pkg/rpmmd"
	"github.com/osbuild/image-builder/pkg/runner"
)

func newTestSysextTree(t *testing.T) (*manifest.SysextTree, manifest.Build) {
	t.Helper()
	mani := manifest.New()
	r := &runner.Linux{}
	build := manifest.NewBuild(&mani, r, nil, nil)
	pf := &platform.Data{Arch: arch.ARCH_X86_64}
	base := manifest.NewSysextTree(build, pf, nil, "test")
	return base, build
}

func testInputs() manifest.Inputs {
	repo := rpmmd.RepoConfig{Id: "test-repo"}
	return manifest.Inputs{
		Depsolved: depsolvednf.DepsolveResult{
			Transactions: depsolvednf.TransactionList{
				{
					{
						Name:     "pkg1",
						Checksum: rpmmd.Checksum{Type: "sha256", Value: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
						RepoID:   repo.Id,
						Repo:     &repo,
					},
				},
			},
			Repos: []rpmmd.RepoConfig{repo},
		},
	}
}

func TestNewSysextTreeName(t *testing.T) {
	base, _ := newTestSysextTree(t)
	assert.Equal(t, "sysext-test-tree", base.Name())
}

func TestSysextTreeSerialize(t *testing.T) {
	base, _ := newTestSysextTree(t)
	pipeline, err := manifest.SerializeWith(base, testInputs())
	require.NoError(t, err)

	assert.Equal(t, "sysext-test-tree", pipeline.Name)
	require.Len(t, pipeline.Stages, 1)
	assert.Equal(t, "org.osbuild.rpm", pipeline.Stages[0].Type)
}

func TestSysextTreeSerializeBaseRPMOptions(t *testing.T) {
	base, _ := newTestSysextTree(t)
	base.Customizations.BaseRPMOptions = &osbuild.RPMStageOptions{
		Exclude:      &osbuild.Exclude{Docs: true},
		InstallLangs: []string{"en_US"},
	}
	pipeline, err := manifest.SerializeWith(base, testInputs())
	require.NoError(t, err)

	require.Len(t, pipeline.Stages, 1)
	opts := pipeline.Stages[0].Options.(*osbuild.RPMStageOptions)
	require.NotNil(t, opts.Exclude)
	assert.True(t, opts.Exclude.Docs)
	assert.Equal(t, []string{"en_US"}, opts.InstallLangs)
}

func TestSysextTreeSerializeNotStarted(t *testing.T) {
	base, _ := newTestSysextTree(t)
	_, err := manifest.Serialize(base)
	assert.Error(t, err, "serialize without serializeStart should fail")
}

func TestSysextTreeGetPackageSetChainEmpty(t *testing.T) {
	base, _ := newTestSysextTree(t)
	chain, err := base.GetPackageSetChain(0)
	require.NoError(t, err)
	assert.Nil(t, chain)
}

func TestSysextTreeGetPackageSetChainWithPackages(t *testing.T) {
	base, _ := newTestSysextTree(t)
	base.Customizations.PackageSet = rpmmd.PackageSet{
		Include: []string{"vim", "git"},
	}
	chain, err := base.GetPackageSetChain(0)
	require.NoError(t, err)
	require.Len(t, chain, 1)
	assert.Equal(t, []string{"vim", "git"}, chain[0].Include)
	assert.False(t, chain[0].InstallWeakDeps)
}

func TestNewSysextPrepName(t *testing.T) {
	mani := manifest.New()
	r := &runner.Linux{}
	build := manifest.NewBuild(&mani, r, nil, nil)
	pf := &platform.Data{Arch: arch.ARCH_X86_64}

	refPipeline := manifest.NewOS(build, pf, nil)
	overlayPipeline := manifest.NewSysextTree(build, pf, nil, "ext")

	final := manifest.NewSysextPrep(build, refPipeline, overlayPipeline, "ext")
	assert.Equal(t, "sysext-ext-prep", final.Name())
}

func TestSysextPrepSerialize(t *testing.T) {
	mani := manifest.New()
	r := &runner.Linux{}
	build := manifest.NewBuild(&mani, r, nil, nil)
	pf := &platform.Data{Arch: arch.ARCH_X86_64}

	refPipeline := manifest.NewOS(build, pf, nil)
	overlayPipeline := manifest.NewSysextTree(build, pf, nil, "ext")

	final := manifest.NewSysextPrep(build, refPipeline, overlayPipeline, "ext")
	final.Customizations.Paths = []string{"/usr", "/opt"}
	final.Customizations.ExtensionRelease.ID = "fedora"
	final.Customizations.ExtensionRelease.VersionID = "44"
	pipeline, err := manifest.Serialize(final)
	require.NoError(t, err)

	assert.Equal(t, "sysext-ext-prep", pipeline.Name)
	require.True(t, len(pipeline.Stages) >= 2, "expected tree-delta and extension-release stages")

	assert.Equal(t, "org.osbuild.tree-delta", pipeline.Stages[0].Type)
	inputs := pipeline.Stages[0].Inputs.(*osbuild.TreeDeltaStageInputs)
	assert.NotNil(t, inputs.Reference)
	assert.NotNil(t, inputs.Overlay)
	opts := pipeline.Stages[0].Options.(*osbuild.TreeDeltaStageOptions)
	assert.Equal(t, []string{"/usr", "/opt"}, opts.Paths)
	assert.Nil(t, opts.ExcludePaths)

	inline := manifest.GetInline(final)
	require.NotEmpty(t, inline, "expected inline data from extension-release file")
	assert.Contains(t, inline[0], "ID=fedora")
	assert.Contains(t, inline[0], "VERSION_ID=44")
}

func TestSysextPrepSerializeWithExcludePaths(t *testing.T) {
	mani := manifest.New()
	r := &runner.Linux{}
	build := manifest.NewBuild(&mani, r, nil, nil)
	pf := &platform.Data{Arch: arch.ARCH_X86_64}

	refPipeline := manifest.NewOS(build, pf, nil)
	overlayPipeline := manifest.NewSysextTree(build, pf, nil, "ext")

	final := manifest.NewSysextPrep(build, refPipeline, overlayPipeline, "ext")
	final.Customizations.Paths = []string{"/usr"}
	final.Customizations.ExcludePaths = []string{"/usr/share/doc", "/usr/share/man"}
	pipeline, err := manifest.Serialize(final)
	require.NoError(t, err)

	opts := pipeline.Stages[0].Options.(*osbuild.TreeDeltaStageOptions)
	assert.Equal(t, []string{"/usr"}, opts.Paths)
	assert.Equal(t, []string{"/usr/share/doc", "/usr/share/man"}, opts.ExcludePaths)
}

func TestSysextPrepExtensionRelease(t *testing.T) {
	newPrep := func(name string) *manifest.SysextPrep {
		mani := manifest.New()
		r := &runner.Linux{}
		build := manifest.NewBuild(&mani, r, nil, nil)
		pf := &platform.Data{Arch: arch.ARCH_X86_64}
		ref := manifest.NewOS(build, pf, nil)
		overlay := manifest.NewSysextTree(build, pf, nil, name)
		return manifest.NewSysextPrep(build, ref, overlay, name)
	}

	t.Run("both fields", func(t *testing.T) {
		prep := newPrep("nginx")
		prep.Customizations.ExtensionRelease.ID = "fedora"
		prep.Customizations.ExtensionRelease.VersionID = "44"
		pipeline, err := manifest.Serialize(prep)
		require.NoError(t, err)

		inline := manifest.GetInline(prep)
		require.Len(t, inline, 1)
		assert.Equal(t, "ID=fedora\nVERSION_ID=44\n", inline[0])

		hasFile := false
		for _, s := range pipeline.Stages {
			if s.Type == "org.osbuild.copy" {
				hasFile = true
			}
		}
		assert.True(t, hasFile, "expected a copy stage for the extension-release file")
	})

	t.Run("only ID", func(t *testing.T) {
		prep := newPrep("test")
		prep.Customizations.ExtensionRelease.ID = "rhel"
		_, err := manifest.Serialize(prep)
		require.NoError(t, err)

		inline := manifest.GetInline(prep)
		require.Len(t, inline, 1)
		assert.Equal(t, "ID=rhel\n", inline[0])
	})

	t.Run("empty", func(t *testing.T) {
		prep := newPrep("empty")
		_, err := manifest.Serialize(prep)
		require.NoError(t, err)

		inline := manifest.GetInline(prep)
		require.Len(t, inline, 1)
		assert.Equal(t, "", inline[0])
	})
}

func TestNewSysextPipelines(t *testing.T) {
	mani := manifest.New()
	r := &runner.Linux{}
	build := manifest.NewBuild(&mani, r, nil, nil)
	pf := &platform.Data{Arch: arch.ARCH_X86_64}

	refPipeline := manifest.NewOS(build, pf, nil)
	pipelines := manifest.NewSysextPipelines(build, pf, nil, refPipeline, "myext")

	assert.Equal(t, "sysext-myext-tree", pipelines.Tree.Name())
	assert.Equal(t, "sysext-myext-prep", pipelines.Prep.Name())
}
