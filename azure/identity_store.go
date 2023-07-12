package azure

import (
	"github.com/raito-io/cli-plugin-azure-ad/ad"
)

func NewIdentityStoreSyncer() *ad.IdentityStoreSyncer {
	return ad.NewIdentityStoreSyncer()
}
