package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ApiResponse represents the standard API response structure
type ApiResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// SendResponse sends a standardized response
func SendResponse(c *gin.Context, statusCode int, success bool, message string, data interface{}, err string) {
	response := ApiResponse{
		Success:   success,
		Message:   message,
		Data:      data,
		Error:     err,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	c.JSON(statusCode, response)
}

// SendSuccess sends a successful response
func SendSuccess(c *gin.Context, message string, data interface{}, statusCode ...int) {
	code := http.StatusOK
	if len(statusCode) > 0 {
		code = statusCode[0]
	}
	SendResponse(c, code, true, message, data, "")
}

// SendError sends an error response
func SendError(c *gin.Context, message string, statusCode int, err ...string) {
	errorMsg := ""
	if len(err) > 0 {
		errorMsg = err[0]
	}
	SendResponse(c, statusCode, false, message, nil, errorMsg)
}

// SendNotFound sends a 404 not found response
func SendNotFound(c *gin.Context, message string) {
	SendError(c, message, http.StatusNotFound, "Resource does not exist")
}

// SendUnauthorized sends a 401 unauthorized response
func SendUnauthorized(c *gin.Context, message string) {
	SendError(c, message, http.StatusUnauthorized, "Authentication required")
}

// SendConflict sends a 409 conflict response
func SendConflict(c *gin.Context, message string) {
	SendError(c, message, http.StatusConflict, "Resource already exists")
}

// SendValidationError sends a 400 validation error response
func SendValidationError(c *gin.Context, message string, errors ...interface{}) {
	errorMsg := "Validation failed"
	if len(errors) > 0 {
		errorMsg = "Validation errors"
	}
	SendError(c, message, http.StatusBadRequest, errorMsg)
}

// SendInternalError sends a 500 internal server error response
func SendInternalError(c *gin.Context, message string) {
	SendError(c, message, http.StatusInternalServerError, "Internal server error")
}

