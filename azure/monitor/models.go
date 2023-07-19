package monitor

import (
	"reflect"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
)

type ResourceDiagnosticSetting struct {
	Resource        string `json:"resource"`
	WorkspaceID     string `json:"workspace_id"`
	ReadLogsEnabled bool   `json:"read_logs_enabled"`
}

type LogEntry struct {
	TimeGenerated      string
	OperationName      string
	ObjectKey          string
	RequesterObjectId  string
	AuthenticationType string
	CorrelationId      string
}

func (l LogEntry) FromRow(row azquery.Row, columns []*azquery.Column) LogEntry {
	structValue := reflect.ValueOf(&l).Elem()
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)
		fieldName := fieldType.Name

		for j, col := range columns {
			if strings.EqualFold(fieldName, *col.Name) {
				field.Set(reflect.ValueOf(row[j]))
				break
			}
		}
	}

	return l
}
