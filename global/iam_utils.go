package global

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/google/uuid"
	"github.com/raito-io/cli-plugin-azure-ad/ad"
	e "github.com/raito-io/cli/base/util/error"
)

var identityContainer *ad.IdentityContainer

func CreateADClientSecretCredential(ctx context.Context, params map[string]string) (*azidentity.ClientSecretCredential, error) {
	secret := params[ad.AdSecret]

	if secret == "" {
		return nil, e.CreateMissingInputParameterError(ad.AdSecret)
	}

	clientId := params[ad.AdClientId]

	if secret == "" {
		return nil, e.CreateMissingInputParameterError(ad.AdClientId)
	}

	tenantId := params[ad.AdTenantId]

	if secret == "" {
		return nil, e.CreateMissingInputParameterError(ad.AdTenantId)
	}

	// Initializing the client credential
	return azidentity.NewClientSecretCredential(tenantId, clientId, secret, nil)
}

func createArmauthorizationClientFactory(ctx context.Context, params map[string]string) (*armauthorization.ClientFactory, error) {
	cred, err := CreateADClientSecretCredential(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("could not create a credential from a secret: %w", err)
	}

	return armauthorization.NewClientFactory(params[AzSubscriptionId], cred, nil)
}

func createRoleAssignmentClient(ctx context.Context, params map[string]string) (*armauthorization.RoleAssignmentsClient, error) {
	clientFactory, err := createArmauthorizationClientFactory(ctx, params)

	if err != nil {
		return nil, err
	}

	return clientFactory.NewRoleAssignmentsClient(), nil
}

func createRoleDefinitionsClient(ctx context.Context, params map[string]string) (*armauthorization.RoleDefinitionsClient, error) {
	clientFactory, err := createArmauthorizationClientFactory(ctx, params)

	if err != nil {
		return nil, err
	}

	return clientFactory.NewRoleDefinitionsClient(), nil
}

var roleDefIdToRoleNameMap = make(map[string]string, 0)
var roleDefNameToRoleIdMap = make(map[string]string, 0)

func GetRoleAssignments(ctx context.Context, params map[string]string) ([]IAMRoleAssignment, error) {
	client, err := createRoleAssignmentClient(ctx, params)

	if err != nil {
		return nil, err
	}

	defClient, err := createRoleDefinitionsClient(ctx, params)

	if err != nil {
		return nil, err
	}

	if identityContainer == nil {
		c, err2 := ad.NewIdentityStoreSyncer().GetIdentityContainer(ctx, params)

		if err2 != nil {
			return nil, err2
		}

		identityContainer = c
	}

	pager := client.NewListForScopePager("/subscriptions/"+params[AzSubscriptionId], nil)

	assignments := make([]IAMRoleAssignment, 0)

	for pager.More() {
		page, err2 := pager.NextPage(ctx)
		if err2 != nil {
			return nil, err2
		}

		for _, v := range page.Value {
			if _, f := roleDefIdToRoleNameMap[*v.Properties.RoleDefinitionID]; !f {
				defresp, err3 := defClient.GetByID(ctx, *v.Properties.RoleDefinitionID, nil)

				if err3 != nil {
					return nil, err3
				}

				roleDefIdToRoleNameMap[*v.Properties.RoleDefinitionID] = *defresp.Properties.RoleName
			}

			assignments = append(assignments, IAMRoleAssignment{
				PrincipalId:      getPrincipalNameById(identityContainer, *v.Properties.PrincipalType, *v.Properties.PrincipalID),
				PrincipalType:    *v.Properties.PrincipalType,
				RoleName:         roleDefIdToRoleNameMap[*v.Properties.RoleDefinitionID],
				RoleDefinitionID: *v.Properties.RoleDefinitionID,
				Scope:            *v.Properties.Scope,
			})
		}
	}

	return assignments, err
}

func GetRoleIdByName(ctx context.Context, params map[string]string, roleName string) (*string, error) {
	defClient, err := createRoleDefinitionsClient(ctx, params)

	if err != nil {
		return nil, err
	}

	if _, f := roleDefIdToRoleNameMap[roleName]; !f {
		pager := defClient.NewListPager("/subscriptions/"+params[AzSubscriptionId], nil)

		for pager.More() {
			page, err2 := pager.NextPage(ctx)
			if err2 != nil {
				return nil, err2
			}

			for _, v := range page.Value {
				roleDefNameToRoleIdMap[*v.Properties.RoleName] = *v.ID
			}
		}
	}

	id, f := roleDefNameToRoleIdMap[roleName]

	if !f {
		return nil, nil
	}

	return &id, nil
}

func getPrincipalNameById(ic *ad.IdentityContainer, principalType armauthorization.PrincipalType, id string) string {
	if principalType == armauthorization.PrincipalTypeGroup {
		for _, group := range ic.Groups {
			if group.ExternalId == id {
				return group.Name
			}
		}
	} else if principalType == armauthorization.PrincipalTypeUser {
		for _, user := range ic.Users {
			if user.ExternalId == id {
				return user.UserName
			}
		}
	}

	return ""
}

func getPrincipalIdByName(ic *ad.IdentityContainer, principalType armauthorization.PrincipalType, name string) string {
	if principalType == armauthorization.PrincipalTypeGroup {
		for _, group := range ic.Groups {
			if group.Name == name {
				return group.ExternalId
			}
		}
	} else if principalType == armauthorization.PrincipalTypeUser {
		for _, user := range ic.Users {
			if user.UserName == name {
				return user.ExternalId
			}
		}
	}

	return ""
}

func GetPrincipalIdByName(ctx context.Context, params map[string]string, principalType armauthorization.PrincipalType, name string) string {
	if identityContainer == nil {
		c, err := ad.NewIdentityStoreSyncer().GetIdentityContainer(ctx, params)

		if err != nil {
			return ""
		}

		identityContainer = c
	}

	return getPrincipalIdByName(identityContainer, principalType, name)
}

func CreateRoleAssignment(ctx context.Context, params map[string]string, binding IAMRoleAssignment) error {
	client, err := createRoleAssignmentClient(ctx, params)

	if err != nil {
		return err
	}

	_, err = client.Create(ctx, binding.Scope, uuid.New().String(), armauthorization.RoleAssignmentCreateParameters{
		Properties: &armauthorization.RoleAssignmentProperties{
			PrincipalID:      &binding.PrincipalId,
			PrincipalType:    &binding.PrincipalType,
			RoleDefinitionID: &binding.RoleDefinitionID,
		},
	}, nil)

	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}

	return err
}

func DeleteRoleAssignment(ctx context.Context, params map[string]string, binding IAMRoleAssignment) error {
	client, err := createRoleAssignmentClient(ctx, params)

	if err != nil {
		return err
	}

	pager := client.NewListForScopePager(binding.Scope, nil)

	for pager.More() {
		page, err2 := pager.NextPage(ctx)
		if err2 != nil {
			return err2
		}

		for _, v := range page.Value {
			if *v.Properties.PrincipalID == binding.PrincipalId && *v.Properties.RoleDefinitionID == binding.RoleDefinitionID {
				logger.Error("found the binding")

				_, err3 := client.Delete(ctx, binding.Scope, *v.Name, nil)

				return err3
			}
		}
	}

	return err
}
