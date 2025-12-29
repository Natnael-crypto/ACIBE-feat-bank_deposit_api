package dashboard

import "github.com/aci/backend/internal/repository"

type repositoryImpl struct {
	store *repository.Store
}

func NewRepository(store *repository.Store) Repository {
	return &repositoryImpl{store: store}
}

