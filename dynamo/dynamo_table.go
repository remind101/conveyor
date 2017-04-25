package dynamo

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/remind101/conveyor/dynamo/dynamomodel"
	"golang.org/x/net/context"
)

type Table interface {
	// Reads
	Query(ctx context.Context, opts dynamomodel.QueryOpts) ([]map[string]*dynamodb.AttributeValue, error)
	GetItem(ctx context.Context, hashKey, rangeKey string) (dynamomodel.Item, error)
	BatchGetDocument(ctx context.Context, keys []dynamomodel.DynamoKey, consistentRead bool, v []dynamomodel.Item) ([]error, error)
	// Writes
	PutItem(ctx context.Context, hashKey, rangeKey string, item dynamomodel.Item) error
	ConditionExpressionPutItem(ctx context.Context, hashKey, rangeKey string, item dynamomodel.Item, expression dynamomodel.Expression) error
	ConditionExpressionDeleteItem(ctx context.Context, hashKey, rangeKey string, expr dynamomodel.Expression) error
	ConditionExpressionUpdateAttributes(ctx context.Context, hashKey, rangeKey string, expr dynamomodel.Expression) error
	BatchPutDocument(ctx context.Context, keys []dynamomodel.DynamoKey, v interface{}) ([]error, error)
	BatchDeleteDocument(ctx context.Context, keys []dynamomodel.DynamoKey) ([]error, error)
	Create(ctx context.Context) error
	Delete(ctx context.Context) error
}

type InstrumentedTable struct {
	Table        Table
	tableName    string
	enableReads  bool
	enableWrites bool
}

func NewInstrumentedTable(opts dynamomodel.TableOpts) (*InstrumentedTable, error) {

	tableName := aws.StringValue(opts.Desc.TableName)

	dt, err := NewDynamoAdapterTable(opts)
	if err != nil {
		return nil, fmt.Errorf("error creating table %s: %#v", tableName, err)
	}

	return &InstrumentedTable{
		Table:        dt,
		enableReads:  opts.EnableReadsLogging,
		enableWrites: opts.EnableWritesLogging,
		tableName:    tableName,
	}, nil
}

func (dt *InstrumentedTable) recordRead(operation, hashKey string) {
	if dt.enableReads {
		fmt.Printf("type=read operation=%s partition_key=%s tablename=%s at=dynamodb_instrumentation\n", operation, hashKey, dt.tableName)
	}
}

func (dt *InstrumentedTable) recordWrite(operation, hashKey string) {
	if dt.enableWrites {
		fmt.Printf("type=write operation=%s partition_key=%s tablename=%s at=dynamodb_instrumentation\n", operation, hashKey, dt.tableName)
	}
}

func (dt *InstrumentedTable) Query(ctx context.Context, opts dynamomodel.QueryOpts) ([]map[string]*dynamodb.AttributeValue, error) {
	dt.recordRead("Query", opts.HashKey)
	return dt.Table.Query(ctx, opts)
}

func (dt *InstrumentedTable) GetItem(ctx context.Context, hashKey, rangeKey string) (dynamomodel.Item, error) {
	dt.recordRead("GetItem", hashKey)
	return dt.Table.GetItem(ctx, hashKey, rangeKey)
}

func (dt *InstrumentedTable) BatchGetDocument(ctx context.Context, keys []dynamomodel.DynamoKey, consistentRead bool, v []dynamomodel.Item) ([]error, error) {
	return dt.Table.BatchGetDocument(ctx, keys, consistentRead, v)
}

func (dt *InstrumentedTable) PutItem(ctx context.Context, hashKey, rangeKey string, item dynamomodel.Item) error {
	dt.recordWrite("PutItem", hashKey)
	return dt.Table.PutItem(ctx, hashKey, rangeKey, item)
}

func (dt *InstrumentedTable) ConditionExpressionPutItem(ctx context.Context, hashKey, rangeKey string, item dynamomodel.Item,
	expression dynamomodel.Expression) error {

	dt.recordWrite("ConditionExpressionPutItem", hashKey)
	return dt.Table.ConditionExpressionPutItem(ctx, hashKey, rangeKey, item, expression)
}

func (dt *InstrumentedTable) ConditionExpressionDeleteItem(ctx context.Context, hashKey, rangeKey string,
	expression dynamomodel.Expression) error {
	dt.recordWrite("ConditionExpressionDeleteItem", hashKey)
	return dt.Table.ConditionExpressionDeleteItem(ctx, hashKey, rangeKey, expression)
}

func (dt *InstrumentedTable) ConditionExpressionUpdateAttributes(ctx context.Context, hashKey, rangeKey string,
	expr dynamomodel.Expression) error {

	dt.recordWrite("ConditionExpressionUpdateAttributes", hashKey)
	return dt.Table.ConditionExpressionUpdateAttributes(ctx, hashKey, rangeKey, expr)
}

func (dt *InstrumentedTable) BatchPutDocument(ctx context.Context, keys []dynamomodel.DynamoKey, v interface{}) ([]error, error) {
	return dt.Table.BatchPutDocument(ctx, keys, v)
}

func (dt *InstrumentedTable) BatchDeleteDocument(ctx context.Context, keys []dynamomodel.DynamoKey) ([]error, error) {
	return dt.Table.BatchDeleteDocument(ctx, keys)
}

func (dt *InstrumentedTable) Create(ctx context.Context) error {
	return dt.Table.Create(ctx)
}

func (dt *InstrumentedTable) Delete(ctx context.Context) error {
	return dt.Table.Delete(ctx)
}
