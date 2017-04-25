package dynamomodel

import "github.com/aws/aws-sdk-go/service/dynamodb"

type Expression struct {
	// http://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb/#UpdateItemInput
	ConditionExpression       string
	UpdateExpression          string
	ExpressionAttributeNames  map[string]*string
	ExpressionAttributeValues map[string]*dynamodb.AttributeValue
}
