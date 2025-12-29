package rates

import (
	"log/slog"
	"net/http"

	ratesdomain "github.com/aci/backend/internal/core/domain/rates"
	rateservice "github.com/aci/backend/internal/core/services/rates"
	"github.com/aci/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type RateHandler struct {
	service *rateservice.Service
	logger  *slog.Logger
}

func NewRateHandler(service *rateservice.Service, logger *slog.Logger) *RateHandler {
	return &RateHandler{
		service: service,
		logger:  logger,
	}
}

// GetAllOutpatientRates godoc
// @Summary      Get All Outpatient Rates
// @Description  Retrieve all outpatient rates from the database
// @Tags         Rates
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /rates/outpatient/all [get]
func (h *RateHandler) GetAllOutpatientRates(c *gin.Context) {
	h.logger.Info("Fetching all outpatient rates")

	rates, err := h.service.GetAllOutPatientRates(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get all outpatient rates", "error", err)
		response.SendError(c, "Failed to retrieve outpatient rates", http.StatusInternalServerError, err.Error())
		return
	}

	response.SendSuccess(c, "Outpatient rates retrieved successfully", rates)
}

// GetOutpatientRate godoc
// @Summary      Get Outpatient Rate by Code and Modifiers
// @Description  Retrieve a specific outpatient rate by code and modifiers
// @Tags         Rates
// @Accept       json
// @Produce      json
// @Param        code      query     string  true  "Rate code"
// @Param        modifiers query     string  false "Rate modifiers (comma-separated)"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /rates/outpatient/by-code [get]
func (h *RateHandler) GetOutpatientRate(c *gin.Context) {
	code := c.Query("code")
	modifiers := c.Query("modifiers")

	if code == "" {
		response.SendError(c, "Code parameter is required", http.StatusBadRequest, "missing code parameter")
		return
	}

	h.logger.Info("Fetching outpatient rate", "code", code, "modifiers", modifiers)

	rate, err := h.service.GetOutPatientRate(c.Request.Context(), code, modifiers)
	if err != nil {
		h.logger.Error("Failed to get outpatient rate", "error", err, "code", code, "modifiers", modifiers)
		response.SendError(c, "Failed to retrieve outpatient rate", http.StatusInternalServerError, err.Error())
		return
	}

	response.SendSuccess(c, "Outpatient rate retrieved successfully", rate)
}

// CreateOutpatientRate godoc
// @Summary      Create Outpatient Rate
// @Description  Create a new outpatient rate
// @Tags         Rates
// @Accept       json
// @Produce      json
// @Param        rate  body      ratesdomain.OutPatientRate  true  "Outpatient rate to create"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /rates/outpatient/create [post]
func (h *RateHandler) CreateOutpatientRate(c *gin.Context) {
	var rate ratesdomain.OutPatientRate

	if err := c.ShouldBindJSON(&rate); err != nil {
		h.logger.Error("Failed to bind create rate request", "error", err)
		response.SendError(c, "Invalid request body", http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("Creating outpatient rate", "code", rate.Codes, "description", rate.Description)

	if err := h.service.CreateOutPatientRate(c.Request.Context(), rate); err != nil {
		h.logger.Error("Failed to create outpatient rate", "error", err)
		response.SendError(c, "Failed to create outpatient rate", http.StatusInternalServerError, err.Error())
		return
	}

	response.SendSuccess(c, "Outpatient rate created successfully", rate, http.StatusCreated)
}
