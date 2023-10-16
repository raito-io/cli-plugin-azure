package global

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/raito-io/cli-plugin-azure-ad/ad"
)

type IamClient struct {
	iamClient *ad.IdentityContainer

	// Cache
	principalNameToIdMap map[string]string
}

func NewIamClient(ctx context.Context, params map[string]string) (*IamClient, error) {
	c, err := ad.NewIdentityStoreSyncer().GetIdentityContainer(ctx, params)
	if err != nil {
		return nil, err
	}

	return &IamClient{
		iamClient:            c,
		principalNameToIdMap: make(map[string]string, 0),
	}, nil
}

func (c *IamClient) GetPrincipalIdByName(principalType armauthorization.PrincipalType, name string) string {
	if principalType == armauthorization.PrincipalTypeGroup {
		for _, group := range c.iamClient.Groups {
			if group.Name == name {
				return group.ExternalId
			}
		}
	} else if principalType == armauthorization.PrincipalTypeUser {
		for _, user := range c.iamClient.Users {
			if user.UserName == name {
				return user.ExternalId
			}
		}
	}

	return ""
}

func (c *IamClient) GetPrincipalNameById(principalType armauthorization.PrincipalType, id string) string {
	if principalType == armauthorization.PrincipalTypeGroup {
		for _, group := range c.iamClient.Groups {
			if group.ExternalId == id {
				return group.Name
			}
		}
	} else if principalType == armauthorization.PrincipalTypeUser {
		for _, user := range c.iamClient.Users {
			if user.ExternalId == id {
				return user.UserName
			}
		}
	}

	return ""
}
