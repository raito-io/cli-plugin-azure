package azure

import (
	"context"

	"github.com/raito-io/cli/base/util/config"
	"github.com/raito-io/cli/base/wrappers"
)

//go:generate go run github.com/vektra/mockery/v2 --name=dataUsageRepository --with-expecter --inpackage
type dataUsageRepository interface {
	GetDataUsage(ctx context.Context, configMap *config.ConfigMap) ([]BQInformationSchemaEntity, error)
}

type DataUsageSyncer struct {
	repoProvider func() dataUsageRepository
}

func NewDataUsageSyncer() *DataUsageSyncer {
	return &DataUsageSyncer{repoProvider: newDatausageRepo}
}

func newDatausageRepo() dataUsageRepository {
	return nil
}

func (s *DataUsageSyncer) SyncDataUsage(ctx context.Context, fileCreator wrappers.DataUsageStatementHandler, configParams *config.ConfigMap) error {
	return nil
}
