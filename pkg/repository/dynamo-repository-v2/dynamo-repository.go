package dynamorepository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"

	"github.com/aludrey/go-aludrey-libs/pkg/repository"
)

type DynamoRepository[T interface{}] struct {
	db        dynamodbiface.DynamoDBAPI
	tableName *string
	keys      []string
}

var clients = make(map[string]*dynamodb.DynamoDB)

func NewDynamoRepository[T interface{}](region string, tableName string, keys []string) repository.Repository[T] {
	return newDynamoRepository[T](getClient(region), tableName, keys)
}

// NewDynamoConditionalRepository es NewDynamoRepository con escrituras condicionales (CreateIf, DeleteIf).
func NewDynamoConditionalRepository[T interface{}](region string, tableName string, keys []string) repository.ConditionalRepository[T] {
	return newDynamoRepository[T](getClient(region), tableName, keys)
}

func newDynamoRepository[T interface{}](db dynamodbiface.DynamoDBAPI, tableName string, keys []string) *DynamoRepository[T] {
	return &DynamoRepository[T]{
		tableName: aws.String(tableName),
		db:        db,
		keys:      keys,
	}
}

func (r *DynamoRepository[T]) FindAll(filters map[string]([]string)) ([]T, error) {
	result := []T{}

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
	expressionAttributeNames["#deleted_at"] = aws.String("deleted_at")
	expressionAttributeValues[":null"] = &dynamodb.AttributeValue{NULL: aws.Bool(true)}
	filterExpressions = append(filterExpressions, "(attribute_not_exists(#deleted_at) or #deleted_at = :null)")

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

func (r *DynamoRepository[T]) FindById(id map[string]string) (*T, error) {
	var result T
	output, err := r.db.GetItem(&dynamodb.GetItemInput{
		TableName: r.tableName,
		Key:       buildKeyAttributes(id, r.keys),
	})
	if err != nil {
		return nil, err
	}
	if output.Item == nil || (output.Item["deleted_at"] != nil && output.Item["deleted_at"].S != nil) {
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
	item["created_at"] = &dynamodb.AttributeValue{S: aws.String(time.Now().Format(time.RFC3339))}
	item["updated_at"] = &dynamodb.AttributeValue{S: aws.String(time.Now().Format(time.RFC3339))}

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
	if err != nil {
		return nil, err
	}

	keys := map[string]*dynamodb.AttributeValue{}
	doNotUpdate := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"deleted_at": true,
	}
	for _, key := range r.keys {
		i, ok := item[key]
		if !ok {
			return nil, errors.New("missing key " + key)
		}
		keys[key] = i
		doNotUpdate[key] = true
	}

	expressionAttributeValues := map[string]*dynamodb.AttributeValue{
		":updated_at": {S: aws.String(time.Now().Format(time.RFC3339))},
	}
	expressionAttributeNames := map[string]*string{
		"#updated_at": aws.String("updated_at"),
	}

	updateExpression := "SET #updated_at = :updated_at"

	for k, v := range item {
		if doNotUpdate[k] {
			continue
		}
		updateExpression += ", #" + strings.ToLower(k) + " = :" + strings.ToLower(k)
		expressionAttributeValues[":"+strings.ToLower(k)] = v
		expressionAttributeNames["#"+strings.ToLower(k)] = aws.String(k)
	}

	input := &dynamodb.UpdateItemInput{
		TableName:                 r.tableName,
		Key:                       keys,
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  expressionAttributeNames,
	}

	_, err = r.db.UpdateItem(input)
	if err != nil {
		return nil, err
	}

	return &entity, nil
}

func (r *DynamoRepository[T]) Delete(id map[string]string) error {
	_, err := r.db.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: r.tableName,
		Key:       buildKeyAttributes(id, r.keys),
	})
	return err
}

func (r *DynamoRepository[T]) SoftDelete(id map[string]string) error {
	t, err := r.FindById(id)
	if err != nil {
		return err
	}
	item, err := dynamodbattribute.MarshalMap(*t)
	item["deleted_at"] = &dynamodb.AttributeValue{S: aws.String(time.Now().Format(time.RFC3339))}
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

// CreateIf es Create con una condición que DynamoDB evalúa de forma atómica junto con el PutItem
// (p. ej. "attribute_not_exists(#id)" para no pisar un registro existente).
func (r *DynamoRepository[T]) CreateIf(ctx context.Context, entity T, condition repository.Condition) (*T, error) {
	item, err := dynamodbattribute.MarshalMap(entity)
	if err != nil {
		return nil, err
	}
	now := aws.String(time.Now().Format(time.RFC3339))
	item["created_at"] = &dynamodb.AttributeValue{S: now}
	item["updated_at"] = &dynamodb.AttributeValue{S: now}

	names, values, err := buildConditionAttributes(condition)
	if err != nil {
		return nil, err
	}
	_, err = r.db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName:                 r.tableName,
		Item:                      item,
		ConditionExpression:       aws.String(condition.Expression),
		ExpressionAttributeNames:  names,
		ExpressionAttributeValues: values,
	})
	if err != nil {
		return nil, conditionalWriteError(err)
	}
	return &entity, nil
}

// DeleteIf borra el registro solo si se cumple la condición, en un único DeleteItem atómico. Sirve para
// consumir registros de un solo uso: de dos llamadas concurrentes, una borra y la otra recibe
// ErrConditionFailed.
func (r *DynamoRepository[T]) DeleteIf(ctx context.Context, id map[string]string, condition repository.Condition) error {
	names, values, err := buildConditionAttributes(condition)
	if err != nil {
		return err
	}
	_, err = r.db.DeleteItemWithContext(ctx, &dynamodb.DeleteItemInput{
		TableName:                 r.tableName,
		Key:                       buildKeyAttributes(id, r.keys),
		ConditionExpression:       aws.String(condition.Expression),
		ExpressionAttributeNames:  names,
		ExpressionAttributeValues: values,
	})
	return conditionalWriteError(err)
}

func getClient(region string) *dynamodb.DynamoDB {
	if client, ok := clients[region]; ok {
		return client
	}
	sess := session.Must(session.NewSession())
	db := dynamodb.New(sess, aws.NewConfig().WithRegion(region))
	clients[region] = db
	return db
}

func buildKeyAttributes(id map[string]string, acceptedKeys []string) map[string]*dynamodb.AttributeValue {
	keyAttributes := make(map[string]*dynamodb.AttributeValue)
	for _, v := range acceptedKeys {
		value, ok := id[v]
		if !ok {
			continue
		}
		keyAttributes[v] = &dynamodb.AttributeValue{S: aws.String(value)}
	}
	return keyAttributes
}

// buildConditionAttributes traduce los nombres y valores de la condición al formato de DynamoDB. Una
// condición sin expresión se rechaza: una escritura "condicional" sin condición no protege nada.
func buildConditionAttributes(condition repository.Condition) (map[string]*string, map[string]*dynamodb.AttributeValue, error) {
	if strings.TrimSpace(condition.Expression) == "" {
		return nil, nil, errors.New("condition expression is required")
	}
	var names map[string]*string
	if len(condition.Names) > 0 {
		names = make(map[string]*string, len(condition.Names))
		for placeholder, attribute := range condition.Names {
			names[placeholder] = aws.String(attribute)
		}
	}
	var values map[string]*dynamodb.AttributeValue
	if len(condition.Values) > 0 {
		values = make(map[string]*dynamodb.AttributeValue, len(condition.Values))
		for placeholder, value := range condition.Values {
			attribute, err := dynamodbattribute.Marshal(value)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid condition value %s: %w", placeholder, err)
			}
			values[placeholder] = attribute
		}
	}
	return names, values, nil
}

// conditionalWriteError separa la condición no cumplida (ErrConditionFailed) de un fallo real.
func conditionalWriteError(err error) error {
	if err == nil {
		return nil
	}
	var awsErr awserr.Error
	if errors.As(err, &awsErr) && awsErr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
		return repository.ErrConditionFailed
	}
	return err
}
