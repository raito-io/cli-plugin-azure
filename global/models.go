package global

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/raito-io/cli/base/access_provider/sync_to_target"
)

type IAMRoleAssignment struct {
	PrincipalId      string                         `json:"principalId"`
	PrincipalType    armauthorization.PrincipalType `json:"principalType"`
	RoleName         string                         `json:"roleName"`
	RoleDefinitionID string                         `json:"roleDefinitionId"`
	Scope            string                         `json:"scope"`
}

type AccessProviderFeedbackHandler interface {
	Error(err string, apIds ...string)
	Warning(warning string, apIds ...string)
}

type AccessProviderFeedbackMap map[string]sync_to_target.AccessProviderSyncFeedback
