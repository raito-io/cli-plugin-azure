package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/filesystem"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/service"
	"github.com/aws/smithy-go/ptr"
	ds "github.com/raito-io/cli/base/data_source"

	"github.com/raito-io/cli/base/util/config"
	"github.com/raito-io/cli/base/wrappers"

	"github.com/raito-io/cli-plugin-azure/global"
)

type DataSourceSyncer struct {
}

func (s *DataSourceSyncer) SyncDataSource(ctx context.Context, dataSourceHandler wrappers.DataSourceObjectHandler, configMap *config.ConfigMap) error {
	stAccnts, err := getStorageAccounts(ctx, configMap.GetStringWithDefault(global.AzSubscriptionId, ""), configMap.Parameters)

	if handleError(err) != nil {
		return err
	}

	err = dataSourceHandler.AddDataObjects(&ds.DataObject{
		ExternalId:       configMap.GetStringWithDefault(global.AzSubscriptionId, ""),
		Name:             fmt.Sprintf("subscription-%s", configMap.GetStringWithDefault(global.AzSubscriptionId, "")),
		FullName:         configMap.GetStringWithDefault(global.AzSubscriptionId, ""),
		Type:             "subscription",
		Description:      fmt.Sprintf("Azure subscription %s", configMap.GetStringWithDefault(global.AzSubscriptionId, "")),
		ParentExternalId: "",
	})

	if err != nil {
		return err
	}

	for k, v := range stAccnts {
		resourceGroup := fmt.Sprintf("%s/%s", configMap.GetStringWithDefault(global.AzSubscriptionId, ""), k)

		err := dataSourceHandler.AddDataObjects(&ds.DataObject{
			ExternalId:       resourceGroup,
			Name:             k,
			FullName:         resourceGroup,
			Type:             ResourceGroup,
			Description:      fmt.Sprintf("Azure Resource group %s", k),
			ParentExternalId: configMap.GetStringWithDefault(global.AzSubscriptionId, ""),
		})

		if err != nil {
			return err
		}

		for _, accnt := range v {
			err2 := s.syncStorageAccount(ctx, resourceGroup, accnt, dataSourceHandler, configMap)
			if err2 != nil {
				logger.Warn(fmt.Sprintf("Failed to sync storage account '%s/%s': %s", resourceGroup, accnt, err2.Error()))
				return err2
			}
		}
	}

	return nil
}

func (s *DataSourceSyncer) syncStorageAccount(ctx context.Context, parent string, accountName string, dataSourceHandler wrappers.DataSourceObjectHandler, configMap *config.ConfigMap) error {
	logger.Info(fmt.Sprintf("Processing storage account %s", accountName))
	storageAccount := fmt.Sprintf("%s/%s", parent, accountName)

	err2 := dataSourceHandler.AddDataObjects(&ds.DataObject{
		ExternalId:       storageAccount,
		Name:             accountName,
		FullName:         storageAccount,
		Type:             "storageaccount",
		Description:      fmt.Sprintf("Azure Storage Account %s", accountName),
		ParentExternalId: parent,
	})

	if err2 != nil {
		return err2
	}

	client, err := createDataLakeServiceClient(ctx, accountName, configMap.Parameters)
	if err != nil {
		return err
	}

	pager := client.NewListFileSystemsPager(&service.ListFileSystemsOptions{Include: service.ListFileSystemsInclude{Deleted: ptr.Bool(false)}})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, fs := range page.ListFileSystemsSegmentResponse.FileSystemItems {
			errFs := s.syncFileSystem(ctx, client, storageAccount, *fs.Name, dataSourceHandler)
			if errFs != nil {
				logger.Warn(fmt.Sprintf("Failed to sync file system '%s/%s': %s", storageAccount, *fs.Name, errFs.Error()))
			}
		}
	}

	return nil
}

func (s *DataSourceSyncer) syncFileSystem(ctx context.Context, serviceClient *service.Client, parent string, fileSystem string, dataSourceHandler wrappers.DataSourceObjectHandler) error {
	logger.Info(fmt.Sprintf("Processing container %s", fileSystem))
	storageContainer := fmt.Sprintf("%s/%s", parent, fileSystem)

	err := dataSourceHandler.AddDataObjects(&ds.DataObject{
		ExternalId:       storageContainer,
		Name:             fileSystem,
		FullName:         storageContainer,
		Type:             Container,
		Description:      fmt.Sprintf("Azure Storage Container %s", fileSystem),
		ParentExternalId: parent,
	})
	if err != nil {
		return err
	}

	client := serviceClient.NewFileSystemClient(fileSystem)

	pager := client.NewListPathsPager(true, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, path := range page.Paths {
			errPath := s.syncContainerObject(storageContainer, path, dataSourceHandler)
			if errPath != nil {
				logger.Warn(fmt.Sprintf("Failed to sync object '%s/%s': %s", storageContainer, *path.Name, errPath.Error()))
			}
		}
	}

	return nil
}

func (s *DataSourceSyncer) syncContainerObject(containerName string, path *filesystem.Path, dataSourceHandler wrappers.DataSourceObjectHandler) error {
	doType := File
	if path.IsDirectory != nil && *path.IsDirectory {
		doType = Folder
	}

	logger.Info(fmt.Sprintf("Processing container object %s %s", doType, *path.Name))
	fullName := fmt.Sprintf("%s/%s", containerName, *path.Name)
	fillNameSplit := strings.Split(fullName, "/")
	parent := strings.Join(fillNameSplit[0:len(fillNameSplit)-1], "/")

	err := dataSourceHandler.AddDataObjects(&ds.DataObject{
		ExternalId:       fullName,
		Name:             *path.Name,
		FullName:         fullName,
		Type:             doType,
		Description:      fmt.Sprintf("Azure Storage %s %s", doType, *path.Name),
		ParentExternalId: parent,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *DataSourceSyncer) GetDataObjectTypes(_ context.Context) ([]string, []*ds.DataObjectType) {
	logger.Debug("Returning meta data for Azure Storage data source")

	return []string{"subscription"}, []*ds.DataObjectType{
		{
			Name:        Subscription,
			Type:        Subscription,
			Permissions: s.GetIAMPermissions(),
			Children:    []string{ResourceGroup},
		},
		{
			Name:        ResourceGroup,
			Type:        ResourceGroup,
			Permissions: s.GetIAMPermissions(),
			Children:    []string{StorageAccount},
		},
		{
			Name:        StorageAccount,
			Type:        StorageAccount,
			Permissions: s.GetIAMPermissions(),
			Children:    []string{"container"},
		},
		{
			Name:        Container,
			Type:        Container,
			Permissions: s.GetIAMPermissions(),
			Children:    []string{Folder, File},
		},
		{
			Name: Folder,
			Type: Folder,
			Permissions: []*ds.DataObjectTypePermission{
				{
					Permission:             "Read",
					Description:            "Read access to the folder",
					GlobalPermissions:      []string{ds.Read},
					UsageGlobalPermissions: []string{ds.Read},
				},
				{
					Permission:             "Write",
					Description:            "Write access to the folder",
					GlobalPermissions:      []string{ds.Write},
					UsageGlobalPermissions: []string{ds.Write},
				},
				{
					Permission:        "Execute",
					Description:       "Execute access to the folder",
					GlobalPermissions: []string{ds.Read, ds.Write},
				},
			},
			Children: []string{Folder, File},
		},
		{
			Name:        File,
			Type:        File,
			Permissions: nil,
			Actions: []*ds.DataObjectTypeAction{
				{
					Action:        "GetBlob",
					GlobalActions: []string{ds.Read},
				},
				{
					Action:        "PutBlob",
					GlobalActions: []string{ds.Write},
				},
				{
					Action:        "DeleteBlob",
					GlobalActions: []string{ds.Write},
				},
				{
					Action:        "DeleteBlob",
					GlobalActions: []string{ds.Write},
				},
			},
			Children: []string{},
		},
	}
}

func (s *DataSourceSyncer) GetDataSourceIAMPermissions() []*ds.DataObjectTypePermission {
	return []*ds.DataObjectTypePermission{}
}

func (s *DataSourceSyncer) GetIAMPermissions() []*ds.DataObjectTypePermission {
	return []*ds.DataObjectTypePermission{
		{
			Permission:             "Owner",
			Description:            "Grants full access to manage all resources, including the ability to assign roles in Azure RBAC.",
			UsageGlobalPermissions: []string{ds.Read, ds.Write, ds.Admin},
		},
		{
			Permission:             "Contributor",
			Description:            "Grants full access to manage all resources, but does not allow you to assign roles in Azure RBAC, manage assignments in Azure Blueprints, or share image galleries.",
			UsageGlobalPermissions: []string{ds.Read, ds.Write},
		},
		{
			Permission:             "Reader",
			Description:            "View all resources, but does not allow you to make any changes.",
			UsageGlobalPermissions: []string{ds.Read},
		},
		{
			Permission:             "Storage Blob Data Owner",
			Description:            "Provides full access to Azure Storage blob containers and data, including assigning POSIX access control.",
			GlobalPermissions:      []string{ds.Admin},
			UsageGlobalPermissions: []string{ds.Read, ds.Write, ds.Admin},
		},
		{
			Permission:             "Storage Blob Data Contributor",
			Description:            "Read, write, and delete Azure Storage containers and blobs.",
			GlobalPermissions:      []string{ds.Write},
			UsageGlobalPermissions: []string{ds.Read, ds.Write},
		},
		{
			Permission:             "Storage Blob Data Reader",
			Description:            "Read and list Azure Storage containers and blobs.",
			GlobalPermissions:      []string{ds.Read},
			UsageGlobalPermissions: []string{ds.Read, ds.Write},
		},
	}
}

func (s *DataSourceSyncer) IsApplicablePermission(ctx context.Context, resourceType, permission string) bool {
	_, doTypes := s.GetDataObjectTypes(ctx)

	for _, t := range doTypes {
		if strings.EqualFold(t.Name, resourceType) {
			for _, p := range t.Permissions {
				if strings.EqualFold(p.Permission, permission) {
					return true
				}
			}
		}
	}

	return false
}

func handleError(e error) error {
	if e != nil && strings.Contains(e.Error(), "403") {
		logger.Warn(fmt.Sprintf("Encountered authorization error during sync, this could indicate a lack of permissions for the App registration: %s", e.Error()))
	}

	return e
}
