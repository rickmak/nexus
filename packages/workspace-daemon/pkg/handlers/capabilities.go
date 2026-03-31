package handlers

import (
	"context"
	"encoding/json"

	rpckit "github.com/nexus/nexus/packages/workspace-daemon/pkg/rpcerrors"
	"github.com/nexus/nexus/packages/workspace-daemon/pkg/runtime"
)

type CapabilitiesListResult struct {
	Capabilities []runtime.Capability `json:"capabilities"`
}

func HandleCapabilitiesList(_ context.Context, _ json.RawMessage, factory *runtime.Factory) (*CapabilitiesListResult, *rpckit.RPCError) {
	if factory == nil {
		return &CapabilitiesListResult{Capabilities: []runtime.Capability{}}, nil
	}
	return &CapabilitiesListResult{Capabilities: factory.Capabilities()}, nil
}
