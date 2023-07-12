package storage

import (
	"context"
	"fmt"
	"strings"

	ds "github.com/raito-io/cli/base/data_source"

	"github.com/raito-io/cli-plugin-azure/global"
	"github.com/raito-io/cli/base/util/config"
	"github.com/raito-io/cli/base/wrappers"
)

type DataSourceSyncer struct {
}

func (s *DataSourceSyncer) SyncDataSource(ctx context.Context, dataSourceHandler wrappers.DataSourceObjectHandler, configMap *config.ConfigMap) error {
	stAccnts, err := getStorageAccounts(ctx, configMap.GetStringWithDefault(global.AzSubscriptionId, ""))

	if err != nil {
		return err
	}

	for k, v := range stAccnts {
		resourceGroup := fmt.Sprintf("%s/%s", configMap.GetStringWithDefault(global.AzSubscriptionId, ""), k)
		dataSourceHandler.AddDataObjects(&ds.DataObject{
			ExternalId:       resourceGroup,
			Name:             k,
			FullName:         resourceGroup,
			Type:             "resourcegroup",
			Description:      fmt.Sprintf("Azure Resource group %s", k),
			ParentExternalId: "",
		})

		for _, accnt := range v {
			storageAccount := fmt.Sprintf("%s/%s", resourceGroup, accnt)
			dataSourceHandler.AddDataObjects(&ds.DataObject{
				ExternalId:       storageAccount,
				Name:             accnt,
				FullName:         storageAccount,
				Type:             "storageaccount",
				Description:      fmt.Sprintf("Azure Storage Account %s", accnt),
				ParentExternalId: resourceGroup,
			})

			client, err := createAZBlobClient(accnt)

			if err != nil {
				return err
			}

			pager := client.NewListContainersPager(nil)
			for pager.More() {
				page, err := pager.NextPage(ctx)
				if err != nil {
					return err
				}

				for _, v := range page.ContainerItems {
					storageContainer := fmt.Sprintf("%s/%s", storageAccount, *v.Name)
					dataSourceHandler.AddDataObjects(&ds.DataObject{
						ExternalId:       storageContainer,
						Name:             *v.Name,
						FullName:         storageContainer,
						Type:             "container",
						Description:      fmt.Sprintf("Azure Storage Container %s", *v.Name),
						ParentExternalId: storageAccount,
					})

					logger.Error(fmt.Sprintf("%s\n\n", *v.Name))

					pager2 := client.NewListBlobsFlatPager(*v.Name, nil)

					for pager2.More() {
						page2, err2 := pager2.NextPage(ctx)
						if err2 != nil {
							return err2
						}

						for _, v2 := range page2.Segment.BlobItems {
							split := strings.Split(*v2.Name, "/")
							name := split[len(split)-1]

							fullName := fmt.Sprintf("%s/%s", *v.Name, *v2.Name)
							parentExternalId := storageContainer

							if len(split) > 1 {
								fsplit := strings.Split(fullName, "/")
								parentExternalId = strings.Join(fsplit[0:len(fsplit)-1], "/")
							}

							doType := "blob"

							if strings.EqualFold(*v2.Properties.ContentType, "application/octet-stream") {
								doType = "folder"
							}

							dataSourceHandler.AddDataObjects(&ds.DataObject{
								ExternalId:       fullName,
								Name:             name,
								FullName:         fullName,
								Type:             doType,
								Description:      fmt.Sprintf("Azure Storage %s %s", doType, fullName),
								ParentExternalId: parentExternalId,
							})
						}
					}
				}
			}
		}
	}

	return nil
}

func (s *DataSourceSyncer) GetDataObjectTypes(ctx context.Context) ([]string, []*ds.DataObjectType) {
	logger.Debug("Returning meta data for Azure Storage data source")

	return []string{"resourcegroup"}, []*ds.DataObjectType{
		{
			Name:        "resourcegroup",
			Type:        "resourcegroup",
			Permissions: s.GetIAMPermissions(),
			Children:    []string{"storageaccount"},
		},
		{
			Name:        "storageaccount",
			Type:        "storageaccount",
			Permissions: s.GetIAMPermissions(),
			Children:    []string{"container"},
		},
		{
			Name:        "container",
			Type:        "container",
			Permissions: s.GetIAMPermissions(),
			Children:    []string{"folder", "object"},
		},
		{
			Name:        "folder",
			Type:        "folder",
			Permissions: []*ds.DataObjectTypePermission{},
			Children:    []string{"folder", "blob"},
		},
		{
			Name:        "blob",
			Type:        "blob",
			Permissions: []*ds.DataObjectTypePermission{},
			Children:    []string{},
		},
	}
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
			UsageGlobalPermissions: []string{ds.Read, ds.Write, ds.Admin},
		},
		{
			Permission:             "Storage Blob Data Contributor",
			Description:            "Read, write, and delete Azure Storage containers and blobs.",
			UsageGlobalPermissions: []string{ds.Read, ds.Write},
		},
		{
			Permission:             "Storage Blob Data Reader",
			Description:            "Read and list Azure Storage containers and blobs.",
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