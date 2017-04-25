package dynamomodel

import "github.com/aws/aws-sdk-go/service/dynamodb"

type QueryOpts struct {
	HashKey                   string
	Limit                     int64
	Descending                bool
	ExpressionAttributeNames  map[string]*string
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue
	ProjectionExpression      string
	KeyConditionExpression    string
	FilterExpression          string
	IndexName                 string
}
