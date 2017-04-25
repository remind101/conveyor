package awsdynamo

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

var CheckThrottleHandler = request.NamedHandler{Name: "remind.unreads.CheckThrottle", Fn: CheckThrottle}

// Retry Handler to log if we get throttled.
func CheckThrottle(req *request.Request) {
	defer func() {
		if r := recover(); r != nil {
			logThrottle("", "we got throttled but panic'd while parsing", "")
		}
	}()

	if !req.IsErrorThrottle() {
		return
	}

	switch t := req.Params.(type) {
	case *dynamodb.QueryInput:
		checkThrottleQuery(t)
	case *dynamodb.PutItemInput:
		checkThrottlePutItem(t)
	case *dynamodb.GetItemInput:
		checkThrottleGetItem(t)
	case *dynamodb.UpdateItemInput:
		checkThrottleUpdateItem(t)
	case *dynamodb.BatchGetItemInput:
		checkThrottleBatchGetItem(t)
	case *dynamodb.BatchWriteItemInput:
		checkThrottleBatchWriteItem(t)
	case *dynamodb.CreateTableInput, *dynamodb.DeleteTableInput:
		// Do nothing
	default:
		logThrottle("ERR", "ERR", "ERR")
	}
}

func checkThrottleQuery(qi *dynamodb.QueryInput) {
	tablename := aws.StringValue(qi.TableName)
	hashKey := aws.StringValue(qi.ExpressionAttributeValues[BuildQueryHashKeyName].S)
	logThrottle(tablename, "Query", hashKey)
}

func checkThrottleGetItem(gii *dynamodb.GetItemInput) {
	tablename := aws.StringValue(gii.TableName)
	table := AwsTablesByName[tablename]
	if table == nil {
		logThrottle("ERR", "GetItem", "ERR")
		return
	}
	key, err := table.DynamoKeyFromAwsDynamoItem(gii.Key)
	if err != nil {
		logThrottle(tablename, "GetItem", "ERR")
		return
	}
	logThrottle(tablename, "GetItem", key.HashKey)
}

func checkThrottlePutItem(pii *dynamodb.PutItemInput) {
	tablename := aws.StringValue(pii.TableName)
	table := AwsTablesByName[tablename]
	if table == nil {
		logThrottle("ERR", "PutItem", "ERR")
		return
	}
	key, err := table.DynamoKeyFromAwsDynamoItem(pii.Item)
	if err != nil {
		logThrottle(tablename, "PutItem", "ERR")
		return
	}
	logThrottle(tablename, "PutItem", key.HashKey)
}

func checkThrottleUpdateItem(uii *dynamodb.UpdateItemInput) {
	tablename := aws.StringValue(uii.TableName)
	table := AwsTablesByName[tablename]
	if table == nil {
		logThrottle("ERR", "UpdateItem", "ERR")
		return
	}
	key, err := table.DynamoKeyFromAwsDynamoItem(uii.Key)
	if err != nil {
		logThrottle(tablename, "UpdateItem", "ERR")
		return
	}
	logThrottle(tablename, "UpdateItem", key.HashKey)
}

func checkThrottleBatchGetItem(bgii *dynamodb.BatchGetItemInput) {
	logThrottle("todo", "BatchGetItem", "todo")
}

func checkThrottleBatchWriteItem(bwii *dynamodb.BatchWriteItemInput) {
	logThrottle("todo", "BatchWriteItem", "todo")
}

func logThrottle(tableName, operation, partitionKey string) {
	logger.Info(context.Background(), "at=aws_dynamo_throttling",
		"tablename", tableName,
		"operation", operation,
		"partition_key", partitionKey,
	)
}
