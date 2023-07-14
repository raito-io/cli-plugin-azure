package global

import "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"

type IAMRoleAssignment struct {
	PrincipalId      string                         `json:"principalId"`
	PrincipalType    armauthorization.PrincipalType `json:"principalType"`
	RoleName         string                         `json:"roleName"`
	RoleDefinitionID string                         `json:"roleDefinitionId"`
	Scope            string                         `json:"scope"`
}
