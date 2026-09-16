package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/osbuild/image-builder/pkg/experimentalflags"
	"github.com/osbuild/image-builder/pkg/imagefilter"
	"github.com/osbuild/image-builder/pkg/progress"
)

type buildOptions struct {
	OutputDir      string
	StoreDir       string
	OutputBasename string
	InVm           []string
	JSONOutput     bool
	WithExtras     []string

	WriteManifest bool
	WriteBuildlog bool
	Metrics       bool
}

func buildImage(pbar progress.ProgressBar, res *imagefilter.Result, osbuildManifest []byte, opts *buildOptions) (string, error) {
	if opts == nil {
		opts = &buildOptions{}
	}

	basename, err := basenameFor(res, opts.OutputBasename)
	if err != nil {
		return "", err
	}
	if opts.WriteManifest {
		p := filepath.Join(opts.OutputDir, fmt.Sprintf("%s.osbuild-manifest.json", basename))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return "", err
		}
		// #nosec: G306
		if err := os.WriteFile(p, osbuildManifest, 0644); err != nil {
			return "", err
		}
	}

	osbuildOpts := &progress.OSBuildOptions{
		StoreDir:   opts.StoreDir,
		OutputDir:  opts.OutputDir,
		Metrics:    opts.Metrics,
		InVm:       opts.InVm,
		JSONOutput: opts.JSONOutput,
	}
	if opts.WriteBuildlog {
		if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
			return "", fmt.Errorf("cannot create buildlog base directory: %w", err)
		}
		p := filepath.Join(opts.OutputDir, fmt.Sprintf("%s.buildlog", basename))
		f, err := os.Create(p)
		if err != nil {
			return "", fmt.Errorf("cannot create buildlog: %w", err)
		}
		defer f.Close()

		osbuildOpts.BuildLog = f
	}
	exports := res.ImgType.Exports()
	if len(opts.WithExtras) > 0 {
		type extrasExporter interface {
			ExportsWithExtras([]string) ([]string, error)
		}
		ee, ok := res.ImgType.(extrasExporter)
		if !ok {
			return "", fmt.Errorf("image type %q does not support extras", res.ImgType.Name())
		}
		var err error
		exports, err = ee.ExportsWithExtras(opts.WithExtras)
		if err != nil {
			return "", err
		}
	}
	if expExports := experimentalflags.StringSlice("exports"); len(expExports) > 0 {
		var err error
		exports, err = expandExportGlobs(expExports, osbuildManifest)
		if err != nil {
			return "", err
		}
	}

	if err := progress.RunOSBuild(pbar, osbuildManifest, exports, osbuildOpts); err != nil {
		return "", err
	}
	// Rename *sigh*, see https://github.com/osbuild/image-builder/pull/1039
	// for my preferred way. Every frontend to images has to duplicate
	// similar code like this.
	pipelineDir := filepath.Join(opts.OutputDir, exports[0])
	srcName := filepath.Join(pipelineDir, res.ImgType.Filename())
	imgExt := strings.SplitN(res.ImgType.Filename(), ".", 2)[1]
	dstName := filepath.Join(opts.OutputDir, fmt.Sprintf("%s.%v", basename, imgExt))
	if err := os.Rename(srcName, dstName); err != nil {
		return "", fmt.Errorf("cannot rename artifact to final name: %w", err)
	}
	_ = os.Remove(pipelineDir)

	nameData := outputTmplDataFor(res)
	nameTmpl := defaultMultiExportTmpl
	for _, export := range exports[1:] {
		pipelineDir := filepath.Join(opts.OutputDir, export)
		entries, err := os.ReadDir(pipelineDir)
		if err != nil {
			return "", fmt.Errorf("cannot read export directory %q: %w", export, err)
		}
		for _, entry := range entries {
			src := filepath.Join(pipelineDir, entry.Name())
			ext := strings.SplitN(entry.Name(), ".", 2)
			artExt := ""
			if len(ext) > 1 {
				artExt = "." + ext[1]
			}
			nameData.Pipeline.ExportName = ext[0]
			artName, err := expandOutputTmpl(nameTmpl, nameData)
			if err != nil {
				return "", err
			}
			dstName := artName + artExt
			dst := filepath.Join(opts.OutputDir, dstName)
			if err := os.Rename(src, dst); err != nil {
				return "", fmt.Errorf("cannot rename extra artifact %q: %w", entry.Name(), err)
			}
		}
		_ = os.Remove(pipelineDir)
	}

	return dstName, nil
}

func expandExportGlobs(patterns []string, manifest []byte) ([]string, error) {
	var mf struct {
		Pipelines []struct {
			Name string `json:"name"`
		} `json:"pipelines"`
	}
	if err := json.Unmarshal(manifest, &mf); err != nil {
		return nil, fmt.Errorf("cannot parse manifest for export glob expansion: %w", err)
	}

	var names []string
	for _, p := range mf.Pipelines {
		names = append(names, p.Name)
	}

	seen := make(map[string]bool)
	var exports []string
	for _, pattern := range patterns {
		var matched bool
		for _, name := range names {
			ok, err := filepath.Match(pattern, name)
			if err != nil {
				return nil, fmt.Errorf("invalid export glob %q: %w", pattern, err)
			}
			if ok {
				matched = true
				if !seen[name] {
					seen[name] = true
					exports = append(exports, name)
				}
			}
		}
		if !matched {
			return nil, fmt.Errorf("export pattern %q did not match any pipeline", pattern)
		}
	}
	return exports, nil
}
