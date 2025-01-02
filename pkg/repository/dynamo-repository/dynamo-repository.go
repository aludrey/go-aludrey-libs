package dynamorepository

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/aludrey/go-aludrey-libs/pkg/repository"
)

var clients = make(map[string]*dynamodb.DynamoDB)

func getClient(region string) *dynamodb.DynamoDB {
	if client, ok := clients[region]; ok {
		return client
	}
	sess := session.Must(session.NewSession())
	db := dynamodb.New(sess, aws.NewConfig().WithRegion(region))
	clients[region] = db
	return db
}

type DynamoRepository[T interface{}] struct {
	db        *dynamodb.DynamoDB
	tableName *string
}

func NewDynamoRepository[T interface{}](region string, tableName string) repository.Repository[T] {
	client := getClient(region)
	return &DynamoRepository[T]{
		tableName: aws.String(tableName),
		db:        client,
	}
}

func (r *DynamoRepository[T]) FindAll(filters map[string]([]string)) ([]T, error) {
	var result []T

	filterExpressions := make([]string, 0)
	expressionAttributeValues := make(map[string]*dynamodb.AttributeValue)
	expressionAttributeNames := make(map[string]*string)

	for k, v := range filters {
		internalFilterExpression := make([]string, 0)
		for i, value := range v {
			internalKey := k + "_" + strconv.Itoa(i)
			internalFilterExpression = append(internalFilterExpression, "#"+internalKey+" = :"+internalKey)
			expressionAttributeValues[":"+internalKey] = &dynamodb.AttributeValue{S: aws.String(value)}
			expressionAttributeNames["#"+internalKey] = aws.String(k)
		}
		filterExpressions = append(filterExpressions, "("+strings.Join(internalFilterExpression, " OR ")+")")
	}
	expressionAttributeNames["#DeletedAt"] = aws.String("DeletedAt")
	expressionAttributeValues[":null"] = &dynamodb.AttributeValue{NULL: aws.Bool(true)}
	filterExpressions = append(filterExpressions, "(attribute_not_exists(#DeletedAt) or #DeletedAt = :null)")

	output, err := r.db.Scan(&dynamodb.ScanInput{
		TableName:                 r.tableName,
		FilterExpression:          aws.String(strings.Join(filterExpressions, " AND ")),
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  expressionAttributeNames,
	})
	if err != nil {
		return nil, err
	}
	for _, i := range output.Items {
		var item T
		err = dynamodbattribute.UnmarshalMap(i, &item)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func buildKeyAttributes[K map[string]string](id K) map[string]*dynamodb.AttributeValue {
	keyAttributes := make(map[string]*dynamodb.AttributeValue)
	for k, v := range id {
		keyAttributes[k] = &dynamodb.AttributeValue{S: aws.String(v)}
	}
	return keyAttributes
}

func (r *DynamoRepository[T]) FindById(id map[string]string) (*T, error) {
	var result T
	output, err := r.db.GetItem(&dynamodb.GetItemInput{
		TableName: r.tableName,
		Key:       buildKeyAttributes(id),
	})
	if err != nil {
		return nil, err
	}
	if output.Item == nil || (output.Item["DeletedAt"] != nil && output.Item["DeletedAt"].S != nil) {
		return nil, errors.New("item not found")
	}
	err = dynamodbattribute.UnmarshalMap(output.Item, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *DynamoRepository[T]) Create(entity T) (*T, error) {
	item, err := dynamodbattribute.MarshalMap(entity)
	if err != nil {
		return nil, err
	}
	item["CreatedAt"] = &dynamodb.AttributeValue{S: aws.String(time.Now().Format(time.RFC3339))}
	item["UpdatedAt"] = &dynamodb.AttributeValue{S: aws.String(time.Now().Format(time.RFC3339))}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: r.tableName,
	}

	_, err = r.db.PutItem(input)
	if err != nil {
		return nil, err
	}

	return &entity, nil
}

func (r *DynamoRepository[T]) Update(entity T) (*T, error) {
	item, err := dynamodbattribute.MarshalMap(entity)
	item["UpdatedAt"] = &dynamodb.AttributeValue{S: aws.String(time.Now().Format(time.RFC3339))}

	if err != nil {
		return nil, err
	}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: r.tableName,
	}

	_, err = r.db.PutItem(input)
	if err != nil {
		return nil, err
	}

	return &entity, nil
}

func (r *DynamoRepository[T]) Delete(id map[string]string) error {
	_, err := r.db.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: r.tableName,
		Key:       buildKeyAttributes(id),
	})
	return err
}

func (r *DynamoRepository[T]) SoftDelete(id map[string]string) error {
	t, err := r.FindById(id)
	if err != nil {
		return err
	}
	item, err := dynamodbattribute.MarshalMap(*t)
	item["DeletedAt"] = &dynamodb.AttributeValue{S: aws.String(time.Now().Format(time.RFC3339))}
	if err != nil {
		return err
	}
	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: r.tableName,
	}
	_, err = r.db.PutItem(input)
	if err != nil {
		return err
	}
	return nil
}
