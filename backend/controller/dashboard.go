package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/entity"
	"escalator/usecase"
)

type DashboardAPI struct {
	show *usecase.ShowDashboard
}

func NewDashboardAPI(show *usecase.ShowDashboard) *DashboardAPI {
	return &DashboardAPI{show: show}
}

//管理者に現場の数字を返す
func (a *DashboardAPI) Show(c echo.Context) error {
	numbers, err := a.show.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"))
	if err != nil {
		return writeDashboardError(c, err)
	}
	return c.JSON(http.StatusOK, toDashboardJSON(numbers))
}

type dashboardJSON struct {
	WaitingCount                int `json:"waiting_count"`
	InProgressCount             int `json:"in_progress_count"`
	AverageHandleMinutes        int `json:"average_handle_minutes"`
	AverageFirstResponseMinutes int `json:"average_first_response_minutes"`
	OverdueCount                int `json:"overdue_count"`
	Overdue24hCount             int `json:"overdue_24h_count"`
	AvailableCount              int `json:"available_count"`
	BusyCount                   int `json:"busy_count"`
	OfflineCount                int `json:"offline_count"`
}

func toDashboardJSON(numbers usecase.Dashboard) dashboardJSON {
	return dashboardJSON{
		WaitingCount:                numbers.WaitingCount,
		InProgressCount:             numbers.InProgressCount,
		AverageHandleMinutes:        numbers.AverageHandleMinutes,
		AverageFirstResponseMinutes: numbers.AverageFirstResponseMinutes,
		OverdueCount:                numbers.OverdueCount,
		Overdue24hCount:             numbers.Overdue24hCount,
		AvailableCount:              numbers.AvailableCount,
		BusyCount:                   numbers.BusyCount,
		OfflineCount:                numbers.OfflineCount,
	}
}

func writeDashboardError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "現場の数字をまとめられませんでした。しばらくしてから、もう一度試してください"})
	}
}
