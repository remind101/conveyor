package dynamomodel

import (
	"strconv"

	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type DynamoKey struct {
	HashKey  string
	RangeKey string
}

type PrimaryKey struct {
	PartitionKey *Key
	SortKey      *Key
}

type Key struct {
	Type string
	Name string
}

func (PrimaryKey *PrimaryKey) HasSortKey() bool {
	if PrimaryKey.SortKey != nil {
		return true
	}
	return false
}

// Builds an attribute value from a PrimaryKey attribute
func (k *Key) NewAttributeValue(value string) *dynamodb.AttributeValue {
	switch k.Type {
	case dynamodb.ScalarAttributeTypeS:
		return NewStringAttributeValue(value)
	case dynamodb.ScalarAttributeTypeN:
		return NewNumberAttributeValue(value)
	case dynamodb.ScalarAttributeTypeB:
		b, _ := strconv.ParseBool(value)
		return NewBoolAttributeValue(b)
	default:
		return nil
	}
}
