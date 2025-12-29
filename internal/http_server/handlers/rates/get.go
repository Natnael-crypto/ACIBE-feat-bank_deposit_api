package rates

import (
	domain "github.com/aci/backend/internal/core/domain/common"

	"github.com/gin-gonic/gin"
)

type GetOutPatientRatesRes struct {
	OutPatientRates []OutPatientRateRes `json:"outPatientRates"`
	Metadata        domain.Metadata     `json:"metadata"`
}

type GetOutPatientRatesQuery struct {
	Code        string   `form:"code"`
	Modifiers   []string `form:"modifiers"`
	Description string   `form:"description"`
	StartDate   string   `form:"startDate"`
	EndDate     string   `form:"endDate"`
	Page        int      `form:"page,default=1"`
	PageSize    int      `form:"pageSize,default=10"`
}

// GetOutPatientRates godoc
// @Summary      Get Out Patient Rates
// @Description  Retrieve a list of out patient rates with optional filtering and pagination
// @Tags         Rates
// @Accept       json
// @Produce      json
// @Param        code         query     string  false  "Filter by code"
// @Param        modifiers    query     []string false "Filter by modifiers"
// @Param        description  query     string  false  "Filter by description"
// @Param        startDate    query     string  false  "Filter by start date (YYYY-MM-DD)"
// @Param        endDate      query     string  false  "Filter by end date (YYYY-MM-DD)"
// @Param        page         query     int     false  "Page number for pagination" default(1)
// @Param        pageSize     query     int     false  "Number of items per page" default(10)
// @Success      200  {object}  GetOutPatientRatesRes
// @Failure      400  {object}  ErrorRes
// @Failure      500  {object}  ErrorRes
// @Router       /rates/outpatient [get]
func (a *ratesApi) GetOutPatientRates(c *gin.Context) {

	// var query GetOutPatientRatesQuery
	// if err := c.BindQuery(&query); err != nil {
	// 	a.logger.Error("failed to bind get rates query", "error", err)
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
	// 	return
	// }
	// filter := rateservice.GetRatesFilter{
	// 	Code:        query.Code,
	// 	Modifiers:   query.Modifiers,
	// 	Description: query.Description,
	// 	Page:        query.Page,
	// 	PageSize:    query.PageSize,
	// }
	// rates, metadata, err := a.rateSvc.GetOutPatientRates(c.Request.Context(), filter)
	// if err != nil {
	// 	a.logger.Error("failed to get out patient rates", "error", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get out patient rates"})
	// 	return
	// }
	// var Resrates []OutPatientRateRes
	// for _, rate := range rates {
	// 	Resrates = append(Resrates, toResponseRate(rate))
	// }
	// res := GetOutPatientRatesRes{
	// 	OutPatientRates: Resrates,
	// 	Metadata:        metadata,
	// }
	// c.JSON(http.StatusOK, res)

}

// GetOutPatientRateById godoc
// @Summary      Get Out Patient Rate by ID
// @Description  Retrieve a specific out patient rate by its ID
// @Tags         Rates
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Out Patient Rate ID"
// @Success      200  {object}  OutPatientRateRes
// @Failure      400  {object}  ErrorRes
// @Failure      500  {object}  ErrorRes
// @Router       /rates/outpatient/{id} [get]
func (a *ratesApi) GetOutPatientRateById(c *gin.Context) {
	// id := c.Param("id")
	// iduuid, err := uuid.Parse(id)
	// if err != nil {
	// 	a.logger.Error("invalid id parameter", "error", err)
	// 	c.JSON(http.StatusBadRequest, ErrorRes{
	// 		Error: "invalid id parameter",
	// 	})
	// 	return
	// }
	// rate, err := a.rateSvc.GetOutPatientRateById(c.Request.Context(), iduuid)
	// if err != nil {
	// 	a.logger.Error("failed to get out patient rate by id", "error", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get out patient rate"})
	// 	return
	// }
	// c.JSON(http.StatusOK, toResponseRate(rate))
}

type GetInPatientRatesRes struct {
	InPatientRates []InPatientRateRes `json:"inPatientRates"`
	Metadata       domain.Metadata    `json:"metadata"`
}

type GetInPatientRatesQuery struct {
	Code        string   `form:"code"`
	Modifiers   []string `form:"modifiers"`
	Description string   `form:"description"`
	StartDate   string   `form:"startDate"`
	EndDate     string   `form:"endDate"`
	Page        int      `form:"page,default=1"`
	PageSize    int      `form:"pageSize,default=10"`
}

// GetInPatientRates godoc
// @Summary      Get In Patient Rates
// @Description  Retrieve a list of in patient rates with optional filtering and pagination
// @Tags         Rates
// @Accept       json
// @Produce      json
// @Param        code         query     string  false  "Filter by code"
// @Param        modifiers    query     []string false "Filter by modifiers"
// @Param        description  query     string  false  "Filter by description"
// @Param        startDate    query     string  false  "Filter by start date (YYYY-MM-DD)"
// @Param        endDate      query     string  false  "Filter by end date (YYYY-MM-DD)"
// @Param        page         query     int     false  "Page number for pagination" default(1)
// @Param        pageSize     query     int     false  "Number of items per page" default(10)
// @Success      200  {object}  GetInPatientRatesRes
// @Failure      400  {object}  ErrorRes
// @Failure      500  {object}  ErrorRes
// @Router       /rates/inpatient [get]
func (a *ratesApi) GetInPatientRates(c *gin.Context) {
	// var query GetInPatientRatesQuery
	// if err := c.BindQuery(&query); err != nil {
	// 	a.logger.Error("failed to bind get rates query", "error", err)
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
	// 	return
	// }
	// filter := rateservice.GetRatesFilter{
	// 	Code:        query.Code,
	// 	Modifiers:   query.Modifiers,
	// 	Description: query.Description,
	// 	Page:        query.Page,
	// 	PageSize:    query.PageSize,
	// }
	// rates, metadata, err := a.rateSvc.GetInPatientRates(c.Request.Context(), filter)
	// if err != nil {
	// 	a.logger.Error("failed to get in patient rates", "error", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get in patient rates"})
	// 	return
	// }
	// var Resrates []InPatientRateRes
	// for _, rate := range rates {
	// 	Resrates = append(Resrates, toResponseInPatientRate(rate))
	// }
	// res := GetInPatientRatesRes{
	// 	InPatientRates: Resrates,
	// 	Metadata:       metadata,
	// }
	// c.JSON(http.StatusOK, res)
}

// GetInPatientRateById godoc
// @Summary      Get In Patient Rate by ID
// @Description  Retrieve a specific in patient rate by its ID
// @Tags         Rates
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "In Patient Rate ID"
// @Success      200  {object}  InPatientRateRes
// @Failure      400  {object}  ErrorRes
// @Failure      500  {object}  ErrorRes
// @Router       /rates/inpatient/{id} [get]
func (a *ratesApi) GetInPatientRateById(c *gin.Context) {
	// id := c.Param("id")
	// iduuid, err := uuid.Parse(id)
	// if err != nil {
	// 	a.logger.Error("invalid id parameter", "error", err)
	// 	c.JSON(http.StatusBadRequest, ErrorRes{
	// 		Error: "invalid id parameter",
	// 	})
	// 	return
	// }
	// rate, err := a.rateSvc.GetInPatientRateById(c.Request.Context(), iduuid)
	// if err != nil {
	// 	a.logger.Error("failed to get in patient rate by id", "error", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get in patient rate"})
	// 	return
	// }
	// c.JSON(http.StatusOK, toResponseInPatientRate(rate))
}
