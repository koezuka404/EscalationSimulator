package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/rule"
)

type AgentAPI struct {
	status changeAgentStatusUsecase
}

func NewAgentAPI(status changeAgentStatusUsecase) *AgentAPI {
	return &AgentAPI{status: status}
}

//稼働の切り替えを受ける
func (a *AgentAPI) Update(c echo.Context) error {
	var body agentStatusJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。待機中か離席を選んでください"})
	}
	status, err := a.status.Execute(c.Request().Context(), accessToken(c), body.UserID, body.Status)
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
	case errors.Is(err, rule.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrAvailabilityRole),
		errors.Is(err, rule.ErrAvailabilityOthers):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrUserNotFound),
		errors.Is(err, rule.ErrCustomerNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrCannotBecomeAvailable):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrInvalidAvailability),
		errors.Is(err, rule.ErrAvailabilityOfflineOnly):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "稼働を切り替えられませんでした。しばらくしてから、もう一度試してください"})
	}
}
