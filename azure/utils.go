package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/raito-io/cli-plugin-azure/global"
	"github.com/raito-io/cli/base"
	"github.com/raito-io/cli/base/util/config"
)

var logger hclog.Logger

func init() {
	logger = base.Logger()
}

func GetDataUsageStartDate(ctx context.Context, configMap *config.ConfigMap) (time.Time, *time.Time, *time.Time) {
	numberOfDays := configMap.GetIntWithDefault(global.DataUsageWindow, 90)
	if numberOfDays > 90 {
		logger.Info(fmt.Sprintf("Capping data usage window to 90 days (from %d days)", numberOfDays))
		numberOfDays = 90
	}

	if numberOfDays <= 0 {
		logger.Info(fmt.Sprintf("Invalid input for data usage window (%d), setting to default 90 days", numberOfDays))
		numberOfDays = 90
	}

	syncStart := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -numberOfDays)

	var earliestTime *time.Time

	if _, found := configMap.Parameters["firstUsed"]; found {
		dateRaw, errLocal := time.Parse(time.RFC3339, configMap.Parameters["firstUsed"])
		logger.Debug(fmt.Sprintf("firstUsed parameter: %s", dateRaw.Format(time.RFC3339)))

		// 12-hour fudge factor; earliest usage data doesn't usually coincide with the start of the window
		if errLocal == nil && dateRaw.Add(-time.Hour*12).After(syncStart) {
			earliestTime = &dateRaw
		}
	}

	var latestTime *time.Time

	if _, found := configMap.Parameters["lastUsed"]; found {
		latestUsageRaw, errLocal := time.Parse(time.RFC3339, configMap.Parameters["lastUsed"])
		logger.Debug(fmt.Sprintf("lastUsed parameter: %s", latestUsageRaw.Format(time.RFC3339)))

		if errLocal == nil && latestUsageRaw.After(syncStart) {
			latestTime = &latestUsageRaw
		}
	}

	if earliestTime == nil && latestTime != nil {
		return *latestTime, nil, nil
	}

	return syncStart, earliestTime, latestTime
}
