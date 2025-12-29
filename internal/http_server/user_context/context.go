package usercontext

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const userContextKey = "user"

type UserContext struct {
	Id uuid.UUID
	// Role      domain.UserRole
	SessionId uuid.UUID
	Token     string
}

func ContextSetUser(c *gin.Context, user UserContext) {
	c.Set(userContextKey, user)
}

// panic is on purpose->it must never happen
func ContextGetUser(c *gin.Context) UserContext {
	val, exists := c.Get(userContextKey)
	if !exists {
		panic("missing user value in gin context")
	}
	user, ok := val.(UserContext)
	if !ok {
		panic("invalid user value in gin context")
	}
	return user
}
