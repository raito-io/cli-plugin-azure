package azure

type GroupEntity struct {
	ExternalId string
	Email      string
	Members    []string
}

type UserEntity struct {
	ExternalId string
	Name       string
	Email      string
}

type BQEntity struct {
	ID       string
	Type     string
	Name     string
	FullName string
	ParentId string
}

type BQInformationSchemaEntity struct {
	CachedQuery bool                `bigquery:"cache_hit"`
	User        string              `bigquery:"user_email"`
	Query       string              `bigquery:"query"`
	Tables      []BQReferencedTable `bigquery:"referenced_tables"`
	StartTime   int64               `bigquery:"start_time"`
	EndTime     int64               `bigquery:"end_time"`
}

type BQReferencedTable struct {
	Project string `bigquery:"project_id"`
	Dataset string `bigquery:"dataset_id"`
	Table   string `bigquery:"table_id"`
}
