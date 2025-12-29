package dashboard

import (
	"log/slog"

	"github.com/aci/backend/internal/core/services/dashboard"
)

type DashboardHandler struct {
	service dashboard.Service
	logger  *slog.Logger
}

func NewDashboardHandler(service dashboard.Service, logger *slog.Logger) *DashboardHandler {
	return &DashboardHandler{service: service, logger: logger}
}
