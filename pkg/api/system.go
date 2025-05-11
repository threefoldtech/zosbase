package api

import (
	"context"
	"os/exec"
	"strings"

	"github.com/threefoldtech/zosbase/pkg"
	"github.com/threefoldtech/zosbase/pkg/capacity/dmi"
	"github.com/threefoldtech/zosbase/pkg/diagnostics"
)

func (a *API) SystemVersion(ctx context.Context) (Version, error) {
	output, err := exec.CommandContext(ctx, "zinit", "-V").CombinedOutput()
	var zInitVer string
	if err != nil {
		zInitVer = err.Error()
	} else {
		zInitVer = strings.TrimSpace(strings.TrimPrefix(string(output), "zinit"))
	}

	version := Version{
		Zos:   a.versionMonitorStub.GetVersion(ctx).String(),
		Zinit: zInitVer,
	}

	return version, nil
}

func (a *API) SystemDMI(ctx context.Context) (dmi.DMI, error) {
	dmi, err := a.oracle.DMI()
	return *dmi, err
}

func (a *API) SystemHypervisor(ctx context.Context) (string, error) {
	return a.oracle.GetHypervisor()
}

func (a *API) SystemDiagnostics(ctx context.Context) (diagnostics.Diagnostics, error) {
	return a.diagnosticsManager.GetSystemDiagnostics(ctx)
}

func (a *API) SystemNodeFeatures(ctx context.Context) []pkg.NodeFeature {
	return a.systemMonitorStub.GetNodeFeatures(ctx)
}
