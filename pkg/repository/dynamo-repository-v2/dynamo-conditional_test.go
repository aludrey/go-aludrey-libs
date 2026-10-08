package dynamorepository

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/stretchr/testify/assert"

	"github.com/aludrey/go-aludrey-libs/pkg/repository"
)

type nonceEntity struct {
	Nonce     string `dynamodbav:"nonce"`
	ExpiresAt int64  `dynamodbav:"expires_at"`
}

type fakeDynamo struct {
	dynamodbiface.DynamoDBAPI
	put    *dynamodb.PutItemInput
	delete *dynamodb.DeleteItemInput
	err    error
}

func (f *fakeDynamo) PutItemWithContext(_ aws.Context, in *dynamodb.PutItemInput, _ ...request.Option) (*dynamodb.PutItemOutput, error) {
	f.put = in
	return &dynamodb.PutItemOutput{}, f.err
}

func (f *fakeDynamo) DeleteItemWithContext(_ aws.Context, in *dynamodb.DeleteItemInput, _ ...request.Option) (*dynamodb.DeleteItemOutput, error) {
	f.delete = in
	return &dynamodb.DeleteItemOutput{}, f.err
}

var conditionalCheckFailed = awserr.New(dynamodb.ErrCodeConditionalCheckFailedException, "The conditional request failed", nil)

func TestCreateIfSendsConditionAndItem(t *testing.T) {
	db := &fakeDynamo{}
	repo := newDynamoRepository[nonceEntity](db, "nonces", []string{"nonce"})

	result, err := repo.CreateIf(context.Background(), nonceEntity{Nonce: "abc", ExpiresAt: 100}, repository.Condition{
		Expression: "attribute_not_exists(#nonce)",
		Names:      map[string]string{"#nonce": "nonce"},
	})

	assert.NoError(t, err)
	assert.Equal(t, "abc", result.Nonce)
	assert.Equal(t, "nonces", aws.StringValue(db.put.TableName))
	assert.Equal(t, "attribute_not_exists(#nonce)", aws.StringValue(db.put.ConditionExpression))
	assert.Equal(t, "nonce", aws.StringValue(db.put.ExpressionAttributeNames["#nonce"]))
	assert.Nil(t, db.put.ExpressionAttributeValues)
	assert.Equal(t, "abc", aws.StringValue(db.put.Item["nonce"].S))
	assert.Equal(t, "100", aws.StringValue(db.put.Item["expires_at"].N))
	assert.NotNil(t, db.put.Item["created_at"])
}

func TestCreateIfReturnsConditionFailed(t *testing.T) {
	repo := newDynamoRepository[nonceEntity](&fakeDynamo{err: conditionalCheckFailed}, "nonces", []string{"nonce"})

	_, err := repo.CreateIf(context.Background(), nonceEntity{Nonce: "abc"}, repository.Condition{Expression: "attribute_not_exists(nonce)"})

	assert.ErrorIs(t, err, repository.ErrConditionFailed)
}

func TestDeleteIfSendsKeyConditionAndValues(t *testing.T) {
	db := &fakeDynamo{}
	repo := newDynamoRepository[nonceEntity](db, "nonces", []string{"nonce"})

	err := repo.DeleteIf(context.Background(), map[string]string{"nonce": "abc"}, repository.Condition{
		Expression: "attribute_exists(#nonce) AND #expires_at > :now",
		Names:      map[string]string{"#nonce": "nonce", "#expires_at": "expires_at"},
		Values:     map[string]interface{}{":now": int64(1700000000)},
	})

	assert.NoError(t, err)
	assert.Equal(t, "abc", aws.StringValue(db.delete.Key["nonce"].S))
	assert.Equal(t, "attribute_exists(#nonce) AND #expires_at > :now", aws.StringValue(db.delete.ConditionExpression))
	assert.Equal(t, "expires_at", aws.StringValue(db.delete.ExpressionAttributeNames["#expires_at"]))
	assert.Equal(t, "1700000000", aws.StringValue(db.delete.ExpressionAttributeValues[":now"].N))
}

func TestDeleteIfReturnsConditionFailed(t *testing.T) {
	repo := newDynamoRepository[nonceEntity](&fakeDynamo{err: conditionalCheckFailed}, "nonces", []string{"nonce"})

	err := repo.DeleteIf(context.Background(), map[string]string{"nonce": "abc"}, repository.Condition{Expression: "attribute_exists(nonce)"})

	assert.ErrorIs(t, err, repository.ErrConditionFailed)
}

func TestDeleteIfPropagatesOtherErrors(t *testing.T) {
	throttled := awserr.New(dynamodb.ErrCodeProvisionedThroughputExceededException, "slow down", nil)
	repo := newDynamoRepository[nonceEntity](&fakeDynamo{err: throttled}, "nonces", []string{"nonce"})

	err := repo.DeleteIf(context.Background(), map[string]string{"nonce": "abc"}, repository.Condition{Expression: "attribute_exists(nonce)"})

	assert.Error(t, err)
	assert.False(t, errors.Is(err, repository.ErrConditionFailed))
}

func TestConditionalWritesRequireAnExpression(t *testing.T) {
	db := &fakeDynamo{}
	repo := newDynamoRepository[nonceEntity](db, "nonces", []string{"nonce"})

	_, createErr := repo.CreateIf(context.Background(), nonceEntity{Nonce: "abc"}, repository.Condition{})
	deleteErr := repo.DeleteIf(context.Background(), map[string]string{"nonce": "abc"}, repository.Condition{Expression: "  "})

	assert.Error(t, createErr)
	assert.Error(t, deleteErr)
	assert.Nil(t, db.put)
	assert.Nil(t, db.delete)
}

func TestConditionalRepositoryKeepsRepositoryContract(t *testing.T) {
	var conditional repository.ConditionalRepository[nonceEntity] = NewDynamoConditionalRepository[nonceEntity]("us-east-2", "nonces", []string{"nonce"})
	var _ repository.Repository[nonceEntity] = conditional
}
