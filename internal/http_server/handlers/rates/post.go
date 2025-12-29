package rates

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateOutPatientRateReq struct {
	Code        string          `json:"code" binding:"required"`
	Modifiers   []string        `json:"modifiers"`
	Description string          `json:"description" binding:"required"`
	ApgWeight   decimal.Decimal `json:"apgWeight" `
	OnSiteRate  decimal.Decimal `json:"onSiteRate" binding:"required"`
	OffSiteRate decimal.Decimal `json:"offSiteRate" binding:"required"`
	StartDate   time.Time       `json:"startDate" binding:"required"`
	EndDate     *time.Time      `json:"endDate"`
	Note        string          `json:"note"`
}
type OutPatientRateRes struct {
	Id          uuid.UUID       `json:"id"`
	Code        string          `json:"code"`
	Modifiers   []string        `json:"modifiers"`
	Description string          `json:"description"`
	ApgWeight   decimal.Decimal `json:"apgWeight"`
	OnSiteRate  decimal.Decimal `json:"onSiteRate"`
	OffSiteRate decimal.Decimal `json:"offSiteRate"`
	StartDate   time.Time       `json:"startDate"`
	EndDate     *time.Time      `json:"endDate,omitempty"`
	Note        string          `json:"note"`
}

// CreateRates godoc
// @Summary      Create Out Patient Rate
// @Description  Create a new out patient rate
// @Tags         Rates
// @Accept       json
// @Produce      json
// @Param        rate  body      CreateOutPatientRateReq  true  "Out Patient Rate to create"
// @Success      201  {object}  OutPatientRateRes
// @Failure      400  {object}  ErrorRes
// @Failure      500  {object}  ErrorRes
// @Router       /rates/outpatient [post]
func (a *ratesApi) CreateOutPatientRate(c *gin.Context) {
	// var req CreateOutPatientRateReq
	// if err := c.BindJSON(&req); err != nil {
	// 	a.logger.Error("failed to bind create rate request", "error", err)
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	// 	return
	// }
	// newRate := rates.OutPatientRate{
	// 	Code:        req.Code,
	// 	Modifiers:   req.Modifiers,
	// 	Description: req.Description,
	// 	ApgWeight:   req.ApgWeight,
	// 	OnSiteRate:  req.OnSiteRate,
	// 	OffSiteRate: req.OffSiteRate,
	// 	StartDate:   req.StartDate,
	// 	Note:        req.Note,
	// 	EndDate:     req.EndDate,
	// }

	// res, err := a.rateSvc.CreateOutPatientRate(c.Request.Context(), newRate)
	// if err != nil {
	// 	a.logger.Error("failed to create rate", "error", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rate"})
	// 	return
	// }

	// c.JSON(http.StatusCreated, toResponseRate(res))

}

// func toResponseRate(rate rates.OutPatientRate) OutPatientRateRes {

// 	return OutPatientRateRes{
// 		Id:          rate.Id,
// 		Code:        rate.Code,
// 		Modifiers:   rate.Modifiers,
// 		Description: rate.Description,
// 		ApgWeight:   rate.ApgWeight,
// 		OnSiteRate:  rate.OnSiteRate,
// 		OffSiteRate: rate.OffSiteRate,
// 		StartDate:   rate.StartDate,
// 		EndDate:     rate.EndDate,
// 		Note:        rate.Note,
// 	}
// }

type CreateInPatientRateReq struct {
	Code        string          `json:"code" binding:"required"`
	Modifiers   []string        `json:"modifiers"`
	Description string          `json:"description" binding:"required"`
	ApgWeight   decimal.Decimal `json:"apgWeight" `
	OnSiteRate  decimal.Decimal `json:"onSiteRate" binding:"required"`
	OffSiteRate decimal.Decimal `json:"offSiteRate" binding:"required"`
	StartDate   time.Time       `json:"startDate" binding:"required"`
	EndDate     *time.Time      `json:"endDate"`
	Note        string          `json:"note"`
}

type InPatientRateRes struct {
	Id          uuid.UUID       `json:"id"`
	Code        string          `json:"code"`
	Modifiers   []string        `json:"modifiers"`
	Description string          `json:"description"`
	ApgWeight   decimal.Decimal `json:"apgWeight"`
	OnSiteRate  decimal.Decimal `json:"onSiteRate"`
	OffSiteRate decimal.Decimal `json:"offSiteRate"`
	StartDate   time.Time       `json:"startDate"`
	EndDate     *time.Time      `json:"endDate,omitempty"`
	Note        string          `json:"note"`
}

// CreateInPatientRate godoc
// @Summary      Create In Patient Rate
// @Description  Create a new in patient rate
// @Tags         Rates
// @Accept       json
// @Produce      json
// @Param        rate  body      CreateInPatientRateReq  true  "In Patient Rate to create"
// @Success      201  {object}  InPatientRateRes
// @Failure      400  {object}  ErrorRes
// @Failure      500  {object}  ErrorRes
// @Router       /rates/inpatient [post]
func (a *ratesApi) CreateInPatientRate(c *gin.Context) {
	// var req CreateInPatientRateReq
	// if err := c.BindJSON(&req); err != nil {
	// 	a.logger.Error("failed to bind create rate request", "error", err)
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	// 	return
	// }
	// newRate := rates.InPatientRate{
	// 	Code:        req.Code,
	// 	Modifiers:   req.Modifiers,
	// 	Description: req.Description,
	// 	ApgWeight:   req.ApgWeight,
	// 	OnSiteRate:  req.OnSiteRate,
	// 	OffSiteRate: req.OffSiteRate,
	// 	StartDate:   req.StartDate,
	// 	Note:        req.Note,
	// 	EndDate:     req.EndDate,
	// }

	// res, err := a.rateSvc.CreateInPatientRate(c.Request.Context(), newRate)
	// if err != nil {
	// 	a.logger.Error("failed to create rate", "error", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rate"})
	// 	return
	// }

	// c.JSON(http.StatusCreated, toResponseInPatientRate(res))
}

// func toResponseInPatientRate(rate rates.InPatientRate) InPatientRateRes {
// 	return InPatientRateRes{
// 		Id:          rate.Id,
// 		Code:        rate.Code,
// 		Modifiers:   rate.Modifiers,
// 		Description: rate.Description,
// 		ApgWeight:   rate.ApgWeight,
// 		OnSiteRate:  rate.OnSiteRate,
// 		OffSiteRate: rate.OffSiteRate,
// 		StartDate:   rate.StartDate,
// 		EndDate:     rate.EndDate,
// 		Note:        rate.Note,
// 	}
// }
