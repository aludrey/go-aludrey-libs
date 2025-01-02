package repository

type Repository[T interface{}] interface {
	FindAll(filters map[string][]string) ([]T, error)
	FindById(id map[string]string) (*T, error)
	Create(entity T) (*T, error)
	Update(entity T) (*T, error)
	Delete(id map[string]string) error
	SoftDelete(id map[string]string) error
}
