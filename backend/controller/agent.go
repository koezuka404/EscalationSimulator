package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/entity"
	"escalator/usecase"
)

type AgentAPI struct {
	status *usecase.ChangeAgentStatus
}

func NewAgentAPI(status *usecase.ChangeAgentStatus) *AgentAPI {
	return &AgentAPI{status: status}
}

//稼働の切り替えを受ける
func (a *AgentAPI) Update(c echo.Context) error {
	var body agentStatusJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。待機中か離席を選んでください"})
	}
	status, err := a.status.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), body.UserID, body.Status)
	if err != nil {
		return writeAgentStatusError(c, err)
	}
	return c.JSON(http.StatusOK, agentStatusJSON{UserID: status.UserID, Status: string(status.Status)})
}

type agentStatusJSON struct {
	UserID string `json:"user_id,omitempty"`
	Status string `json:"status"`
}

func writeAgentStatusError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrAvailabilityRole),
		errors.Is(err, entity.ErrAvailabilityOthers):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrUserNotFound),
		errors.Is(err, entity.ErrCustomerNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrCannotBecomeAvailable):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrInvalidAvailability),
		errors.Is(err, entity.ErrAvailabilityOfflineOnly):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "稼働を切り替えられませんでした。しばらくしてから、もう一度試してください"})
	}
}
