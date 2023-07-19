package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/raito-io/cli-plugin-azure/global"
	"github.com/raito-io/cli/base"
	"github.com/raito-io/cli/base/util/config"
)

//go:generate go run github.com/vektra/mockery/v2 --name=MonitorService --with-expecter --inpackage
type MonitorService interface {
	HasLogsEnabled(ctx context.Context, configMap *config.ConfigMap, resourceGroup, nameSpace, resourceType, resourceName string) (bool, error)
	GetResourceDiagnosticSetting(ctx context.Context, configMap *config.ConfigMap, resourceGroup, nameSpace, resourceType, resourceName string) (*ResourceDiagnosticSetting, error)
	GetLogs(ctx context.Context, configMap *config.ConfigMap, query string, startDate time.Time, resourceGroup, nameSpace, resourceType, resourceName string) ([]LogEntry, error)
}

type monitorService struct {
	resourceDiagSettings map[string]*ResourceDiagnosticSetting
}

func NewMonitorService() MonitorService {
	return &monitorService{resourceDiagSettings: make(map[string]*ResourceDiagnosticSetting)}
}

func (m *monitorService) HasLogsEnabled(ctx context.Context, configMap *config.ConfigMap, resourceGroup, nameSpace, resourceType, resourceName string) (bool, error) {
	setting, err := m.GetResourceDiagnosticSetting(ctx, configMap, resourceGroup, nameSpace, resourceType, resourceName)
	if err != nil {
		return false, err
	}

	if setting == nil {
		return false, nil
	}

	return setting.ReadLogsEnabled, nil
}

func (m *monitorService) GetResourceDiagnosticSetting(ctx context.Context, configMap *config.ConfigMap, resourceGroup, nameSpace, resourceType, resourceName string) (*ResourceDiagnosticSetting, error) {
	resourceURI := getResourceUri(configMap.GetString(global.AzSubscriptionId), resourceGroup, nameSpace, resourceType, resourceName)

	if s, f := m.resourceDiagSettings[resourceURI]; f {
		return s, nil
	}

	client, err := createDiagnosticsSettingsClient(ctx, configMap.Parameters)

	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(resourceURI, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Value {
			if v.Properties.WorkspaceID == nil {
				continue
			}

			split := strings.Split(*v.Properties.WorkspaceID, "/")

			setting := ResourceDiagnosticSetting{
				WorkspaceID:     split[len(split)-1],
				Resource:        resourceURI,
				ReadLogsEnabled: false,
			}

			for _, log := range v.Properties.Logs {
				if log == nil || !*log.Enabled {
					continue
				}

				// category for azure blob storage is "StorageRead" we assume here that all other services will have read in their category, to be verified!
				if strings.Contains(strings.ToLower(*log.Category), "read") {
					setting.ReadLogsEnabled = true
					break
				}
			}

			m.resourceDiagSettings[resourceURI] = &setting
		}
	}

	return m.resourceDiagSettings[resourceURI], nil
}

func (m *monitorService) GetLogs(ctx context.Context, configMap *config.ConfigMap, query string, startDate time.Time, resourceGroup, nameSpace, resourceType, resourceName string) ([]LogEntry, error) {
	client, err := createAzQueryLogsClient(ctx, configMap.Parameters)

	if err != nil {
		return nil, err
	}

	interval := azquery.NewTimeInterval(startDate, time.Now())

	resp, err := client.QueryResource(ctx, getResourceUri(configMap.GetString(global.AzSubscriptionId), resourceGroup, nameSpace, resourceType, resourceName), azquery.Body{
		Query:    &query,
		Timespan: &interval,
	}, nil)

	if err != nil {
		base.Logger().Error(err.Error())
		return nil, err
	}

	entries := make([]LogEntry, 0)

	for _, table := range resp.Tables {
		for _, row := range table.Rows {
			entries = append(entries, LogEntry{}.FromRow(row, table.Columns))
		}
	}

	return entries, nil
}

func getResourceUri(subscription, resourceGroup, nameSpace, resourceType, resourceName string) string {
	return fmt.Sprintf("subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s", subscription, resourceGroup, nameSpace, resourceType, resourceName)
}
