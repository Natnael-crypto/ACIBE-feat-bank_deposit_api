package rates

import (
	"github.com/gin-gonic/gin"
)

// @Summary      Delete Out Patient Rate
// @Description  Deletes an out patient rate by its ID
// @Tags         Rates
// @Param        id   path      string  true  "Out Patient Rate ID"
// @Success      204  "No Content"
// @Failure      400  {object}  ErrorRes
// @Failure      500  {object}  ErrorRes
// @Router       /rates/outpatient/{id} [delete]
func (a *ratesApi) DeleteOutPatientRate(c *gin.Context) {
	// idParam := c.Param("id")
	// idParamUuid, err := uuid.Parse(idParam)
	// if err != nil {
	// 	a.logger.Error("invalid id parameter", "error", err)
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id parameter"})
	// 	return
	// }
	// err = a.rateSvc.DeleteOutPatientRate(c.Request.Context(), idParamUuid)
	// if err != nil {
	// 	a.logger.Error("failed to delete out patient rate", "error", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete out patient rate"})
	// 	return
	// }
	// c.Status(http.StatusNoContent)
}

// DeleteInPatientRate godoc
// @Summary      Delete In Patient Rate
// @Description  Deletes an in patient rate by its ID
// @Tags         Rates
// @Param        id   path      string  true  "In Patient Rate ID"
// @Success      204  "No Content"
// @Failure      400  {object}  ErrorRes
// @Failure      500  {object}  ErrorRes
// @Router       /rates/inpatient/{id} [delete]
func (a *ratesApi) DeleteInPatientRate(c *gin.Context) {
	// idParam := c.Param("id")
	// idParamUuid, err := uuid.Parse(idParam)
	// if err != nil {
	// 	a.logger.Error("invalid id parameter", "error", err)
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id parameter"})
	// 	return
	// }
	// err = a.rateSvc.DeleteInPatientRate(c.Request.Context(), idParamUuid)
	// if err != nil {
	// 	a.logger.Error("failed to delete in patient rate", "error", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete in patient rate"})
	// 	return
	// }
	// c.Status(http.StatusNoContent)
}
