package azure

import (
	"context"

	"github.com/raito-io/cli-plugin-azure/azure/storage"
	ds "github.com/raito-io/cli/base/data_source"

	"github.com/raito-io/cli/base/util/config"
	"github.com/raito-io/cli/base/wrappers"
)

type AzureServiceDataObjectSyncer interface {
	SyncDataSource(ctx context.Context, dataSourceHandler wrappers.DataSourceObjectHandler, configMap *config.ConfigMap) error
	GetDataObjectTypes(ctx context.Context) ([]string, []*ds.DataObjectType)
	GetIAMPermissions() []*ds.DataObjectTypePermission
}

type DataSourceSyncer struct {
	serviceSyncers []AzureServiceDataObjectSyncer
}

func NewDataSourceSyncer() *DataSourceSyncer {
	return &DataSourceSyncer{serviceSyncers: []AzureServiceDataObjectSyncer{
		&storage.DataSourceSyncer{},
	}}
}

func (s *DataSourceSyncer) SyncDataSource(ctx context.Context, dataSourceHandler wrappers.DataSourceObjectHandler, configMap *config.ConfigMap) error {
	for _, syncer := range s.serviceSyncers {
		err := syncer.SyncDataSource(ctx, dataSourceHandler, configMap)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *DataSourceSyncer) GetDataSourceMetaData(ctx context.Context) (*ds.MetaData, error) {
	logger.Debug("Returning meta data for Azure data source")

	meta := &ds.MetaData{
		Type:              "azure",
		SupportedFeatures: []string{},
		DataObjectTypes: []*ds.DataObjectType{
			{
				Name:        ds.Datasource,
				Type:        ds.Datasource,
				Permissions: []*ds.DataObjectTypePermission{},
				Children:    []string{},
			},
		},
	}

	for _, syncer := range s.serviceSyncers {
		topLevelDoTypeNames, doTypes := syncer.GetDataObjectTypes(ctx)

		meta.DataObjectTypes = append(meta.DataObjectTypes, doTypes...)

		meta.DataObjectTypes[0].Children = append(meta.DataObjectTypes[0].Children, topLevelDoTypeNames...)

		meta.DataObjectTypes[0].Permissions = append(meta.DataObjectTypes[0].Permissions, syncer.GetIAMPermissions()...)
	}

	return meta, nil
}
