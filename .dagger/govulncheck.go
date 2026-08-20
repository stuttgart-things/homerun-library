package main

import (
	"context"
	"dagger/dagger/internal/dagger"
	"fmt"
)

// Govulncheck reports vulnerabilities from the Go vulnerability database that
// are actually reachable from this module's code. Unlike a dependabot alert it
// only flags a CVE when a vulnerable symbol is genuinely called, so a finding
// here is always worth acting on.
//
// This repository is a library: every finding propagates into all seven
// homerun2 services that import it. main is clean as of v3.1.7, so this runs as
// a hard gate by default (failOnVuln=true) - the point is to keep it clean, not
// to report on it. Pass --fail-on-vuln=false to get the report without the gate.
func (m *Dagger) Govulncheck(
	ctx context.Context,
	src *dagger.Directory,
	// +optional
	// +default="1.26.6"
	goVersion string,
	// +optional
	// +default="latest"
	govulncheckVersion string,
	// +optional
	// +default=true
	failOnVuln bool,
) *dagger.File {
	const reportPath = "/tmp/govulncheck-report.txt"

	// govulncheck exits 3 when it finds something. Capture the report first and
	// always echo it, so the run is readable whether or not it is a hard gate.
	script := fmt.Sprintf(`
set -u
govulncheck ./... > %[1]s 2>&1
code=$?
cat %[1]s
if [ "%[2]t" = "true" ] && [ "$code" -ne 0 ]; then
  echo "govulncheck found vulnerabilities (exit $code)" >&2
  exit "$code"
fi
exit 0
`, reportPath, failOnVuln)

	return dag.Container().
		From("golang:"+goVersion).
		// The golang image pins GOTOOLCHAIN=local; govulncheck and this module
		// may both want a newer toolchain than the image ships.
		WithEnvVariable("GOTOOLCHAIN", "auto").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("gomod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("gobuild")).
		WithDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@" + govulncheckVersion}).
		WithExec([]string{"sh", "-c", script}).
		File(reportPath)
}
