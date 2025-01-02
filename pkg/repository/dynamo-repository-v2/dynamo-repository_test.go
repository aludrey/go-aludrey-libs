package dynamorepository

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

// SomeEntity represents a generic entity for testing
type SomeEntity struct {
	ID       string
	Name     string
	TestJson string `json:"test_json"`
}

func TestDynamoRepository_Create(t *testing.T) {

	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table", []string{"ID"})

	entity := SomeEntity{
		ID:   "1",
		Name: "Fanta",
	}

	result, err := repo.Create(entity)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "1", result.ID)
	assert.Equal(t, "Fanta", result.Name)
}

func TestDynamoRepository_FindAll(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table", []string{"ID"})

	filters := map[string][]string{"Name": {"Test Name", "Test Name 2", "Fanta"}}
	result, err := repo.FindAll(filters)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "1", result[0].ID)
	assert.Equal(t, "Fanta", result[0].Name)
}

func TestDynamoRepository_FindAllSingleValue(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table", []string{"ID"})

	filters := map[string][]string{"Name": {"Fanta"}}
	result, err := repo.FindAll(filters)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "1", result[0].ID)
	assert.Equal(t, "Fanta", result[0].Name)
}

func TestDynamoRepository_FindAllFail(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table", []string{"ID"})

	filters := map[string][]string{"Name": {"Pepsi", "Orange Juice"}}
	result, err := repo.FindAll(filters)

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestDynamoRepository_FindById(t *testing.T) {

	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table", []string{"ID"})

	id := map[string]string{"ID": "1"}
	result, err := repo.FindById(id)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "1", result.ID)
	assert.Equal(t, "Fanta", result.Name)
}

func TestDynamoRepository_FindByIdFail(t *testing.T) {

	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table", []string{"ID"})

	id := map[string]string{"ID": "99999"}
	result, err := repo.FindById(id)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDynamoRepository_Update(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table", []string{"ID"})

	id := map[string]string{"ID": "1"}
	entity, err := repo.FindById(id)
	if err != nil {
		t.Fatal(err)
	}
	entity.Name = "Updated Name"
	result, err := repo.Update(*entity)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "1", result.ID)
	assert.Equal(t, "Updated Name", result.Name)
}

func TestDynamoRepository_Delete(t *testing.T) {

	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table", []string{"ID"})

	id := map[string]string{"ID": "1"}

	err := repo.Delete(id)

	assert.NoError(t, err)
}

func TestDynamoRepository_SoftDelete(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table", []string{"ID"})

	entity := SomeEntity{
		ID:   "90",
		Name: "Fanta",
	}

	result, err := repo.Create(entity)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Initial entity to be soft deleted
	id := map[string]string{"ID": "90"}

	// Perform the soft delete operation
	err = repo.SoftDelete(id)

	// Check if there was no error in the operation
	assert.NoError(t, err)

	// Retrieve the entity to confirm it was soft deleted
	result, err = repo.FindById(id)
	assert.Error(t, err)
	assert.Nil(t, result)
	err = repo.Delete(id)
	assert.NoError(t, err)
}

func TestDynamoRepository_SoftDeleteFail(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table", []string{"ID"})

	// Try to soft delete a non-existent entity
	id := map[string]string{"ID": "99999"}

	err := repo.SoftDelete(id)

	// Expect an error since the item does not exist
	assert.Error(t, err)
}

func TestDynamoRepository_SoftDelete_ServiceError(t *testing.T) {
	// Simulate a service error by using a non-existent table
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "non-existent-table", []string{"ID"})

	// Attempt to soft delete an item
	id := map[string]string{"ID": "1"}

	err := repo.SoftDelete(id)

	// Expect an error due to the table not existing
	assert.Error(t, err, "Expected an error due to the non-existent DynamoDB table")
}

func TestDynamoRepository_SoftDelete_ExcludedFromFindAll(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table", []string{"ID"})
	entity := SomeEntity{
		ID:   "91",
		Name: "Fanta",
	}
	entity2 := SomeEntity{
		ID:   "92",
		Name: "Fanta",
	}
	entity3 := SomeEntity{
		ID:   "93",
		Name: "Pepsi",
	}
	_, _ = repo.Create(entity)
	_, _ = repo.Create(entity2)
	_, _ = repo.Create(entity3)

	// Create an entity and soft delete it
	id := map[string]string{"ID": "91"}
	err := repo.SoftDelete(id)
	assert.NoError(t, err, "Soft delete should not return an error")

	// Attempt to fetch all items, including the soft-deleted one
	filters := map[string][]string{"Name": {"Fanta"}}
	result, err := repo.FindAll(filters)

	// Ensure no errors occurred during the fetch
	assert.NoError(t, err)

	// Check that the soft-deleted item does not appear in the results
	assert.Len(t, result, 1, "Soft-deleted items should not be included in FindAll results")
	repo.Delete(map[string]string{"ID": "91"})
	repo.Delete(map[string]string{"ID": "92"})
	repo.Delete(map[string]string{"ID": "93"})
}

type SomeEntityWithTimestamps struct {
	ID        string
	Name      string
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	DeletedAt string `json:"deleted_at"`
}

func TestDynamoRepository_Create_AutomaticTimestamps(t *testing.T) {
	repo := NewDynamoRepository[SomeEntityWithTimestamps]("us-east-2", "test-table", []string{"ID"})

	// Create an entity without specifying timestamps
	entity := SomeEntityWithTimestamps{
		ID:   "300",
		Name: "Auto Timestamps Test",
	}
	result, err := repo.Create(entity)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Retrieve the created entity
	retrievedResult, err := repo.FindById(map[string]string{"ID": "300"})
	assert.NoError(t, err)
	assert.NotNil(t, retrievedResult)

	// Ensure `created_at` and `updated_at` were set correctly
	now := time.Now()
	createdAt, _ := time.Parse(time.RFC3339, retrievedResult.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, retrievedResult.UpdatedAt)

	assert.WithinDuration(t, now, createdAt, 5*time.Second, "created_at should be close to current time")
	assert.WithinDuration(t, now, updatedAt, 5*time.Second, "updated_at should be close to current time")
	assert.Empty(t, retrievedResult.DeletedAt, "deleted_at should not be set")

	// Clean up
	err = repo.Delete(map[string]string{"ID": "300"})
	assert.NoError(t, err)
}

func TestDynamoRepository_Update_AutomaticTimestamps(t *testing.T) {
	repo := NewDynamoRepository[SomeEntityWithTimestamps]("us-east-2", "test-table", []string{"ID"})

	// Create an entity
	entity := SomeEntityWithTimestamps{
		ID:   "301",
		Name: "Initial Name",
	}
	_, err := repo.Create(entity)
	time.Sleep(1 * time.Second)
	assert.NoError(t, err)

	// Update the entity
	updatedEntity := SomeEntityWithTimestamps{
		ID:   "301",
		Name: "Updated Name",
	}
	_, err = repo.Update(updatedEntity)
	assert.NoError(t, err)

	// Retrieve the updated entity
	retrievedResult, err := repo.FindById(map[string]string{"ID": "301"})
	assert.NoError(t, err)
	assert.NotNil(t, retrievedResult)

	// Ensure `updated_at` was updated and `created_at` remains unchanged
	createdAt, _ := time.Parse(time.RFC3339, retrievedResult.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, retrievedResult.UpdatedAt)

	now := time.Now()
	assert.WithinDuration(t, now, updatedAt, 5*time.Second, "updated_at should be updated to current time")
	assert.GreaterOrEqual(t, now.Unix()-createdAt.Unix(), int64(1), "created_at should remain the original value")
	assert.Empty(t, retrievedResult.DeletedAt, "deleted_at should not be set")

	// Clean up
	err = repo.Delete(map[string]string{"ID": "301"})
	assert.NoError(t, err)
}

func TestDynamoRepository_SoftDelete_SetsDeletedAt(t *testing.T) {
	repo := NewDynamoRepository[SomeEntityWithTimestamps]("us-east-2", "test-table", []string{"ID"})

	// Create an entity
	entity := SomeEntityWithTimestamps{
		ID:   "302",
		Name: "Soft Delete Test",
	}
	_, err := repo.Create(entity)
	assert.NoError(t, err)

	// Perform a soft delete
	err = repo.SoftDelete(map[string]string{"ID": "302"})
	assert.NoError(t, err)

	// Retrieve the entity
	retrievedResult, err := repo.FindById(map[string]string{"ID": "302"})
	assert.Error(t, err, "Entity should not be retrievable after soft delete")
	assert.Nil(t, retrievedResult)

	sess := session.Must(session.NewSession())
	db := dynamodb.New(sess, aws.NewConfig().WithRegion("us-east-2"))
	output, err := db.Scan(&dynamodb.ScanInput{
		TableName:        aws.String("test-table"),
		FilterExpression: aws.String("ID = :id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {S: aws.String("302")},
		},
	})
	assert.NoError(t, err)
	assert.Len(t, output.Items, 1, "Soft-deleted item should still exist in the table")

	deletedAt, err := time.Parse(time.RFC3339, *output.Items[0]["deleted_at"].S)
	assert.NoError(t, err)
	now := time.Now()
	assert.WithinDuration(t, now, deletedAt, 5*time.Second, "deleted_at should be close to current time")

	// Clean up
	err = repo.Delete(map[string]string{"ID": "302"})
	assert.NoError(t, err)
}
