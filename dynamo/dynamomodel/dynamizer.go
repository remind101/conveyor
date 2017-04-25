package dynamomodel

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/cbroglie/mapstructure"
)

// This is a modified version of
//https://github.com/AdRoll/goamz/blob/34e96a017307e0442cd8e5f0c15e199c9934563f/dynamodb/dynamizer/dynamizer.go#L76-L138
// to unmarshal a dynamo item into the passed in type
func FromDynamo(item map[string]*dynamodb.AttributeValue, v interface{}) (err error) {

	// Clean up errors
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(runtime.Error); ok {
				err = e
			} else if s, ok := r.(string); ok {
				err = errors.New(s)
			} else {
				err = r.(error)
			}
			item = nil
		}
	}()

	// Use aws sdk UnmarshalMap on the Item
	m := make(map[string]interface{})
	err = dynamodbattribute.UnmarshalMap(item, &m)
	if err != nil {
		return err
	}

	// Handle the case where v is already a reflect.Value object representing a
	// struct or map.
	rv, ok := v.(reflect.Value)
	var rt reflect.Type
	if ok {
		rt = rv.Type()
		if !rv.CanAddr() {
			return fmt.Errorf("v is not addressable")
		}
	} else {
		rv = reflect.ValueOf(v)
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			if rv.IsValid() {
				return fmt.Errorf("v must be a non-nil pointer to a map[string]interface{} or struct (or an addressable reflect.Value), got %s", rv.Type().String())
			}
			return fmt.Errorf("v must be a non-nil pointer to a map[string]interface{} or struct (or an addressable reflect.Value), got zero-value")
		}
		rt = rv.Type()
		rv = rv.Elem()
	}
	_ = rt

	// Use cbroglie/mapstructure to decode the map values from UnMarshalMap
	// into the native Go structure of the interface passed in
	switch rv.Kind() {
	case reflect.Struct:
		config := &mapstructure.DecoderConfig{
			TagName: "json",
			Result:  v,
		}
		decoder, err := mapstructure.NewDecoder(config)
		if err != nil {
			return err
		}
		return decoder.Decode(m)
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("v must be a non-nil pointer to a map[string]interface{} or struct (or an addressable reflect.Value), got %s", rt.String())
		}
		rv.Set(reflect.ValueOf(m))
	default:
		return fmt.Errorf("v must be a non-nil pointer to a map[string]interface{} or struct (or an addressable reflect.Value), got %s", rt.String())
	}

	return nil
}

// Modified version of
// https://github.com/AdRoll/goamz/blob/34e96a017307e0442cd8e5f0c15e199c9934563f/dynamodb/dynamizer/dynamizer.go#L37-L69
func ToDynamo(in interface{}) (item map[string]*dynamodb.AttributeValue, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(runtime.Error); ok {
				err = e
			} else if s, ok := r.(string); ok {
				err = errors.New(s)
			} else {
				err = r.(error)
			}
			item = nil
		}
	}()

	v := reflect.ValueOf(in)
	switch v.Kind() {
	case reflect.Struct:
		item, err = dynamizeStruct(in)
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("item must be a map[string]interface{}, got %s", v.Type().String())
		}
		item, err = dynamizeMap(in)
	case reflect.Ptr:
		if v.IsNil() {
			return nil, fmt.Errorf("item must not be nil")
		}
		return ToDynamo(v.Elem().Interface())
	default:
		return nil, fmt.Errorf("item must be a map[string]interface{} or struct (or a non-nil pointer to one), got %s", v.Type().String())
	}
	return item, err
}

func dynamizeMap(in interface{}) (map[string]*dynamodb.AttributeValue, error) {
	item := make(map[string]*dynamodb.AttributeValue)
	m := in.(map[string]interface{})
	for k, v := range m {
		var err error
		encoder := dynamodbattribute.NewEncoder()
		// Setting NullEmptyString is required for FromDynamo and ToDynamo to be inverses
		// For the struct tests
		encoder.NullEmptyString = false
		item[k], err = encoder.Encode(v)
		if err != nil {
			return nil, err
		}
	}
	return item, nil
}

func dynamizeStruct(in interface{}) (map[string]*dynamodb.AttributeValue, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(b))
	err = decoder.Decode(&m)
	if err != nil {
		return nil, err
	}

	return dynamizeMap(m)
}

func RemoveEmptyDynamoItems(originalItem map[string]*dynamodb.AttributeValue) map[string]*dynamodb.AttributeValue {
	item := map[string]*dynamodb.AttributeValue{}

	for k, v := range originalItem {
		if v.S != nil {
			if aws.StringValue(v.S) != "" {
				item[k] = v
			}
			continue
		}
		if v.M != nil {
			item[k] = removeEmptyMapAttributeValues(*v)
			continue
		}
		item[k] = v
	}
	return item
}

func removeEmptyMapAttributeValues(originalAv dynamodb.AttributeValue) *dynamodb.AttributeValue {
	av := dynamodb.AttributeValue{M: map[string]*dynamodb.AttributeValue{}}
	for k, v := range originalAv.M {
		if v.S != nil {
			if aws.StringValue(v.S) != "" {
				av.M[k] = v
			}
			continue
		}

		av.M[k] = v
	}
	return &av
}
