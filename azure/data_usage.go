package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/raito-io/cli-plugin-azure/azure/storage"
	"github.com/raito-io/cli/base/data_usage"
	"github.com/raito-io/cli/base/util/config"
	"github.com/raito-io/cli/base/wrappers"
)

//go:generate go run github.com/vektra/mockery/v2 --name=AzureServiceDataUsageSyncer --with-expecter --inpackage
type AzureServiceDataUsageSyncer interface {
	SyncDataUsage(ctx context.Context, startDate time.Time, configParams *config.ConfigMap, commit func(st data_usage.Statement) error) error
}

type DataUsageSyncer struct {
	serviceSyncers []AzureServiceDataUsageSyncer
}

func NewDataUsageSyncer() *DataUsageSyncer {
	return &DataUsageSyncer{serviceSyncers: []AzureServiceDataUsageSyncer{
		&storage.DataUsageSyncer{},
	}}
}

func (s *DataUsageSyncer) SyncDataUsage(ctx context.Context, fileCreator wrappers.DataUsageStatementHandler, configParams *config.ConfigMap) error {
	startDate, _, _ := GetDataUsageStartDate(ctx, configParams)
	loggingThreshold := uint64(1 * 1024 * 1024)
	maximumFileSize := uint64(2 * 1024 * 1024 * 1024) // TODO: temporary limit of ~2Gb for debugging
	numStatements := 0

	for _, syncer := range s.serviceSyncers {
		err := syncer.SyncDataUsage(ctx, startDate, configParams, func(st data_usage.Statement) error {
			fileSize := fileCreator.GetImportFileSize()
			if fileSize > loggingThreshold {
				logger.Info(fmt.Sprintf("Import file size larger than %d bytes after %d statements => ~%.1f bytes/statement", fileSize, numStatements, float32(fileSize)/float32(numStatements)))
				loggingThreshold = 10 * loggingThreshold
			}

			if fileSize > maximumFileSize {
				logger.Warn(fmt.Sprintf("Current data usage file size larger than %d bytes(%d statements), not adding any more data to import", maximumFileSize, numStatements))
			} else {
				err := fileCreator.AddStatements([]data_usage.Statement{st})

				if err != nil {
					return err
				}

				numStatements += 1
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}
