package azure

import (
	"context"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/go-multierror"
	"github.com/raito-io/cli/base/wrappers"

	"github.com/raito-io/cli-plugin-azure/azure/constants"
	"github.com/raito-io/cli-plugin-azure/azure/storage"
	"github.com/raito-io/cli-plugin-azure/global"

	importer "github.com/raito-io/cli/base/access_provider/sync_to_target"
	"github.com/raito-io/cli/base/util/config"
)

type AzureServiceDataAccessSyncer interface {
	SyncAccessProvidersFromTarget(ctx context.Context, raitoManagedBindings []global.IAMRoleAssignment, iamRoleAssignments []global.IAMRoleAssignment, accessProviderHandler wrappers.AccessProviderHandler, configMap *config.ConfigMap) error
	SyncAccessProvidersToTarget(ctx context.Context, accessProviders []*importer.AccessProvider, feedbackHandler global.AccessProviderFeedbackHandler, configMap *config.ConfigMap) error
}

var _ global.AccessProviderFeedbackHandler = (*apFeedbackHandler)(nil)

type apFeedbackHandler struct {
	feedbackObjects map[string]importer.AccessProviderSyncFeedback
}

func newApFeedbackHandler() *apFeedbackHandler {
	return &apFeedbackHandler{
		feedbackObjects: make(map[string]importer.AccessProviderSyncFeedback),
	}
}

func (a apFeedbackHandler) Error(err string, apIds ...string) {
	for _, apId := range apIds {
		if ap, found := a.feedbackObjects[apId]; found {
			ap.Errors = append(ap.Errors, err)
		}
	}
}

func (a apFeedbackHandler) Warning(warning string, apIds ...string) {
	for _, apId := range apIds {
		if ap, found := a.feedbackObjects[apId]; found {
			ap.Warnings = append(ap.Warnings, warning)
		}
	}
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
func (a *AccessSyncer) SyncAccessProviderToTarget(ctx context.Context, accessProviders *importer.AccessProviderImport, accessProviderFeedbackHandler wrappers.AccessProviderFeedbackHandler, configMap *config.ConfigMap) (err error) {
	feedbackObjects := newApFeedbackHandler()

	for _, ap := range accessProviders.AccessProviders {
		fo := importer.AccessProviderSyncFeedback{
			AccessProvider: ap.Id,
			ActualName:     ap.Id,
			Type:           ptr.String(constants.RoleAssignments),
		}

		feedbackObjects.feedbackObjects[ap.Id] = fo
	}

	defer func() {
		for _, feedbackObject := range feedbackObjects.feedbackObjects {
			fErr := accessProviderFeedbackHandler.AddAccessProviderFeedback(feedbackObject)
			if fErr != nil {
				err = multierror.Append(err, fErr)
			}
		}
	}()

	for _, syncer := range a.serviceSyncers {
		err := syncer.SyncAccessProvidersToTarget(ctx, accessProviders.AccessProviders, feedbackObjects, configMap)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *AccessSyncer) SyncAccessAsCodeToTarget(ctx context.Context, accessProviders *importer.AccessProviderImport, prefix string, configMap *config.ConfigMap) error {
	return nil
}
