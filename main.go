package main

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/raito-io/cli-plugin-azure-ad/ad"
	"github.com/raito-io/cli-plugin-azure/azure"
	"github.com/raito-io/cli-plugin-azure/global"
	"github.com/raito-io/cli/base"
	"github.com/raito-io/cli/base/info"
	"github.com/raito-io/cli/base/util/plugin"
	"github.com/raito-io/cli/base/wrappers"
)

var version = "0.0.0"

var logger hclog.Logger

func main() {
	logger = base.Logger()
	logger.SetLevel(hclog.Debug)

	err := base.RegisterPlugins(
		wrappers.IdentityStoreSync(azure.NewIdentityStoreSyncer()),
		wrappers.DataSourceSync(azure.NewDataSourceSyncer()),
		wrappers.DataAccessSync(azure.NewDataAccessSyncer()),
		wrappers.DataUsageSync(azure.NewDataUsageSyncer()), &info.InfoImpl{
			Info: &plugin.PluginInfo{
				Name:    "Azure",
				Version: plugin.ParseVersion(version),
				Parameters: []*plugin.ParameterInfo{
					{Name: ad.AdTenantId, Description: "The tenant ID for Azure Active Directory", Mandatory: true},
					{Name: ad.AdClientId, Description: "The client ID for Azure Active Directory", Mandatory: true},
					{Name: ad.AdSecret, Description: "The secret to connect to Azure Active Directory", Mandatory: true},
					{Name: global.AzSubscriptionId, Description: "The Azure Subscription ID", Mandatory: true},
				},
			},
		})

	if err != nil {
		logger.Error(fmt.Sprintf("error while registering plugins: %s", err.Error()))
	}
}
