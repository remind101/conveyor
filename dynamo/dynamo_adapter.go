package dynamo

import (
	"fmt"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/metrics"
	"github.com/remind101/conveyor/dynamo/awsdynamo"
	"github.com/remind101/conveyor/dynamo/dynamomodel"
	"golang.org/x/net/context"
)

const MaxDynamoBatchDeleteKeys = 25
const MaxDynamoBatchGetKeys = 100

type AdapterDynamoTable struct {
	Desc    *dynamodb.TableDescription
	Table   *awsdynamo.AwsTable
	Retryer request.Retryer
}

func NewDynamoAdapterTable(opts dynamomodel.TableOpts) (*AdapterDynamoTable, error) {

	dt, err := awsdynamo.NewAwsTable(opts)
	if err != nil {
		return nil, err
	}

	return &AdapterDynamoTable{
		Desc:    opts.Desc,
		Table:   dt,
		Retryer: client.DefaultRetryer{NumMaxRetries: 10}, // The default
	}, nil
}

func (dt *AdapterDynamoTable) Query(ctx context.Context, opts dynamomodel.QueryOpts) ([]map[string]*dynamodb.AttributeValue, error) {

	return dt.Table.Query(ctx, opts)
}

func (dt *AdapterDynamoTable) PutItem(ctx context.Context, hashKey, rangeKey string, item dynamomodel.Item) error {
	item = dynamomodel.RemoveEmptyDynamoItems(item)

	return dt.Table.PutItem(ctx, hashKey, rangeKey, item)
}

func (dt *AdapterDynamoTable) ConditionExpressionDeleteItem(ctx context.Context, hashKey, rangeKey string,
	expression dynamomodel.Expression) error {
	return dt.Table.ConditionExpressionDeleteItem(ctx, hashKey, rangeKey, expression)
}

func (dt *AdapterDynamoTable) ConditionExpressionPutItem(ctx context.Context, hashKey, rangeKey string,
	item dynamomodel.Item, expression dynamomodel.Expression) error {
	item = dynamomodel.RemoveEmptyDynamoItems(item)

	return dt.Table.ConditionExpressionPutItem(ctx, hashKey, rangeKey, item, expression)
}

func (dt *AdapterDynamoTable) ConditionExpressionUpdateAttributes(ctx context.Context, hashKey, rangeKey string,
	expr dynamomodel.Expression) error {

	return dt.Table.ConditionExpressionUpdateAttributes(ctx, hashKey, rangeKey, expr)
}

func (dt *AdapterDynamoTable) GetItem(ctx context.Context, hashKey, rangeKey string) (dynamomodel.Item, error) {

	return dt.Table.GetItem(ctx, hashKey, rangeKey)
}

func (dt *AdapterDynamoTable) Create(ctx context.Context) error {

	return dt.Table.Create(ctx)
}

func (dt *AdapterDynamoTable) Delete(ctx context.Context) error {

	return dt.Table.Delete(ctx)
}

func (dt *AdapterDynamoTable) BatchGetDocument(ctx context.Context, keys []dynamomodel.DynamoKey, consistentRead bool, outputs []dynamomodel.Item) ([]error, error) {
	documentErrs := []error{}
	errs := []error{}

	outputBatches := [][]dynamomodel.Item{}

	// TODO: parallelize this: do these batch gets in goroutines
	for i := 0; i < len(keys); i += MaxDynamoBatchGetKeys {
		keyBatch := keys[i:min(i+MaxDynamoBatchGetKeys, len(keys))]
		outputBatch := make([]dynamomodel.Item, len(keyBatch))

		thisBatchErrs, err := dt.batchGetDocument(ctx, keyBatch, consistentRead, &outputBatch)
		outputBatches = append(outputBatches, outputBatch)
		documentErrs = append(documentErrs, thisBatchErrs...)
		if err != nil {
			errs = append(errs, err)
		}
	}

	outputsIndex := 0
	for _, outputBatch := range outputBatches {
		for i := 0; i < len(outputBatch); i++ {
			if outputBatch[i] == nil {
				continue
			}
			outputs[outputsIndex] = outputBatch[i]
			outputsIndex++
		}
	}

	var err error
	if len(errs) > 0 {
		errStrings := make([]string, len(errs))
		for i := range errs {
			errStrings[i] = errs[i].Error()
		}
		err = fmt.Errorf("batch get errors: %#v", errStrings)
	}
	return documentErrs, err
}

func (dt *AdapterDynamoTable) batchGetDocument(ctx context.Context, keys []dynamomodel.DynamoKey, consistentRead bool, outputs *[]dynamomodel.Item) ([]error, error) {

	processed := make(map[dynamomodel.DynamoKey]bool)
	errs := make([]error, len(*outputs))

	numRetries := 0

	for {

		// Build the batch
		batch := []dynamomodel.DynamoKey{}
		for _, key := range keys {
			if _, ok := processed[key]; ok {
				continue
			}
			batch = append(batch, key)
		}

		// Make the request
		bgio, bgdReq, bgdErr := dt.Table.BatchGetDocument(ctx, batch, consistentRead)
		if bgdErr != nil {
			return nil, bgdErr
		}

		// Handle successful responses
		responses := make(map[dynamomodel.DynamoKey]dynamomodel.Item)
		for _, item := range bgio.Responses[dt.tableName()] {
			key, err := dt.Table.DynamoKeyFromAwsDynamoItem(item)
			if err != nil {
				return nil, err
			}
			dt.deleteKeyFromItem(item)
			responses[key] = item
			*outputs = append(*outputs, item)
		}

		// Handle unprocessed keys
		unprocessed := make(map[dynamomodel.DynamoKey]bool)
		numUnprocessed := 0
		if r, ok := bgio.UnprocessedKeys[dt.tableName()]; ok {
			for _, item := range r.Keys {
				key, err := dt.Table.DynamoKeyFromAwsDynamoItem(item)
				if err != nil {
					return nil, err
				}
				unprocessed[key] = true
				numUnprocessed++
			}
		}

		// Package the responses maintaining the original ordering as specified by the caller
		// Set ErrNotProcessed for all unprocessed in case we don't retry
		for i, key := range keys {
			if _, ok := processed[key]; ok {
				continue
			}

			if _, ok := unprocessed[key]; !ok {
				errs[i] = awsdynamo.ErrNotFound
				processed[key] = true
			} else {
				errs[i] = awsdynamo.ErrNotProcessed
			}
		}

		bgdReq.RetryCount = numRetries
		if numUnprocessed == 0 || !dt.Retryer.ShouldRetry(bgdReq) {

			return errs, nil
		}

		// http://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_BatchGetItem.html
		// If none of the items can be processed due to insufficient provisioned throughput on all of the tables in the
		// request, then BatchGetItem will return a ProvisionedThroughputExceededException. If at least one of the items is
		// successfully processed, then BatchGetItem completes successfully, while returning the keys of the unread items in
		// UnprocessedKeys.

		// The Aws SDK provides top level (network) retrying using the default retryer:
		// https://github.com/aws/aws-sdk-go/blob/101d2e228fea0ab462a7e0180c607290c4850f15/aws/client/default_retryer.go
		// However, we still need to backoff in the case of throttling.
		// Sleep according to the retry strategy and then attempt again with the remaining keys

		d := dt.Retryer.RetryRules(bgdReq)
		logger.Debug(ctx, "at=batch-retry",
			"operation", "BatchGetDocument",
			"tablename", dt.tableName(),
			"num_retries", numRetries,
			"delay_duration", d,
		)

		metrics.Count("aws.dynamo.request_retry", 1, map[string]string{"operation": "BatchDeleteDocument", "table_name": dt.tableName()}, 1.0)

		time.Sleep(d)
		numRetries++
	}
}

func (dt *AdapterDynamoTable) BatchPutDocument(ctx context.Context, keys []dynamomodel.DynamoKey, v interface{}) ([]error, error) {
	numKeys := len(keys)

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("v must be a slice")
	} else if rv.Len() != numKeys {
		return nil, fmt.Errorf("v must be a slice with the same length as keys")
	}

	processed := make(map[dynamomodel.DynamoKey]bool)
	errs := make([]error, numKeys)

	numRetries := 0
	for {
		itemBatch := []dynamomodel.Item{}
		keyBatch := []dynamomodel.DynamoKey{}
		for i, key := range keys {
			if _, ok := processed[key]; ok {
				continue
			}
			object := rv.Index(i).Interface()
			item, err := dynamomodel.ToDynamo(object)
			if err != nil {
				return nil, err
			}
			itemBatch = append(itemBatch, item)
			keyBatch = append(keyBatch, key)
		}

		bwio, bpdReq, bpdErr := dt.Table.BatchPutDocument(ctx, keyBatch, itemBatch)
		if bpdErr != nil {
			return nil, bpdErr
		}

		unprocessed := make(map[dynamomodel.DynamoKey]bool)
		numUnprocessed := 0
		if writeRequests, ok := bwio.UnprocessedItems[dt.tableName()]; ok {
			for _, writeRequest := range writeRequests {
				key, err := dt.Table.DynamoKeyFromAwsDynamoItem(writeRequest.PutRequest.Item)
				if err != nil {
					return nil, err
				}

				unprocessed[key] = true
				numUnprocessed++
			}
		}

		for i, key := range keys {
			if _, ok := processed[key]; ok {
				continue
			}

			if _, ok := unprocessed[key]; ok {
				errs[i] = awsdynamo.ErrNotProcessed
			} else {
				// Was successfully processed
				errs[i] = nil
				processed[key] = true
			}
		}

		bpdReq.RetryCount = numRetries
		if numUnprocessed == 0 || !dt.Retryer.ShouldRetry(bpdReq) {
			return errs, nil
		}

		d := dt.Retryer.RetryRules(bpdReq)
		logger.Debug(ctx, "at=batch-retry",
			"operation", "BatchPutDocument",
			"tablename", dt.tableName(),
			"num_retries", numRetries,
			"delay_duration", d,
		)

		metrics.Count("aws.dynamo.request_retry", 1, map[string]string{"operation": "BatchPutDocument", "table_name": dt.tableName()}, 1.0)

		time.Sleep(d)
		numRetries++
	}

	return nil, nil
}

func (dt *AdapterDynamoTable) BatchDeleteDocument(ctx context.Context, keys []dynamomodel.DynamoKey) ([]error, error) {
	documentErrs := []error{}
	errs := []error{}
	// Group into batches of MaxDynamoBatchDeleteKeys because dynamo doesn't allow
	// deleting more than that many keys.
	for i := 0; i < len(keys); i += MaxDynamoBatchDeleteKeys {
		batch := keys[i:min(i+MaxDynamoBatchDeleteKeys, len(keys))]
		thisBatchErrs, err := dt.batchDeleteDocument(ctx, batch)
		documentErrs = append(documentErrs, thisBatchErrs...)
		if err != nil {
			errs = append(errs, err)
		}
	}

	var err error
	if len(errs) > 0 {
		err = fmt.Errorf("batch insert errors: %#v", errs)
	}
	return documentErrs, err
}

func (dt *AdapterDynamoTable) batchDeleteDocument(ctx context.Context, keys []dynamomodel.DynamoKey) ([]error, error) {
	numKeys := len(keys)

	processed := make(map[dynamomodel.DynamoKey]bool)
	errs := make([]error, numKeys)

	numRetries := 0

	for {
		batch := []dynamomodel.DynamoKey{}
		for _, key := range keys {
			if _, ok := processed[key]; ok {
				continue
			}
			batch = append(batch, key)
		}

		bwio, bddReq, bddErr := dt.Table.BatchDeleteDocument(ctx, batch)
		if bddErr != nil {
			return nil, bddErr
		}

		unprocessed := make(map[dynamomodel.DynamoKey]bool)
		numUnprocessed := 0
		if writeRequests, ok := bwio.UnprocessedItems[dt.tableName()]; ok {
			for _, writeRequest := range writeRequests {
				key, err := dt.Table.DynamoKeyFromAwsDynamoItem(writeRequest.DeleteRequest.Key)
				if err != nil {
					return nil, err
				}

				unprocessed[key] = true
				numUnprocessed++
			}
		}

		for i, key := range keys {
			if _, ok := processed[key]; ok {
				continue
			}

			if _, ok := unprocessed[key]; ok {
				errs[i] = awsdynamo.ErrNotProcessed
			} else {
				errs[i] = nil
				processed[key] = true
			}
		}

		bddReq.RetryCount = numRetries
		if numUnprocessed == 0 || !dt.Retryer.ShouldRetry(bddReq) {
			return errs, nil
		}

		d := dt.Retryer.RetryRules(bddReq)
		logger.Debug(ctx, "at=batch-retry",
			"operation", "BatchDeleteDocument",
			"tablename", dt.tableName(),
			"num_retries", numRetries,
			"delay_duration", d,
		)

		metrics.Count("aws.dynamo.request_retry", 1, map[string]string{"operation": "BatchDeleteDocument", "table_name": dt.tableName()}, 1.0)

		time.Sleep(d)
		numRetries++
	}
}

func IsFailedConditionalCheck(err error) bool {
	// Support both Goamz and Aws Error Versions for now
	switch err := err.(type) {
	case awserr.Error:
		return err.Code() == "ConditionalCheckFailedException"
	default:
		return false
	}
}

func (dt *AdapterDynamoTable) deleteKeyFromItem(item dynamomodel.Item) {
	delete(item, dt.Table.PrimaryKey.PartitionKey.Name)
	if dt.Table.PrimaryKey.HasSortKey() {
		delete(item, dt.Table.PrimaryKey.SortKey.Name)
	}
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (dt *AdapterDynamoTable) tableName() string {
	return aws.StringValue(dt.Desc.TableName)
}
