package azure

import (
	"context"

	"github.com/aws/smithy-go/ptr"
	"github.com/raito-io/cli/base/wrappers"
	"github.com/raito-io/golang-set/set"

	"github.com/raito-io/cli-plugin-azure/azure/constants"
	"github.com/raito-io/cli-plugin-azure/azure/storage"
	"github.com/raito-io/cli-plugin-azure/global"

	importer "github.com/raito-io/cli/base/access_provider/sync_to_target"
	"github.com/raito-io/cli/base/util/config"
)

type AzureServiceDataAccessSyncer interface {
	SyncAccessProvidersFromTarget(ctx context.Context, raitoManagedBindings []global.IAMRoleAssignment, iamRoleAssignments []global.IAMRoleAssignment, accessProviderHandler wrappers.AccessProviderHandler, configMap *config.ConfigMap) error
	ConvertAccessProviderToIamRoleAssignments(ctx context.Context, accessProvider *importer.AccessProvider, configMap *config.ConfigMap) (bindings_to_add []global.IAMRoleAssignment, bindings_to_remove []global.IAMRoleAssignment, err error)
}

type AccessSyncer struct {
	serviceSyncers []AzureServiceDataAccessSyncer

	raitoManagedBindings []global.IAMRoleAssignment
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
		err := syncer.SyncAccessProvidersFromTarget(ctx, a.raitoManagedBindings, assignments, accessProviderHandler, configMap)

		if err != nil {
			return err
		}
	}

	return nil
}
func (a *AccessSyncer) SyncAccessProviderToTarget(ctx context.Context, accessProviders *importer.AccessProviderImport, accessProviderFeedbackHandler wrappers.AccessProviderFeedbackHandler, configMap *config.ConfigMap) error {
	bindingsToAdd := set.NewSet[global.IAMRoleAssignment]()
	bindingsToRemove := set.NewSet[global.IAMRoleAssignment]()
	bindingApMap := map[global.IAMRoleAssignment][]string{}

	feedbackObjects := make(map[string]importer.AccessProviderSyncFeedback)

	for _, ap := range accessProviders.AccessProviders {
		fo := importer.AccessProviderSyncFeedback{
			AccessProvider: ap.Id,
			ActualName:     ap.Id,
			Type:           ptr.String(constants.RoleAssignments),
		}

		for _, syncer := range a.serviceSyncers {
			apSyncerBindingsToAdd, apSyncerBindingsToRemove, err := syncer.ConvertAccessProviderToIamRoleAssignments(ctx, ap, configMap)
			if err != nil {
				fo.Errors = append(fo.Errors, err.Error())
			}

			bindingsToAdd.Add(apSyncerBindingsToAdd...)
			bindingsToRemove.Add(apSyncerBindingsToRemove...)

			for _, binding := range apSyncerBindingsToAdd {
				bindingApMap[binding] = append(bindingApMap[binding], ap.Id)
			}

			for _, binding := range apSyncerBindingsToRemove {
				bindingApMap[binding] = append(bindingApMap[binding], ap.Id)
			}
		}

		feedbackObjects[ap.Id] = fo
	}

	bindingsToRemove.RemoveAll(bindingsToAdd.Slice()...)

	for binding := range bindingsToRemove {
		err := global.DeleteRoleAssignment(ctx, configMap.Parameters, binding)
		if err != nil {
			handleRoleAssignmentError(bindingApMap, feedbackObjects, binding, err)
		}
	}

	for binding := range bindingsToAdd {
		a.raitoManagedBindings = append(a.raitoManagedBindings, binding)

		err := global.CreateRoleAssignment(ctx, configMap.Parameters, binding)
		if err != nil {
			handleRoleAssignmentError(bindingApMap, feedbackObjects, binding, err)
		}
	}

	return nil
}

func handleRoleAssignmentError(bindingApMap map[global.IAMRoleAssignment][]string, feedbackObjects map[string]importer.AccessProviderSyncFeedback, binding global.IAMRoleAssignment, err error) {
	for _, apId := range bindingApMap[binding] {
		fo := feedbackObjects[apId]
		fo.Errors = append(fo.Errors, err.Error())
		feedbackObjects[apId] = fo
	}
}

func (a *AccessSyncer) SyncAccessAsCodeToTarget(ctx context.Context, accessProviders *importer.AccessProviderImport, prefix string, configMap *config.ConfigMap) error {
	return nil
}
