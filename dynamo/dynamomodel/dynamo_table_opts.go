package dynamomodel

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

type TableOpts struct {
	Desc                *dynamodb.TableDescription
	AwsDynamo           dynamodbiface.DynamoDBAPI
	EnableReadsLogging  bool
	EnableWritesLogging bool
}
