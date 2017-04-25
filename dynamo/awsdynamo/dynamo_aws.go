package awsdynamo

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/remind101/conveyor/dynamo/dynamomodel"
)

func NewAwsDynamo(params dynamomodel.ConnectionParams) *dynamodb.DynamoDB {
	session := session.New()
	config := &aws.Config{
		Region: aws.String(params.RegionName),
	}
	svc := dynamodb.New(session, config)
	svc.Handlers.Retry.PushFrontNamed(CheckThrottleHandler)
	return svc
}

func NewTestAwsDynamo(params dynamomodel.ConnectionParams) *dynamodb.DynamoDB {
	if params.RegionName == "" {
		params.RegionName = "us-east-1"
	}
	session := session.New()
	config := &aws.Config{
		Credentials: credentials.NewStaticCredentials("abc", "def", ""),
		Region:      aws.String(params.RegionName),
		Endpoint:    aws.String(params.LocalDynamoURL),
	}
	svc := dynamodb.New(session, config)
	svc.Handlers.Retry.PushFrontNamed(CheckThrottleHandler)
	return svc
}

func MakeAwsTables(params dynamomodel.ConnectionParams) map[string]*dynamodb.TableDescription {
	return map[string]*dynamodb.TableDescription{
		"builds": &dynamodb.TableDescription{
			TableName: aws.String(ScopedTableName("builds", params)),
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				&dynamodb.AttributeDefinition{
					AttributeName: aws.String("uuid"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				&dynamodb.KeySchemaElement{
					AttributeName: aws.String("uuid"),
					KeyType:       aws.String("HASH"),
				},
			},
			ProvisionedThroughput: &dynamodb.ProvisionedThroughputDescription{
				ReadCapacityUnits:  aws.Int64(1),
				WriteCapacityUnits: aws.Int64(1),
			},
		},
		"artifacts": &dynamodb.TableDescription{
			TableName: aws.String(ScopedTableName("artifacts", params)),
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				&dynamodb.AttributeDefinition{
					AttributeName: aws.String("uuid"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				&dynamodb.KeySchemaElement{
					AttributeName: aws.String("uuid"),
					KeyType:       aws.String("HASH"),
				},
			},
			ProvisionedThroughput: &dynamodb.ProvisionedThroughputDescription{
				ReadCapacityUnits:  aws.Int64(1),
				WriteCapacityUnits: aws.Int64(1),
			},
		},
	}
}

func ScopedTableName(name string, params dynamomodel.ConnectionParams) string {
	return fmt.Sprintf("%s-%s", params.Scope, name)
}
