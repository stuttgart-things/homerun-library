package main

import (
	"context"
	"dagger/dagger/internal/dagger"
)

func (m *Dagger) RunStaticStage(
	ctx context.Context,
	src *dagger.Directory,
	// +optional
	// +default="500s"
	lintTimeout string,
	// +optional
	// +default="1.25.4"
	goVersion string,
	// +optional
	// +default="linux"
	os string,
	// +optional
	// +default="amd64"
	arch string,
	// +optional
	// +default="main.go"
	goMainFile string,
	// +optional
	// +default="main"
	binName string,
	// +optional
	ldflags string,
	// +optional
	// +default="false"
	lintCanFail bool,
	// +optional
	// +default="./..."
	testArg string,
	// +optional
	// +default=true
	lintEnabled bool,
	// +optional
	// +default=true
	test bool,
) *dagger.File {
	return dag.
		GoMicroservice().RunStaticStage(
		src,
		dagger.GoMicroserviceRunStaticStageOpts{
			LintTimeout: lintTimeout,
			GoVersion:   goVersion,
			Os:          os,
			Arch:        arch,
			GoMainFile:  goMainFile,
			BinName:     binName,
			Ldflags:     ldflags,
			LintCanFail: lintCanFail,
			TestArg:     testArg,
			LintEnabled: lintEnabled,
			Test:        test,
		},
	)
}
