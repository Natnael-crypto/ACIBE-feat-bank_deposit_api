package collaborate

import (
	"fmt"
	"log/slog"

	ratesdomain "github.com/aci/backend/internal/core/domain/rates"
	"github.com/aci/backend/internal/repository"
)

type api struct {
	Username string
	Password string
	Logger   *slog.Logger
	Store    *repository.Store
	Cache    *Cache
}

func NewAPI(username, password string, store *repository.Store) *api {
	return &api{
		Username: username,
		Password: password,
		Logger:   slog.Default(),
		Store:    store,
		Cache:    NewCache(),
	}
}

type EDIResponse struct {
	Segments []RawSegment837 `json:"segments"`
}
type RawSegment837 map[string]interface{}

type Cache struct {
	Cache map[string]ratesdomain.OutPatientRate
}

func NewCache() *Cache {
	return &Cache{
		Cache: make(map[string]ratesdomain.OutPatientRate),
	}
}
func (c *Cache) Get(key string) (ratesdomain.OutPatientRate, error) {
	rate, ok := c.Cache[key]
	if !ok {
		return ratesdomain.OutPatientRate{}, fmt.Errorf("rate not found")
	}
	return rate, nil
}
func (c *Cache) Set(key string, rate ratesdomain.OutPatientRate) {
	c.Cache[key] = rate
}
