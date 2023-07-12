package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/hashicorp/go-hclog"
	"github.com/raito-io/cli/base"
)

var logger hclog.Logger

func init() {
	logger = base.Logger()
}

func createAZBlobClient(storageAccount string) (*azblob.Client, error) {
	// create a credential for authenticating with Azure Active Directory
	cred, err := azidentity.NewDefaultAzureCredential(nil)

	if err != nil {
		return nil, err
	}

	// create an azblob.Client for the specified storage account that uses the above credential
	return azblob.NewClient(fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccount), cred, nil)
}

func getStorageAccounts(ctx context.Context, subscription string) (map[string][]string, error) {
	// create a credential for authenticating with Azure Active Directory
	cred, err := azidentity.NewDefaultAzureCredential(nil)

	if err != nil {
		return nil, err
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
