package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/directory"
	"github.com/aws/smithy-go/ptr"
	"github.com/raito-io/cli/base/data_source"
	"github.com/raito-io/cli/base/wrappers"
	"github.com/raito-io/golang-set/set"

	"github.com/raito-io/cli-plugin-azure/azure/constants"
	"github.com/raito-io/cli-plugin-azure/global"

	"github.com/raito-io/cli/base/access_provider/sync_from_target"
	importer "github.com/raito-io/cli/base/access_provider/sync_to_target"
	"github.com/raito-io/cli/base/util/config"
)

type DataAccessSyncer struct {
}

func (a *DataAccessSyncer) SyncAccessProvidersFromTarget(_ context.Context, raitoManagedBindings []global.IAMRoleAssignment, iamRoleAssignments []global.IAMRoleAssignment, accessProviderHandler wrappers.AccessProviderHandler, configMap *config.ConfigMap) error {
	apMap := make(map[string]*sync_from_target.AccessProvider)

	for _, assignment := range iamRoleAssignments {
		if assignment.PrincipalType != armauthorization.PrincipalTypeGroup && assignment.PrincipalType != armauthorization.PrincipalTypeUser {
			continue
		}

		raitoManaged := false

		for _, rm := range raitoManagedBindings {
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
				Type:       ptr.String(constants.RoleAssignments),
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

func (a *DataAccessSyncer) SyncAccessProvidersToTarget(ctx context.Context, accessProviders []*importer.AccessProvider, feedbackHandler global.AccessProviderFeedbackHandler, configMap *config.ConfigMap) error {
	iamClient, err := global.NewIamClient(ctx, configMap.Parameters)
	if err != nil {
		return err
	}

	roleBindingsToAdd := set.NewSet[global.IAMRoleAssignment]()
	roleBindingsToRemove := set.NewSet[global.IAMRoleAssignment]()
	roleBindingApMap := map[global.IAMRoleAssignment][]string{}

	aclAssignments := make(ACLAssignmentsWithAP)

	for _, ap := range accessProviders {
		apBindingsToAdd, apBindingsToRemove, apAclAssignemnts, err2 := convertAccessProviderToIamRoleAssignments(ctx, ap, iamClient, configMap.Parameters)
		if err2 != nil {
			feedbackHandler.Error(err2.Error(), ap.Id)
			continue
		}

		roleBindingsToAdd.Add(apBindingsToAdd...)
		roleBindingsToRemove.Add(apBindingsToRemove...)

		for _, binding := range apBindingsToAdd {
			roleBindingApMap[binding] = append(roleBindingApMap[binding], ap.Id)
		}

		for _, binding := range apBindingsToRemove {
			roleBindingApMap[binding] = append(roleBindingApMap[binding], ap.Id)
		}

		aclAssignments.AddAssignments(apAclAssignemnts, ap.Id)
	}

	roleBindingsToRemove.RemoveAll(roleBindingsToAdd.Slice()...)

	for binding := range roleBindingsToRemove {
		err2 := global.DeleteRoleAssignment(ctx, configMap.Parameters, binding)
		if err2 != nil {
			feedbackHandler.Error(err2.Error(), roleBindingApMap[binding]...)
		}
	}

	for binding := range roleBindingsToAdd {
		err2 := global.CreateRoleAssignment(ctx, configMap.Parameters, binding)
		if err2 != nil {
			feedbackHandler.Error(err2.Error(), roleBindingApMap[binding]...)
		}
	}

	err = setACLs(ctx, aclAssignments, feedbackHandler, configMap.Parameters)
	if err != nil {
		return err
	}

	return nil
}

// return value 1: bindings to create, 2: bindings to delete, 3: acl bindings change set
func convertAccessProviderToIamRoleAssignments(ctx context.Context, accessProvider *importer.AccessProvider, iamClient *global.IamClient, params map[string]string) ([]global.IAMRoleAssignment, []global.IAMRoleAssignment, ACLAssignments, error) {
	dsSync := DataSourceSyncer{}

	bindings := make([][]global.IAMRoleAssignment, 2)
	aclAssignments := make(ACLAssignments)

	var aclAssignees, removedAclAssignees []ACLAssignee

	userPrincipalIds := make([]string, 0, len(accessProvider.Who.Users))
	groupPrincipalIds := make([]string, 0, len(accessProvider.Who.Groups))

	for _, user := range accessProvider.Who.Users {
		userPrincipalIds = append(userPrincipalIds, iamClient.GetPrincipalIdByName(armauthorization.PrincipalTypeUser, user))
	}

	for _, group := range accessProvider.Who.Groups {
		groupPrincipalIds = append(groupPrincipalIds, iamClient.GetPrincipalIdByName(armauthorization.PrincipalTypeGroup, group))
	}

	var deletedUserPrincipalIds, deletedGroupPrincipalIds []string
	if accessProvider.DeletedWho != nil {
		deletedUserPrincipalIds = make([]string, 0, len(accessProvider.DeletedWho.Users))
		deletedGroupPrincipalIds = make([]string, 0, len(accessProvider.DeletedWho.Groups))

		for _, user := range accessProvider.DeletedWho.Users {
			deletedUserPrincipalIds = append(deletedUserPrincipalIds, iamClient.GetPrincipalIdByName(armauthorization.PrincipalTypeUser, user))
		}

		for _, group := range accessProvider.DeletedWho.Groups {
			deletedGroupPrincipalIds = append(deletedGroupPrincipalIds, iamClient.GetPrincipalIdByName(armauthorization.PrincipalTypeGroup, group))
		}
	}

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
			case "folder":
				if aclAssignees == nil {
					aclAssignees, removedAclAssignees = generateACLAssignees(userPrincipalIds, groupPrincipalIds, deletedUserPrincipalIds, deletedGroupPrincipalIds)
				}

				assignments, err := convertToACLAssignment(fullNameParts, what.Permissions, i == 1 || accessProvider.Delete, aclAssignees, removedAclAssignees)
				if err != nil {
					return nil, nil, nil, err
				}

				aclAssignments.AddAssignments(assignments)
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

				for _, u := range userPrincipalIds {
					bindings[i] = append(bindings[i], global.IAMRoleAssignment{
						Scope:            scope,
						RoleName:         permission,
						RoleDefinitionID: *permissionId,
						PrincipalType:    armauthorization.PrincipalTypeUser,
						PrincipalId:      u,
					})
				}

				for _, g := range groupPrincipalIds {
					bindings[i] = append(bindings[i], global.IAMRoleAssignment{
						Scope:            scope,
						RoleName:         permission,
						RoleDefinitionID: *permissionId,
						PrincipalType:    armauthorization.PrincipalTypeGroup,
						PrincipalId:      g,
					})
				}
			}
		}
	}

	if accessProvider.Delete {
		return nil, append(bindings[1], bindings[0]...), aclAssignments, nil
	}

	return bindings[0], bindings[1], aclAssignments, nil
}

func generateACLAssignees(userPrincipalIds, groupPrincipalIds, deletedUserPrincipalIds, deletedGroupPrincipalIds []string) ([]ACLAssignee, []ACLAssignee) {
	assignees := make([]ACLAssignee, 0, len(userPrincipalIds)+len(groupPrincipalIds))

	for _, u := range userPrincipalIds {
		assignees = append(assignees, ACLAssignee(fmt.Sprintf("user:%s", u)))
	}

	for _, g := range groupPrincipalIds {
		assignees = append(assignees, ACLAssignee(fmt.Sprintf("group:%s", g)))
	}

	removedAssignees := make([]ACLAssignee, 0, len(deletedUserPrincipalIds)+len(deletedGroupPrincipalIds))
	for _, u := range deletedUserPrincipalIds {
		removedAssignees = append(removedAssignees, ACLAssignee(fmt.Sprintf("user:%s", u)))
	}

	for _, g := range deletedGroupPrincipalIds {
		removedAssignees = append(removedAssignees, ACLAssignee(fmt.Sprintf("group:%s", g)))
	}

	return assignees, removedAssignees
}

func convertToACLAssignment(fullNameParts []string, permissions []string, deleted bool, assignees []ACLAssignee, removedAssignees []ACLAssignee) (ACLAssignments, error) {
	item := ACLAssignedItem{
		StorageAccount: fullNameParts[2],
		Container:      fullNameParts[3],
		Path:           strings.Join(fullNameParts[4:], "/"),
	}

	permissionSet := ACLPermissionSet(0)

	for _, permission := range permissions {
		aclPermission, err := ACLPermissionString(permission)
		if err != nil {
			return nil, err
		}

		permissionSet = permissionSet.Add(aclPermission)
	}

	changes := ACLPermissionChanges{}
	if deleted {
		changes.Removed = permissionSet
	} else {
		changes.Added = permissionSet
	}

	removedChanges := ACLPermissionChanges{
		Removed: permissionSet,
	}

	result := make(ACLAssignments)

	for _, assignee := range assignees {
		assignmentId := ACLAssignment{
			Assignee: assignee,
			Item:     item,
		}
		result[assignmentId] = changes
	}

	for _, assignee := range removedAssignees {
		assignmentId := ACLAssignment{
			Assignee: assignee,
			Item:     item,
		}
		result[assignmentId] = removedChanges
	}

	return result, nil
}

func setACLs(ctx context.Context, acls ACLAssignmentsWithAP, feedbackHandler global.AccessProviderFeedbackHandler, params map[string]string) error {
	var assignedItems []ACLAssignedItem
	aclsPerItem := make(map[ACLAssignedItem]map[ACLAssignee]ACLPermissionChangesWithAP)

	for aclAssignment, changes := range acls {
		item := aclAssignment.Item

		if _, ok := aclsPerItem[item]; !ok {
			aclsPerItem[item] = make(map[ACLAssignee]ACLPermissionChangesWithAP)

			assignedItems = append(assignedItems, item)
		}

		aclsPerItem[item][aclAssignment.Assignee] = changes
	}

	sort.Slice(assignedItems, func(i, j int) bool {
		sectionsI := strings.Count(assignedItems[i].Path, "/")
		sectionsJ := strings.Count(assignedItems[j].Path, "/")

		if sectionsI == sectionsJ {
			return assignedItems[i].Path < assignedItems[j].Path
		}

		return sectionsI < sectionsJ
	})

	for _, item := range assignedItems {
		assigneesAndChanges := aclsPerItem[item]

		aclStringsToAdd := make([]string, 0, len(assigneesAndChanges)*2)
		aclStringsToRemove := make([]string, 0, len(assigneesAndChanges)*2)

		apIds := set.NewSet[string]()

		for assignee, changes := range assigneesAndChanges {
			aclPermissionSet, toRemove := changes.ChangeSet()

			aclStringForAssignee := string(assignee)
			if !toRemove {
				aclStringForAssignee += ":" + aclPermissionSet.String()
			}

			defaultAclStringForAssignee := fmt.Sprintf("default:%s", aclStringForAssignee)

			if toRemove {
				aclStringsToRemove = append(aclStringsToRemove, aclStringForAssignee, defaultAclStringForAssignee)
			} else {
				aclStringsToAdd = append(aclStringsToAdd, aclStringForAssignee, defaultAclStringForAssignee)
			}

			apIds.Add(changes.APIds...)
		}

		client, err := createDirectoryClient(ctx, item.StorageAccount, item.Container, item.Path, params)
		if err != nil {
			feedbackHandler.Error(err.Error(), apIds.Slice()...)
			continue
		}

		if len(aclStringsToRemove) > 0 {
			aclStringToRemove := strings.Join(aclStringsToRemove, ",")
			logger.Info(fmt.Sprintf("Remove ACLs for %s/%s/%s: %s", item.StorageAccount, item.Container, item.Path, aclStringToRemove))

			r, err := client.RemoveAccessControlRecursive(ctx, aclStringToRemove, nil)
			if err != nil {
				logger.Error(fmt.Sprintf("Something went wrong while removing ACLs for %s/%s/%s: %s", item.StorageAccount, item.Container, item.Path, err.Error()))

				feedbackHandler.Error(err.Error(), apIds.Slice()...)

				continue
			}

			err = parseRemoveAccessControlResult(&r)
			if err != nil {
				feedbackHandler.Error(err.Error(), apIds.Slice()...)

				continue
			}
		}

		if len(aclStringsToAdd) > 0 {
			aclStringToAdd := strings.Join(aclStringsToAdd, ",")

			logger.Info(fmt.Sprintf("Add ACLs for %s/%s/%s: %s", item.StorageAccount, item.Container, item.Path, aclStringToAdd))

			r, err := client.UpdateAccessControlRecursive(ctx, aclStringToAdd, &directory.UpdateAccessControlRecursiveOptions{
				BatchSize: ptr.Int32(1),
			})
			if err != nil {
				logger.Error(fmt.Sprintf("Something went wrong while adding ACLs for %s/%s/%s: %s", item.StorageAccount, item.Container, item.Path, err.Error()))

				feedbackHandler.Error(err.Error(), apIds.Slice()...)

				continue
			}

			err = parseUpdateAccessControlResult(&r)
			if err != nil {
				feedbackHandler.Error(err.Error(), apIds.Slice()...)

				continue
			}
		}
	}

	return nil
}

func parseUpdateAccessControlResult(r *directory.SetAccessControlRecursiveResponse) error {
	if r.FailureCount != nil && *r.FailureCount > 0 {
		logger.Error(fmt.Sprintf("Recursive Access Control failed for %d entities", *r.FailureCount))

		for _, e := range r.FailedEntries {
			logger.Debug(fmt.Sprintf("Failed to update access control for %s %s", *e.Type, *e.Name))
		}

		return fmt.Errorf("recursive access control update failed for %d entities", *r.FailureCount)
	} else {
		var updatedDirs int32
		var updatedFiles int32

		if r.DirectoriesSuccessful != nil {
			updatedDirs = *r.DirectoriesSuccessful
		}

		if r.FilesSuccessful != nil {
			updatedFiles = *r.FilesSuccessful
		}

		logger.Debug(fmt.Sprintf("Recursive Access Control updated %d directories and %d files", updatedDirs, updatedFiles))
	}

	return nil
}

func parseRemoveAccessControlResult(r *directory.RemoveAccessControlRecursiveResponse) error {
	if r.FailureCount != nil && *r.FailureCount > 0 {
		logger.Error(fmt.Sprintf("Recursive Access Control removal failed for %d entities", *r.FailureCount))

		for _, e := range r.FailedEntries {
			logger.Debug(fmt.Sprintf("Failed to remove access control for %s %s", *e.Type, *e.Name))
		}

		return fmt.Errorf("recursive access control removal failed for %d entities", *r.FailureCount)
	} else {
		var removedDirs int32
		var removedFiles int32

		if r.DirectoriesSuccessful != nil {
			removedDirs = *r.DirectoriesSuccessful
		}

		if r.FilesSuccessful != nil {
			removedFiles = *r.FilesSuccessful
		}

		logger.Debug(fmt.Sprintf("Recursive Access Control removed %d directories and %d files", removedDirs, removedFiles))
	}

	return nil
}
