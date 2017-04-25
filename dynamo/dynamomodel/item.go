package dynamomodel

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type Item map[string]*dynamodb.AttributeValue

func NewStringAttributeValue(s string) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{S: aws.String(s)}
}

func NewNumberAttributeValue(n string) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{N: aws.String(n)}
}

func NewBoolAttributeValue(b bool) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{BOOL: aws.Bool(b)}
}

func NewStringSetAttributeValue(ss []string) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{SS: aws.StringSlice(ss)}
}

func NewListAttributeValue(l []*dynamodb.AttributeValue) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{L: l}
}

func NewMapAttributeValue(m map[string]*dynamodb.AttributeValue) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{M: m}
}

func NewByteAttributeValue(b []byte) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{B: b}
}

func NewBinarySetAttributeValue(bs [][]byte) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{BS: bs}
}

func NewNumberSetAttributeValue(ns []string) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{NS: aws.StringSlice(ns)}
}

func NewNullAttributeValue(isNull bool) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{NULL: aws.Bool(isNull)}
}
