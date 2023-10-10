package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/aws/smithy-go/ptr"
	"github.com/raito-io/cli/base/data_source"
	"github.com/raito-io/cli/base/wrappers"

	"github.com/raito-io/cli-plugin-azure/global"

	"github.com/raito-io/cli/base/access_provider/sync_from_target"
	importer "github.com/raito-io/cli/base/access_provider/sync_to_target"
	"github.com/raito-io/cli/base/util/config"
)

type DataAccessSyncer struct {
	raitoManagedBindings []global.IAMRoleAssignment
}

func (a *DataAccessSyncer) SyncAccessProvidersFromTarget(ctx context.Context, iamRoleAssignments []global.IAMRoleAssignment, accessProviderHandler wrappers.AccessProviderHandler, configMap *config.ConfigMap) error {
	apMap := make(map[string]*sync_from_target.AccessProvider)

	for _, assignment := range iamRoleAssignments {
		if assignment.PrincipalType != armauthorization.PrincipalTypeGroup && assignment.PrincipalType != armauthorization.PrincipalTypeUser {
			continue
		}

		raitoManaged := false

		for _, rm := range a.raitoManagedBindings {
			if rm.PrincipalId == assignment.PrincipalId && rm.Scope == assignment.Scope && rm.RoleDefinitionID == assignment.RoleDefinitionID {
				raitoManaged = true
			}
		}

		if raitoManaged {
			continue
		}

		apName := ""

		scopeSplit := strings.Split(assignment.Scope, "/")

		doType := ""
		doFullname := ""

		if len(scopeSplit) == 3 {
			apName = fmt.Sprintf("subscription-%s", strings.ReplaceAll(assignment.RoleName, " ", "-"))
			doType = "subscription"
			doFullname = configMap.GetStringWithDefault(global.AzSubscriptionId, "")
		} else {
			doType = strings.ToLower(scopeSplit[len(scopeSplit)-2])
			doType = doType[0 : len(doType)-1]

			doFullname = strings.Join(scopeSplit[2:], "/")
			doFullname = strings.Replace(doFullname, "/providers/Microsoft.Storage/", "", 1)
			doFullname = strings.Replace(doFullname, "resourceGroups", "", 1)
			doFullname = strings.Replace(doFullname, "resourcegroups", "", 1)
			doFullname = strings.Replace(doFullname, "storageAccounts", "", 1)
			doFullname = strings.Replace(doFullname, "storageaccounts", "", 1)
			doFullname = strings.Replace(doFullname, "blobServices/default/containers", "", 1)
			doFullname = strings.Replace(doFullname, "containers", "", 1)
			doFullname = strings.Replace(doFullname, "//", "/", -1)

			apName = fmt.Sprintf("%s-%s-%s", doType, scopeSplit[len(scopeSplit)-1], strings.ReplaceAll(assignment.RoleName, " ", "-"))
		}

		logger.Debug(fmt.Sprintf("Rewrite scope: %q to doFullName: %q", assignment.Scope, doFullname))

		if _, f := apMap[apName]; !f {
			apMap[apName] = &sync_from_target.AccessProvider{
				ExternalId: apName,
				Name:       apName,
				NamingHint: apName,
				ActualName: apName,
				Action:     sync_from_target.Grant,
				Type:       ptr.String(RoleAssignments),
				Who: &sync_from_target.WhoItem{
					Users:  []string{},
					Groups: []string{},
				},
				What: []sync_from_target.WhatItem{{
					Permissions: []string{assignment.RoleName},
					DataObject: &data_source.DataObjectReference{
						Type:     doType,
						FullName: doFullname,
					},
				}},
			}
		}

		if assignment.PrincipalType == armauthorization.PrincipalTypeGroup {
			apMap[apName].Who.Groups = append(apMap[apName].Who.Groups, assignment.PrincipalId)
		} else if assignment.PrincipalType == armauthorization.PrincipalTypeUser {
			apMap[apName].Who.Users = append(apMap[apName].Who.Users, assignment.PrincipalId)
		}
	}

	for _, v := range apMap {
		err := accessProviderHandler.AddAccessProviders(v)
		if err != nil {
			return err
		}
	}

	return nil
}
func (a *DataAccessSyncer) SyncAccessProviderToTarget(ctx context.Context, accessProviders *importer.AccessProviderImport, accessProviderFeedbackHandler wrappers.AccessProviderFeedbackHandler, configMap *config.ConfigMap) error {
	for _, ap := range accessProviders.AccessProviders {
		bindings_to_add, bindings_to_remove := convertAccessProviderToIamRoleAssignments(ctx, ap, configMap.Parameters)

		for _, b := range bindings_to_remove {
			err := global.DeleteRoleAssignment(ctx, configMap.Parameters, b)

			if err != nil {
				return err
			}
		}

		for _, b := range bindings_to_add {
			a.raitoManagedBindings = append(a.raitoManagedBindings, b)

			err := global.CreateRoleAssignment(ctx, configMap.Parameters, b)

			if err != nil {
				return err
			}
		}

		err := accessProviderFeedbackHandler.AddAccessProviderFeedback(ap.Id, importer.AccessSyncFeedbackInformation{
			AccessId:   ap.Id,
			ActualName: ap.NamingHint,
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// return value 1: bindings to create, 2: bindings to delete
func convertAccessProviderToIamRoleAssignments(ctx context.Context, accessProvider *importer.AccessProvider, params map[string]string) ([]global.IAMRoleAssignment, []global.IAMRoleAssignment) {
	dsSync := DataSourceSyncer{}

	bindings := make([][]global.IAMRoleAssignment, 2)

	for i := range bindings {
		bindings[i] = make([]global.IAMRoleAssignment, 0)
		whatList := accessProvider.What

		if i == 1 {
			whatList = accessProvider.DeleteWhat
		}

		for _, what := range whatList {
			scope := ""
			fullNameParts := strings.Split(what.DataObject.FullName, "/")

			switch what.DataObject.Type {
			case "datasource":
				scope = "/"
			case "subscription":
				scope = fmt.Sprintf("/subscriptions/%s", fullNameParts[0])
			case "resourcegroup":
				if len(fullNameParts) < 2 {
					break
				}
				scope = fmt.Sprintf("/subscriptions/%s/resourcegroups/%s", fullNameParts[0], fullNameParts[1])
			case "storageaccount":
				if len(fullNameParts) < 3 {
					break
				}
				scope = fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.Storage/storageAccounts/%s", fullNameParts[0], fullNameParts[1], fullNameParts[2])
			case "container":
				if len(fullNameParts) < 4 {
					break
				}
				scope = fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/default/containers/%s", fullNameParts[0], fullNameParts[1], fullNameParts[2], fullNameParts[3])
			}

			if scope == "" {
				continue
			}

			for _, permission := range what.Permissions {
				if !dsSync.IsApplicablePermission(context.Background(), what.DataObject.Type, permission) {
					continue
				}

				permissionId, err := global.GetRoleIdByName(ctx, params, permission)

				if err != nil {
					logger.Error("Something went wrong while converting accessProvider to azure IAM assignment", err.Error())
					continue
				}

				if permissionId == nil {
					logger.Error(fmt.Sprintf("Permission %q could not be converted into a RoleDefinitionId", permission))
					continue
				}

				for _, u := range accessProvider.Who.Users {
					bindings[i] = append(bindings[i], global.IAMRoleAssignment{
						Scope:            scope,
						RoleName:         permission,
						RoleDefinitionID: *permissionId,
						PrincipalType:    armauthorization.PrincipalTypeUser,
						PrincipalId:      global.GetPrincipalIdByName(ctx, params, armauthorization.PrincipalTypeUser, u),
					})
				}

				for _, g := range accessProvider.Who.Groups {
					bindings[i] = append(bindings[i], global.IAMRoleAssignment{
						Scope:            scope,
						RoleName:         permission,
						RoleDefinitionID: *permissionId,
						PrincipalType:    armauthorization.PrincipalTypeGroup,
						PrincipalId:      global.GetPrincipalIdByName(ctx, params, armauthorization.PrincipalTypeGroup, g),
					})
				}
			}
		}
	}

	if accessProvider.Delete {
		return []global.IAMRoleAssignment{}, append(bindings[1], bindings[0]...)
	}

	return bindings[0], bindings[1]
}
