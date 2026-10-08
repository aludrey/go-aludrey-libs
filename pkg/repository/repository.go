package repository

import (
	"context"
	"errors"
)

// Condition es la condición de una escritura condicional. Expression usa placeholders #nombre para los
// atributos (Names) y :valor para los valores (Values), como las expresiones de DynamoDB.
type Condition struct {
	Expression string
	Names      map[string]string
	Values     map[string]interface{}
}

// ErrConditionFailed lo devuelven las escrituras condicionales cuando la condición no se cumple: el
// registro no se escribió ni se borró. No es un fallo de infraestructura.
var ErrConditionFailed = errors.New("repository: condition failed")

type Repository[T interface{}] interface {
	FindAll(filters map[string][]string) ([]T, error)
	FindById(id map[string]string) (*T, error)
	Create(entity T) (*T, error)
	Update(entity T) (*T, error)
	Delete(id map[string]string) error
	SoftDelete(id map[string]string) error
}

// ConditionalRepository suma escrituras condicionales atómicas: la condición se evalúa en la base junto con
// la escritura, así que dos llamadas concurrentes no pueden cumplirla las dos. Es lo que hace falta para
// registros de un solo uso, donde leer y después borrar deja una ventana en la que ambos pasan.
type ConditionalRepository[T interface{}] interface {
	Repository[T]
	// CreateIf crea el registro solo si se cumple la condición; si no, devuelve ErrConditionFailed.
	CreateIf(ctx context.Context, entity T, condition Condition) (*T, error)
	// DeleteIf borra el registro solo si se cumple la condición; si no, devuelve ErrConditionFailed.
	DeleteIf(ctx context.Context, id map[string]string, condition Condition) error
}
