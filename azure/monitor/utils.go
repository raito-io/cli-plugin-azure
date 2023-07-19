package monitor

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/raito-io/cli-plugin-azure/global"
)

func createArmMonitorClientFactory(ctx context.Context, params map[string]string) (*armmonitor.ClientFactory, error) {
	// create a credential for authenticating with Azure Active Directory
	cred, err := global.CreateADClientSecretCredential(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("could not create a credential from a secret: %w", err)
	}

	return armmonitor.NewClientFactory(params[global.AzSubscriptionId], cred, nil)
}

func createDiagnosticsSettingsClient(ctx context.Context, params map[string]string) (*armmonitor.DiagnosticSettingsClient, error) {
	factory, err := createArmMonitorClientFactory(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("could not create the client factoru for AZ monitor: %w", err)
	}

	return factory.NewDiagnosticSettingsClient(), nil
}

func createAzQueryLogsClient(ctx context.Context, params map[string]string) (*azquery.LogsClient, error) {
	cred, err := global.CreateADClientSecretCredential(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("could not create a credential from a secret: %w", err)
	}

	return azquery.NewLogsClient(cred, nil)
}
