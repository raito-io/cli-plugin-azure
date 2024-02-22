package azure

import (
	"context"

	ds "github.com/raito-io/cli/base/data_source"

	"github.com/raito-io/cli-plugin-azure/azure/constants"
	"github.com/raito-io/cli-plugin-azure/azure/storage"

	"github.com/raito-io/cli/base/util/config"
	"github.com/raito-io/cli/base/wrappers"
)

type AzureServiceDataObjectSyncer interface {
	SyncDataSource(ctx context.Context, dataSourceHandler wrappers.DataSourceObjectHandler, config *ds.DataSourceSyncConfig) error
	GetDataObjectTypes(ctx context.Context) ([]string, []*ds.DataObjectType)
	GetDataSourceIAMPermissions() []*ds.DataObjectTypePermission
}

type DataSourceSyncer struct {
	serviceSyncers []AzureServiceDataObjectSyncer
}

func NewDataSourceSyncer() *DataSourceSyncer {
	return &DataSourceSyncer{serviceSyncers: []AzureServiceDataObjectSyncer{
		&storage.DataSourceSyncer{},
	}}
}

func (s *DataSourceSyncer) SyncDataSource(ctx context.Context, dataSourceHandler wrappers.DataSourceObjectHandler, config *ds.DataSourceSyncConfig) error {
	for _, syncer := range s.serviceSyncers {
		err := syncer.SyncDataSource(ctx, dataSourceHandler, config)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *DataSourceSyncer) GetDataSourceMetaData(ctx context.Context, _ *config.ConfigMap) (*ds.MetaData, error) {
	logger.Debug("Returning meta data for Azure data source")

	meta := &ds.MetaData{
		Type:                  "azure",
		SupportedFeatures:     []string{},
		SupportsApInheritance: false,
		DataObjectTypes: []*ds.DataObjectType{
			{
				Name:        ds.Datasource,
				Type:        ds.Datasource,
				Permissions: []*ds.DataObjectTypePermission{},
				Children:    []string{},
			},
		},
		UsageMetaInfo: &ds.UsageMetaInput{
			DefaultLevel: ds.File,
			Levels: []*ds.UsageMetaInputDetail{
				{
					Name:            ds.File,
					DataObjectTypes: []string{ds.File},
				},
			},
		},
		AccessProviderTypes: []*ds.AccessProviderType{
			{
				Type:          constants.RoleAssignments,
				Label:         "Role Assignment",
				IsNamedEntity: false,
				CanBeCreated:  true,
				CanBeAssumed:  false,
			},
		},
	}

	for _, syncer := range s.serviceSyncers {
		topLevelDoTypeNames, doTypes := syncer.GetDataObjectTypes(ctx)

		meta.DataObjectTypes = append(meta.DataObjectTypes, doTypes...)

		meta.DataObjectTypes[0].Children = append(meta.DataObjectTypes[0].Children, topLevelDoTypeNames...)

		meta.DataObjectTypes[0].Permissions = append(meta.DataObjectTypes[0].Permissions, syncer.GetDataSourceIAMPermissions()...)
	}

	return meta, nil
}
