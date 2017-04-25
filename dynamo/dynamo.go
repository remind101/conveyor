package dynamo

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/remind101/conveyor/dynamo/awsdynamo"
	"github.com/remind101/conveyor/dynamo/dynamomodel"
	"github.com/remind101/conveyor/dynamo/dynamoutils"
	"github.com/remind101/pkg/logger"
)

type Dynamo struct {
	awsDynamo    *dynamodb.DynamoDB
	tablesByName map[string]Table
}

func NewDynamo(params dynamomodel.ConnectionParams) (*Dynamo, error) {
	awsDynamo := awsdynamo.NewAwsDynamo(params)
	tablesByName := map[string]Table{}

	awsTables := awsdynamo.MakeAwsTables(params)
	for tableName, awsDesc := range awsTables {
		dt, err := NewInstrumentedTable(dynamomodel.TableOpts{
			Desc:                awsDesc,
			AwsDynamo:           awsDynamo,
			EnableReadsLogging:  dynamoutils.IsTableEnabledFromEnv("DYNAMO_READ_INSTRUMENTATION", tableName),
			EnableWritesLogging: dynamoutils.IsTableEnabledFromEnv("DYNAMO_WRITE_INSTRUMENTATION", tableName),
		})

		if err != nil {
			return nil, fmt.Errorf("error creating table %s: %#v", tableName, err)
		}
		tablesByName[tableName] = dt
	}
	return &Dynamo{
		awsDynamo:    awsDynamo,
		tablesByName: tablesByName,
	}, nil
}

func NewTestDynamo(params dynamomodel.ConnectionParams) (*Dynamo, error) {
	awsDynamo := awsdynamo.NewTestAwsDynamo(params)
	tablesByName := map[string]Table{}

	awsTables := awsdynamo.MakeAwsTables(params)
	for tableName, awsDesc := range awsTables {
		dt, err := NewInstrumentedTable(dynamomodel.TableOpts{
			Desc:                awsDesc,
			AwsDynamo:           awsDynamo,
			EnableReadsLogging:  dynamoutils.IsTableEnabledFromEnv("DYNAMO_READ_INSTRUMENTATION", tableName),
			EnableWritesLogging: dynamoutils.IsTableEnabledFromEnv("DYNAMO_WRITE_INSTRUMENTATION", tableName),
		})

		if err != nil {
			return nil, fmt.Errorf("error creating table %s: %#v", tableName, err)
		}
		tablesByName[tableName] = dt
	}
	return &Dynamo{
		awsDynamo:    awsDynamo,
		tablesByName: tablesByName,
	}, nil
}

func (d *Dynamo) GetTable(tableName string) Table {
	return d.tablesByName[tableName]
}

func (d *Dynamo) CreateTables() error {
	var err error
	for tableName, table := range d.tablesByName {
		err = table.Create(context.Background())
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "ResourceInUseException" { // table already exists
					logger.Debug(context.Background(), "table", tableName, "already exists")
					continue
				}
			}
			return fmt.Errorf("Error creating table %s: %v", tableName, err)
		}
		logger.Debug(context.Background(), "table", tableName, "created successfully")
	}
	return nil
}

func (d *Dynamo) DeleteTables() error {
	var err error
	for tableName, table := range d.tablesByName {
		err = table.Delete(context.Background())
		if err != nil {
			return fmt.Errorf("Error deleting table %s: %v", tableName, err)
		}
	}
	return nil
}
