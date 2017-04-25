package dynamomodel

type ConnectionParams struct {
	RegionName     string
	LocalDynamoURL string // if not "", URL of local Dynamo to point to.
	Scope          string // configures the table name per env
}
