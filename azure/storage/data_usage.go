package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/raito-io/cli-plugin-azure/azure/monitor"
	"github.com/raito-io/cli-plugin-azure/global"
	"github.com/raito-io/cli/base/access_provider/sync_from_target"
	"github.com/raito-io/cli/base/data_source"
	"github.com/raito-io/cli/base/data_usage"
	"github.com/raito-io/cli/base/util/config"
)

type DataUsageSyncer struct {
}

func (s *DataUsageSyncer) SyncDataUsage(ctx context.Context, startDate time.Time, configParams *config.ConfigMap, commit func(st data_usage.Statement) error) error {
	// we use the monitor service to 1. check if logging is enabled on our storage account and 2. extract the logs
	monitorService := monitor.NewMonitorService()

	storageAccountsPerResourceGroup, err := getStorageAccounts(ctx, configParams.GetString(global.AzSubscriptionId), configParams.Parameters)

	if err != nil {
		return err
	}

	for resourceGroup, storageAccounts := range storageAccountsPerResourceGroup {
		for _, storageAccount := range storageAccounts {
			enabled, _ := monitorService.HasLogsEnabled(ctx, configParams, resourceGroup, AzApiNamespace, "storageAccounts", fmt.Sprintf("%s/blobServices/default/", storageAccount))

			if !enabled {
				continue
			}

			query := "StorageBlobLogs | where OperationName == \"GetBlob\" and MetricResponseType == \"Success\" and AuthenticationType == \"OAuth\""

			entries, err2 := monitorService.GetLogs(ctx, configParams, query, startDate, resourceGroup, AzApiNamespace, "storageAccounts", fmt.Sprintf("%s/blobServices/default/", storageAccount))

			if err2 != nil {
				return err2
			}

			for _, rt := range entries {
				accessedResource := sync_from_target.WhatItem{
					DataObject: &data_source.DataObjectReference{
						FullName: strings.Join(strings.Split(rt.ObjectKey, "/")[2:], "/"),
						Type:     "file",
					},
					Permissions: []string{rt.OperationName},
				}

				timeGenerated, err3 := time.Parse(time.RFC3339, rt.TimeGenerated)

				if err3 != nil {
					return err3
				}

				err3 = commit(data_usage.Statement{
					ExternalId:          rt.CorrelationId,
					User:                global.GetPrincipalNameById(ctx, configParams.Parameters, armauthorization.PrincipalTypeUser, rt.RequesterObjectId),
					StartTime:           timeGenerated.Unix(),
					EndTime:             timeGenerated.Unix(),
					AccessedDataObjects: []sync_from_target.WhatItem{accessedResource},
					Success:             true,
				})

				if err3 != nil {
					return err3
				}
			}
		}
	}

	return nil
}
