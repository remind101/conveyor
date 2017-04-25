package awsdynamo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/metrics"
	"github.com/remind101/conveyor/dynamo/dynamomodel"
	"golang.org/x/net/context"
)

type AwsTable struct {
	desc                *dynamodb.TableDescription
	client              dynamodbiface.DynamoDBAPI
	PrimaryKey          *dynamomodel.PrimaryKey
	tableName           string
	capacityLevel       string // Consumed Capacity Logging Level: [INDEXES, TOTAL, NONE]
	enableReadsLogging  bool
	enableWritesLogging bool
}

var (
	ErrNotFound     = fmt.Errorf("item not found")
	ErrNotProcessed = fmt.Errorf("item not processed")
	AwsTablesByName = map[string]*AwsTable{}
)

const (
	BuildQueryHashKeyName = ":hash_key"
)

func NewAwsTable(opts dynamomodel.TableOpts) (*AwsTable, error) {
	tableName := aws.StringValue(opts.Desc.TableName)
	PrimaryKey, err := buildPrimaryKey(opts.Desc)
	if err != nil {
		return nil, fmt.Errorf("Error building primary key for %s: %v", tableName, err)
	}

	// Consumed Capacity Logging Level: [NONE, TOTAL, INDEXES]
	ccLevel := os.Getenv("DYNAMO_CAPACITY_LOG_LEVEL")
	if !isValidConsumedCapacityLevel(ccLevel) {
		ccLevel = dynamodb.ReturnConsumedCapacityNone
	}

	table := &AwsTable{
		desc:                opts.Desc,
		client:              opts.AwsDynamo,
		PrimaryKey:          &PrimaryKey,
		tableName:           tableName,
		capacityLevel:       ccLevel,
		enableReadsLogging:  opts.EnableReadsLogging,
		enableWritesLogging: opts.EnableWritesLogging,
	}

	AwsTablesByName[aws.StringValue(table.desc.TableName)] = table
	return table, nil
}

func (dt *AwsTable) Query(ctx context.Context, opts dynamomodel.QueryOpts) ([]map[string]*dynamodb.AttributeValue, error) {
	qi := dt.buildQuery(opts)
	qi.ReturnConsumedCapacity = aws.String(dt.capacityLevel)
	qo, err := dt.client.Query(qi)
	if qo != nil {
		dt.recordConsumedCapacity(ctx, "Query", opts.HashKey, qo.ConsumedCapacity)
	}
	return qo.Items, err
}

func (dt *AwsTable) PutItem(ctx context.Context, hashKey, rangeKey string, item dynamomodel.Item) error {
	item = dt.addPrimaryKey(hashKey, rangeKey, item)

	pii := &dynamodb.PutItemInput{
		Item:                   item,
		TableName:              dt.desc.TableName,
		ReturnConsumedCapacity: aws.String(dt.capacityLevel),
	}

	pio, err := dt.client.PutItem(pii)
	if pio != nil {
		dt.recordConsumedCapacity(ctx, "PutItem", hashKey, pio.ConsumedCapacity)
	}
	return err
}

func (dt *AwsTable) ConditionExpressionDeleteItem(ctx context.Context, hashKey, rangeKey string, expr dynamomodel.Expression) error {
	key := dt.addPrimaryKey(hashKey, rangeKey, dynamomodel.Item{})

	dii := &dynamodb.DeleteItemInput{
		TableName:                 dt.desc.TableName,
		Key:                       key,
		ConditionExpression:       aws.String(expr.ConditionExpression),
		ExpressionAttributeNames:  expr.ExpressionAttributeNames,
		ExpressionAttributeValues: expr.ExpressionAttributeValues,
		ReturnConsumedCapacity:    aws.String(dt.capacityLevel),
	}
	dio, err := dt.client.DeleteItem(dii)
	if dio != nil {
		dt.recordConsumedCapacity(ctx, "ConditionExpressionDeleteItem", hashKey, dio.ConsumedCapacity)
	}
	return err
}

func (dt *AwsTable) ConditionExpressionPutItem(ctx context.Context, hashKey, rangeKey string, item dynamomodel.Item, expr dynamomodel.Expression) error {
	item = dt.addPrimaryKey(hashKey, rangeKey, item)

	pii := &dynamodb.PutItemInput{
		TableName:                 dt.desc.TableName,
		Item:                      item,
		ConditionExpression:       aws.String(expr.ConditionExpression),
		ExpressionAttributeNames:  expr.ExpressionAttributeNames,
		ExpressionAttributeValues: expr.ExpressionAttributeValues,
		ReturnConsumedCapacity:    aws.String(dt.capacityLevel),
	}
	pio, err := dt.client.PutItem(pii)
	if pio != nil {
		dt.recordConsumedCapacity(ctx, "ConditionExpressionPutItem", hashKey, pio.ConsumedCapacity)
	}
	return err
}

func (dt *AwsTable) ConditionExpressionUpdateAttributes(ctx context.Context, hashKey, rangeKey string, expr dynamomodel.Expression) error {
	key := dt.addPrimaryKey(hashKey, rangeKey, dynamomodel.Item{})

	uii := &dynamodb.UpdateItemInput{
		TableName:                 dt.desc.TableName,
		Key:                       key,
		ConditionExpression:       aws.String(expr.ConditionExpression),
		ExpressionAttributeNames:  expr.ExpressionAttributeNames,
		ExpressionAttributeValues: expr.ExpressionAttributeValues,
		UpdateExpression:          aws.String(expr.UpdateExpression),
		ReturnConsumedCapacity:    aws.String(dt.capacityLevel),
	}
	uio, err := dt.client.UpdateItem(uii)
	if uio != nil {
		dt.recordConsumedCapacity(ctx, "ConditionExpressionUpdateAttributes", hashKey, uio.ConsumedCapacity)
	}

	return err
}

func (dt *AwsTable) GetItem(ctx context.Context, hashKey, rangeKey string) (dynamomodel.Item, error) {
	item := dt.addPrimaryKey(hashKey, rangeKey, dynamomodel.Item{})

	gii := &dynamodb.GetItemInput{
		TableName:              dt.desc.TableName,
		Key:                    item,
		ConsistentRead:         aws.Bool(false),
		ReturnConsumedCapacity: aws.String(dt.capacityLevel),
	}

	gio, err := dt.client.GetItem(gii)
	if gio != nil {
		dt.recordConsumedCapacity(ctx, "GetItem", hashKey, gio.ConsumedCapacity)
	}

	if isEmptyGetItemOutput(gio) {
		return nil, ErrNotFound
	}

	return gio.Item, err
}

func (dt *AwsTable) Create(ctx context.Context) error {
	pt := &dynamodb.ProvisionedThroughput{
		ReadCapacityUnits:  dt.desc.ProvisionedThroughput.ReadCapacityUnits,
		WriteCapacityUnits: dt.desc.ProvisionedThroughput.WriteCapacityUnits,
	}

	var localSecondaryIndexes []*dynamodb.LocalSecondaryIndex
	for _, desc := range dt.desc.LocalSecondaryIndexes {
		localSecondaryIndexes = append(localSecondaryIndexes, &dynamodb.LocalSecondaryIndex{
			IndexName:  desc.IndexName,
			KeySchema:  desc.KeySchema,
			Projection: desc.Projection,
		})
	}

	var globalSecondaryIndexes []*dynamodb.GlobalSecondaryIndex
	for _, desc := range dt.desc.GlobalSecondaryIndexes {
		gpt := &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  desc.ProvisionedThroughput.ReadCapacityUnits,
			WriteCapacityUnits: desc.ProvisionedThroughput.WriteCapacityUnits,
		}
		globalSecondaryIndexes = append(globalSecondaryIndexes, &dynamodb.GlobalSecondaryIndex{
			IndexName:             desc.IndexName,
			KeySchema:             desc.KeySchema,
			Projection:            desc.Projection,
			ProvisionedThroughput: gpt,
		})
	}

	cii := &dynamodb.CreateTableInput{
		AttributeDefinitions:   dt.desc.AttributeDefinitions,
		ProvisionedThroughput:  pt,
		KeySchema:              dt.desc.KeySchema,
		TableName:              dt.desc.TableName,
		LocalSecondaryIndexes:  localSecondaryIndexes,
		GlobalSecondaryIndexes: globalSecondaryIndexes,
		StreamSpecification:    dt.desc.StreamSpecification,
	}

	_, err := dt.client.CreateTable(cii)
	return err
}

func (dt *AwsTable) Delete(ctx context.Context) error {
	dii := &dynamodb.DeleteTableInput{
		TableName: dt.desc.TableName,
	}

	_, err := dt.client.DeleteTable(dii)
	return err
}

func (dt *AwsTable) PutDocument(ctx context.Context, key *dynamomodel.DynamoKey, data interface{}) error {
	return nil
}

func (dt *AwsTable) BatchGetDocument(ctx context.Context, keys []dynamomodel.DynamoKey, consistentRead bool) (*dynamodb.BatchGetItemOutput, *request.Request, error) {
	keysSlice := make([]map[string]*dynamodb.AttributeValue, len(keys))
	for i, key := range keys {
		keysSlice[i] = dt.addPrimaryKey(key.HashKey, key.RangeKey, dynamomodel.Item{})
	}

	requestItems := map[string]*dynamodb.KeysAndAttributes{
		*dt.desc.TableName: &dynamodb.KeysAndAttributes{
			ConsistentRead: aws.Bool(consistentRead),
			Keys:           keysSlice,
		},
	}

	bgii := &dynamodb.BatchGetItemInput{
		RequestItems:           requestItems,
		ReturnConsumedCapacity: aws.String(dt.capacityLevel),
	}

	req, bgio := dt.client.BatchGetItemRequest(bgii)
	err := req.Send()

	if bgio != nil {
		for _, c := range bgio.ConsumedCapacity {
			dt.recordConsumedCapacity(ctx, "BatchGetItem", "multi", c)
		}
	}

	return bgio, req, err
}

func (dt *AwsTable) BatchPutDocument(ctx context.Context, keys []dynamomodel.DynamoKey, items []dynamomodel.Item) (*dynamodb.BatchWriteItemOutput, *request.Request, error) {
	if len(keys) != len(items) {
		return nil, nil, fmt.Errorf("keys and items must have same length")
	}

	writeRequests := make([]*dynamodb.WriteRequest, len(keys))
	for index, key := range keys {
		item := dt.addPrimaryKey(key.HashKey, key.RangeKey, items[index])

		writeRequests[index] = &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: item,
			},
		}
	}

	requestItems := map[string][]*dynamodb.WriteRequest{
		*dt.desc.TableName: writeRequests,
	}

	bwii := &dynamodb.BatchWriteItemInput{
		RequestItems:           requestItems,
		ReturnConsumedCapacity: aws.String(dt.capacityLevel),
	}

	req, bwio := dt.client.BatchWriteItemRequest(bwii)
	err := req.Send()
	if bwio != nil {
		for _, c := range bwio.ConsumedCapacity {
			dt.recordConsumedCapacity(ctx, "BatchPutDocument", "multi", c)
		}
	}

	return bwio, req, err
}

func (dt *AwsTable) BatchDeleteDocument(ctx context.Context, keys []dynamomodel.DynamoKey) (*dynamodb.BatchWriteItemOutput, *request.Request, error) {
	writeRequests := make([]*dynamodb.WriteRequest, len(keys))
	for i, key := range keys {
		item := dt.addPrimaryKey(key.HashKey, key.RangeKey, dynamomodel.Item{})

		writeRequests[i] = &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: item,
			},
		}
	}

	requestItems := map[string][]*dynamodb.WriteRequest{
		*dt.desc.TableName: writeRequests,
	}

	bwii := &dynamodb.BatchWriteItemInput{
		RequestItems:           requestItems,
		ReturnConsumedCapacity: aws.String(dt.capacityLevel),
	}

	req, bwio := dt.client.BatchWriteItemRequest(bwii)
	err := req.Send()
	if bwio != nil {
		for _, c := range bwio.ConsumedCapacity {
			dt.recordConsumedCapacity(ctx, "BatchDeleteDocument", "multi", c)
		}
	}
	return bwio, req, err
}

// Builds a PrimaryKey from a TableDescription
func buildPrimaryKey(t *dynamodb.TableDescription) (PrimaryKey dynamomodel.PrimaryKey, err error) {
	for _, k := range t.KeySchema {
		ad := findAttributeDefinitionByName(t.AttributeDefinitions, aws.StringValue(k.AttributeName))
		if ad == nil {
			return PrimaryKey, fmt.Errorf("An inconsistency found in TableDescription")
		}

		switch aws.StringValue(k.KeyType) {
		case dynamodb.KeyTypeHash:
			PrimaryKey.PartitionKey = &dynamomodel.Key{Type: aws.StringValue(ad.AttributeType), Name: aws.StringValue(k.AttributeName)}
		case dynamodb.KeyTypeRange:
			PrimaryKey.SortKey = &dynamomodel.Key{Type: aws.StringValue(ad.AttributeType), Name: aws.StringValue(k.AttributeName)}
		default:
			return PrimaryKey, fmt.Errorf("key type not supported")
		}
	}
	return
}

// Finds Attribute Definition matching the passed in name
func findAttributeDefinitionByName(ads []*dynamodb.AttributeDefinition, name string) *dynamodb.AttributeDefinition {
	for _, ad := range ads {
		if aws.StringValue(ad.AttributeName) == name {
			return ad
		}
	}
	return nil
}

// Adds HashKey and RangeKey to a Item
func (dt *AwsTable) addPrimaryKey(hashKey, rangeKey string, item dynamomodel.Item) dynamomodel.Item {
	item[dt.PrimaryKey.PartitionKey.Name] = dt.PrimaryKey.PartitionKey.NewAttributeValue(hashKey)

	if dt.PrimaryKey.HasSortKey() {
		item[dt.PrimaryKey.SortKey.Name] = dt.PrimaryKey.SortKey.NewAttributeValue(rangeKey)
	}

	return item
}

func isEmptyGetItemOutput(gio *dynamodb.GetItemOutput) bool {
	return gio.Item == nil
}

// TODO Move this into InstrumentedDynamoTable when Goamz is removed
func (dt *AwsTable) recordConsumedCapacity(ctx context.Context, operation string, hashKey string, c *dynamodb.ConsumedCapacity) {
	operationType := "write"
	if operation == "GetItem" || operation == "BatchGetItem" || operation == "Query" {
		operationType = "read"
	}

	var totalConsumedCapacity, tableConsumedCapacity float64
	if c != nil {

		tags := map[string]string{"operation_type": operationType, "table_operation": operation, "table_name": dt.tableName}

		if c.CapacityUnits != nil {
			totalConsumedCapacity = aws.Float64Value(c.CapacityUnits)
			metrics.Gauge("unreads.consumed_capacity.capacity_units",
				totalConsumedCapacity, tags, 1.0)
		}

		if c.GlobalSecondaryIndexes != nil {
			for _, indexDesc := range dt.desc.GlobalSecondaryIndexes {
				indexName := aws.StringValue(indexDesc.IndexName)
				indexCap := c.GlobalSecondaryIndexes[indexName]
				if indexCap != nil {
					metrics.Gauge("unreads.consumed_capacity.global_secondary_index_units",
						aws.Float64Value(indexCap.CapacityUnits), tags, 1.0)
				}
			}
		}

		if c.LocalSecondaryIndexes != nil {
			for _, indexDesc := range dt.desc.LocalSecondaryIndexes {
				indexName := aws.StringValue(indexDesc.IndexName)
				indexCap := c.LocalSecondaryIndexes[indexName]
				if indexCap != nil {
					metrics.Gauge("unreads.consumed_capacity.local_secondary_index_units",
						aws.Float64Value(indexCap.CapacityUnits), tags, 1.0)
				}
			}
		}

		if c.Table != nil {
			tableConsumedCapacity = aws.Float64Value(c.Table.CapacityUnits)
			metrics.Gauge("unreads.consumed_capacity.consumed_by_table",
				tableConsumedCapacity, tags, 1.0)
		}
	}

	shouldLog := (operationType == "read" && dt.enableReadsLogging) || (operationType == "write" && dt.enableWritesLogging)

	if shouldLog {
		// 1 -> recordConsumedCapacity
		// 2 -> AwsTable Operation
		// 3 -> AdapterDynamoTable Operation
		// 4 -> InstrumentedDynamoTable Operation
		// 5 -> Store
		caller := callerFunctionName(5)
		logger.Info(ctx, "at=aws_dynamodb_instrumentation",
			"type", operationType,
			"operation", operation,
			"partition_key", hashKey,
			"tablename", dt.tableName,
			"tot_cc", totalConsumedCapacity,
			"tcc", tableConsumedCapacity,
			"caller", caller,
		)
	}
}

// caller returns the filename and line number that called the function this function was called from and the line number
// n = 1 => caller of caller()
// n = 2 => caller of caller of call()
// etc.
func callerFunctionName(n int) string {
	name := "unknown"
	if pc, _, _, ok := runtime.Caller(n); ok {
		name = filepath.Base(runtime.FuncForPC(pc).Name())
	}
	return name
}

func isValidConsumedCapacityLevel(level string) bool {
	switch level {
	case dynamodb.ReturnConsumedCapacityIndexes:
		return true
	case dynamodb.ReturnConsumedCapacityTotal:
		return true
	case dynamodb.ReturnConsumedCapacityNone:
		return true
	default:
		return false
	}
}

func (dt *AwsTable) DynamoKeyFromAwsDynamoItem(item dynamomodel.Item) (dynamomodel.DynamoKey, error) {
	key := dynamomodel.DynamoKey{}
	hashKey, err := dt.ReadStringFromAwsAttributeValue(item[dt.PrimaryKey.PartitionKey.Name])
	if err != nil {
		return key, err
	}
	key.HashKey = hashKey

	if dt.PrimaryKey.HasSortKey() {
		rangeKey, err := dt.ReadStringFromAwsAttributeValue(item[dt.PrimaryKey.SortKey.Name])
		if err != nil {
			return key, err
		}
		key.RangeKey = rangeKey
	}
	return key, nil
}

func (dt *AwsTable) ReadStringFromAwsAttributeValue(av *dynamodb.AttributeValue) (string, error) {
	if av.S != nil {
		return aws.StringValue(av.S), nil
	}

	if av.N != nil {
		return aws.StringValue(av.N), nil
	}

	return "", fmt.Errorf("only string and numberic attributes are supported as keys")
}

func (dt *AwsTable) buildQuery(opts dynamomodel.QueryOpts) *dynamodb.QueryInput {
	qi := &dynamodb.QueryInput{TableName: dt.desc.TableName}

	// Copy over most fields from opts
	if opts.Limit != 0 {
		qi.Limit = aws.Int64(opts.Limit)
	}
	if opts.Descending {
		qi.ScanIndexForward = aws.Bool(false)
	}
	if opts.IndexName != "" {
		qi.IndexName = aws.String(opts.IndexName)
	}

	if opts.FilterExpression != "" {
		qi.FilterExpression = aws.String(opts.FilterExpression)
	}

	if opts.ProjectionExpression != "" {
		qi.ProjectionExpression = aws.String(opts.ProjectionExpression)
	}
	qi.ExpressionAttributeValues = opts.ExpressionAttributeValues
	qi.ExpressionAttributeNames = opts.ExpressionAttributeNames

	// HashKeys are added as key conditions
	keyCondition := ""
	if opts.HashKey != "" {
		keyCondition += fmt.Sprintf("%s = %s", dt.PrimaryKey.PartitionKey.Name, BuildQueryHashKeyName)

		if qi.ExpressionAttributeValues == nil {
			qi.ExpressionAttributeValues = map[string]*dynamodb.AttributeValue{}
		}
		qi.ExpressionAttributeValues[BuildQueryHashKeyName] = dynamomodel.NewStringAttributeValue(opts.HashKey)

		if opts.KeyConditionExpression != "" {
			keyCondition += " AND "
		}
	}
	keyCondition += opts.KeyConditionExpression
	qi.KeyConditionExpression = aws.String(keyCondition)
	return qi
}
