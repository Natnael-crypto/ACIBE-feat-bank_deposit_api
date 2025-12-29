package rates

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

type ratesApi struct {
	router *gin.Engine
	logger *slog.Logger
	// rateSvc *rateservice.Service
}

func (a *ratesApi) RegisterRoutes() {
	api := a.router.Group("/api")
	api.GET("/rates/outpatient/:id", a.GetOutPatientRateById)
	api.POST("/rates/outpatient", a.CreateOutPatientRate)
	api.GET("/rates/outpatient", a.GetOutPatientRates)
	// api.PUT("/rates/outpatient/:id", a.UpdateOutPatientRate)
	api.DELETE("/rates/outpatient/:id", a.DeleteOutPatientRate)
	
	// In-patient rate routes
	api.GET("/rates/inpatient/:id", a.GetInPatientRateById)
	api.POST("/rates/inpatient", a.CreateInPatientRate)
	api.GET("/rates/inpatient", a.GetInPatientRates)
	// api.PUT("/rates/inpatient/:id", a.UpdateInPatientRate)
	api.DELETE("/rates/inpatient/:id", a.DeleteInPatientRate)
}
func NewAuthenticationHandlers(router *gin.Engine, logger *slog.Logger) {
	ratesapi := &ratesApi{
		router: router,
		// rateSvc: rateSvc,
		logger: logger,
	}
	ratesapi.RegisterRoutes()

}

type ErrorRes struct {
	Error string `json:"error"`
}
