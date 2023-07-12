package azure

import (
	"context"

	"github.com/raito-io/cli-plugin-azure/azure/storage"
	"github.com/raito-io/cli-plugin-azure/global"
	"github.com/raito-io/cli/base/wrappers"

	importer "github.com/raito-io/cli/base/access_provider/sync_to_target"
	"github.com/raito-io/cli/base/util/config"
)

type AzureServiceDataAccessSyncer interface {
	SyncAccessProvidersFromTarget(ctx context.Context, iamRoleAssignments []global.IAMRoleAssignment, accessProviderHandler wrappers.AccessProviderHandler, configMap *config.ConfigMap) error
	SyncAccessProviderToTarget(ctx context.Context, accessProviders *importer.AccessProviderImport, accessProviderFeedbackHandler wrappers.AccessProviderFeedbackHandler, configMap *config.ConfigMap) error
}

type AccessSyncer struct {
	serviceSyncers []AzureServiceDataAccessSyncer
}

func NewDataAccessSyncer() *AccessSyncer {
	return &AccessSyncer{serviceSyncers: []AzureServiceDataAccessSyncer{
		&storage.DataAccessSyncer{},
	}}
}

func (a *AccessSyncer) SyncAccessProvidersFromTarget(ctx context.Context, accessProviderHandler wrappers.AccessProviderHandler, configMap *config.ConfigMap) error {
	assignments, err := global.GetRoleAssignments(ctx, configMap.Parameters)

	if err != nil {
		return err
	}

	for _, syncer := range a.serviceSyncers {
		err := syncer.SyncAccessProvidersFromTarget(ctx, assignments, accessProviderHandler, configMap)

		if err != nil {
			return err
		}
	}

	return nil
}
func (a *AccessSyncer) SyncAccessProviderToTarget(ctx context.Context, accessProviders *importer.AccessProviderImport, accessProviderFeedbackHandler wrappers.AccessProviderFeedbackHandler, configMap *config.ConfigMap) error {
	for _, syncer := range a.serviceSyncers {
		err := syncer.SyncAccessProviderToTarget(ctx, accessProviders, accessProviderFeedbackHandler, configMap)

		if err != nil {
			return err
		}
	}

	return nil
}

func (a *AccessSyncer) SyncAccessAsCodeToTarget(ctx context.Context, accessProviders *importer.AccessProviderImport, prefix string, configMap *config.ConfigMap) error {
	return nil
}
