package rates

// // UpdateOutPatientRate godoc
// // @Summary      Update Out Patient Rate
// // @Description  Update an existing out patient rate by its ID
// // @Tags         Rates
// // @Accept       json
// // @Produce      json
// // @Param        id    path      string                  true  "Out Patient Rate ID"
// // @Param        rate  body      CreateOutPatientRateReq  true  "Out Patient Rate to update"
// // @Success      200  {object}  OutPatientRateRes
// // @Failure      400  {object}  ErrorRes
// // @Failure      500  {object}  ErrorRes
// // @Router       /rates/outpatient/{id} [put]
// func (a *ratesApi) UpdateOutPatientRate(c *gin.Context) {
// 	id := c.Param("id")
// 	iduuid, err := uuid.Parse(id)
// 	if err != nil {
// 		a.logger.Error("invalid id parameter", "error", err)
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id parameter"})
// 		return
// 	}
// 	var req CreateOutPatientRateReq
// 	if err := c.BindJSON(&req); err != nil {
// 		a.logger.Error("failed to bind update rate request", "error", err)
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
// 		return
// 	}
// 	newRate := rates.OutPatientRate{
// 		Id:          iduuid,
// 		Code:        req.Code,
// 		Modifiers:   req.Modifiers,
// 		Description: req.Description,
// 		ApgWeight:   req.ApgWeight,
// 		OnSiteRate:  req.OnSiteRate,
// 		OffSiteRate: req.OffSiteRate,
// 		StartDate:   req.StartDate,
// 		Note:        req.Note,
// 		EndDate:     req.EndDate,
// 	}

// 	rate, err := a.rateSvc.UpdateOutPatientRate(c.Request.Context(), newRate)
// 	if err != nil {
// 		a.logger.Error("failed to update out patient rate", "error", err)
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update out patient rate"})
// 		return
// 	}
// 	c.JSON(http.StatusOK, toResponseRate(rate))
// }
