package registry

import (
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
	"github.com/azusachino/iroha/apps/iroha-providers/applehealth"
	"github.com/azusachino/iroha/apps/iroha-providers/gpx"
)

func New() (*provider.Registry, error) {
	return provider.NewRegistry(applehealth.New(), gpx.New())
}
