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
