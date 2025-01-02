package dynamorepository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// SomeEntity represents a generic entity for testing
type SomeEntity struct {
	ID        string
	Name      string
	CreatedAt string
	UpdatedAt string
	DeletedAt string
}

func TestDynamoRepository_Create(t *testing.T) {

	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table")

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
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table")

	filters := map[string][]string{"Name": {"Test Name", "Test Name 2", "Fanta"}}
	result, err := repo.FindAll(filters)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "1", result[0].ID)
	assert.Equal(t, "Fanta", result[0].Name)
}

func TestDynamoRepository_FindAllSingleValue(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table")

	filters := map[string][]string{"Name": {"Fanta"}}
	result, err := repo.FindAll(filters)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "1", result[0].ID)
	assert.Equal(t, "Fanta", result[0].Name)
}

func TestDynamoRepository_FindAllFail(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table")

	filters := map[string][]string{"Name": {"Pepsi", "Orange Juice"}}
	result, err := repo.FindAll(filters)

	assert.Nil(t, err)
	assert.Len(t, result, 0)
}

func TestDynamoRepository_FindById(t *testing.T) {

	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table")

	id := map[string]string{"ID": "1"}
	result, err := repo.FindById(id)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "1", result.ID)
	assert.Equal(t, "Fanta", result.Name)
}

func TestDynamoRepository_FindByIdFail(t *testing.T) {

	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table")

	id := map[string]string{"ID": "99999"}
	result, err := repo.FindById(id)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDynamoRepository_Update(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table")

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

	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table")

	id := map[string]string{"ID": "1"}

	err := repo.Delete(id)

	assert.NoError(t, err)
}

func TestDynamoRepository_SoftDelete(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table")

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
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table")

	// Try to soft delete a non-existent entity
	id := map[string]string{"ID": "99999"}

	err := repo.SoftDelete(id)

	// Expect an error since the item does not exist
	assert.Error(t, err)
}

func TestDynamoRepository_SoftDelete_ServiceError(t *testing.T) {
	// Simulate a service error by using a non-existent table
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "non-existent-table")

	// Attempt to soft delete an item
	id := map[string]string{"ID": "1"}

	err := repo.SoftDelete(id)

	// Expect an error due to the table not existing
	assert.Error(t, err, "Expected an error due to the non-existent DynamoDB table")
}

func TestDynamoRepository_SoftDelete_ExcludedFromFindAll(t *testing.T) {
	repo := NewDynamoRepository[SomeEntity]("us-east-2", "test-table")
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
