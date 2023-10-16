package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/directory"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/service"
	"github.com/hashicorp/go-hclog"
	"github.com/raito-io/cli/base"

	"github.com/raito-io/cli-plugin-azure/global"
)

var logger hclog.Logger

func init() {
	logger = base.Logger()
}

func getStorageAccounts(ctx context.Context, subscription string, params map[string]string) (map[string][]string, error) {
	cred, err := global.CreateADClientSecretCredential(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("could not create a credential from a secret: %w", err)
	}

	clientFactory, err := armstorage.NewClientFactory(subscription, cred, nil)

	if err != nil {
		return nil, err
	}

	subscriptions := make(map[string][]string, 0)
	pager := clientFactory.NewAccountsClient().NewListPager(nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Value {
			resourceGroup := strings.Split(*v.ID, "/")[4]

			if _, f := subscriptions[resourceGroup]; !f {
				subscriptions[resourceGroup] = make([]string, 0)
			}

			subscriptions[resourceGroup] = append(subscriptions[resourceGroup], *v.Name)
		}
	}

	return subscriptions, nil
}

func createDataLakeServiceClient(ctx context.Context, accountName string, params map[string]string) (*service.Client, error) {
	cred, err := global.CreateADClientSecretCredential(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("could not create a credential from a secret: %w", err)
	}

	serviceURL := fmt.Sprintf("https://%s.dfs.core.windows.net/", accountName)

	return service.NewClient(serviceURL, cred, nil)
}

func createDirectoryClient(ctx context.Context, accountName string, fileSystem string, path string, params map[string]string) (*directory.Client, error) {
	cred, err := global.CreateADClientSecretCredential(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("could not create a credential from a secret: %w", err)
	}

	serviceURL := fmt.Sprintf("https://%s.dfs.core.windows.net/%s/%s", accountName, fileSystem, path)

	return directory.NewClient(serviceURL, cred, nil)
}
