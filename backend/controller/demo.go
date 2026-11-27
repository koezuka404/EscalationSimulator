package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/entity"
	"escalator/rule"
)

type DemoAPI struct {
	demo runDemoUsecase
}

func NewDemoAPI(demo runDemoUsecase) *DemoAPI {
	return &DemoAPI{demo: demo}
}

//デモの開始を受ける
func (a *DemoAPI) Start(c echo.Context) error {
	var body startDemoJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。作る件数、間隔、出方を入力してください"})
	}
	run, err := a.demo.Start(c.Request().Context(), accessToken(c), body.Count, body.IntervalSeconds, body.Distribution)
	if err != nil {
		return writeDemoError(c, err, "デモを始められませんでした。しばらくしてから、もう一度試してください")
	}
	return c.JSON(http.StatusAccepted, toDemoJSON(run))
}

//デモの停止を受ける
func (a *DemoAPI) Stop(c echo.Context) error {
	run, err := a.demo.Stop(c.Request().Context(), accessToken(c))
	if err != nil {
		return writeDemoError(c, err, "デモを止められませんでした。しばらくしてから、もう一度試してください")
	}
	return c.JSON(http.StatusOK, toDemoJSON(run))
}

type startDemoJSON struct {
	Count           int    `json:"count"`
	IntervalSeconds int    `json:"interval_seconds"`
	Distribution    string `json:"distribution"`
}

type demoJSON struct {
	ID              string `json:"id"`
	TotalCount      int    `json:"total_count"`
	IntervalSeconds int    `json:"interval_seconds"`
	Distribution    string `json:"distribution"`
	Status          string `json:"status"`
	GeneratedCount  int    `json:"generated_count"`
}

func toDemoJSON(run entity.DemoRun) demoJSON {
	return demoJSON{
		ID:              run.ID,
		TotalCount:      run.TotalCount,
		IntervalSeconds: run.IntervalSec,
		Distribution:    run.Distribution,
		Status:          run.Status,
		GeneratedCount:  run.GeneratedCount,
	}
}

func writeDemoError(c echo.Context, err error, fallback string) error {
	switch {
	case errors.Is(err, rule.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrInvalidDemoCount),
		errors.Is(err, rule.ErrInvalidDemoInterval),
		errors.Is(err, rule.ErrInvalidDemoDistribution):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrDemoNeedsApplicant),
		errors.Is(err, rule.ErrDemoRunning),
		errors.Is(err, rule.ErrDemoNotRunning):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: fallback})
	}
}
