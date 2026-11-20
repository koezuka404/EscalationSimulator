package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/rule"
)

type UserAPI struct {
	link    linkApplicantUsecase
	create  createAgentUsecase
	list    listUsersUsecase
	suspend suspendUserUsecase
	resume  resumeUserUsecase
	remove  deleteUserUsecase
	role    changeRoleUsecase
}

func NewUserAPI(link linkApplicantUsecase, create createAgentUsecase, list listUsersUsecase, suspend suspendUserUsecase, resume resumeUserUsecase, remove deleteUserUsecase, role changeRoleUsecase) *UserAPI {
	return &UserAPI{link: link, create: create, list: list, suspend: suspend, resume: resume, remove: remove, role: role}
}

// 利用者の一覧を返す
func (a *UserAPI) List(c echo.Context) error {
	users, err := a.list.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"))
	if err != nil {
		return writeListUsersError(c, err)
	}
	body := make([]userJSON, 0, len(users))
	for _, user := range users {
		body = append(body, toUserJSON(user))
	}
	return c.JSON(http.StatusOK, body)
}

// 担当者の登録を受ける
func (a *UserAPI) CreateAgent(c echo.Context) error {
	var body registerJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。名前、メールアドレス、パスワードを入力してください"})
	}
	user, err := a.create.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), body.Name, body.Email, body.Password)
	if err != nil {
		return writeCreateAgentError(c, err)
	}
	return c.JSON(http.StatusCreated, toUserJSON(user))
}

// 申請者を顧客に結びつける
func (a *UserAPI) LinkCustomer(c echo.Context) error {
	var body linkCustomerJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。結びつける顧客を選んでください"})
	}
	user, err := a.link.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), c.Param("id"), body.CustomerID)
	if err != nil {
		return writeLinkError(c, err)
	}
	return c.JSON(http.StatusOK, toUserJSON(user))
}

// 利用者の停止を受ける
func (a *UserAPI) Suspend(c echo.Context) error {
	user, err := a.suspend.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), c.Param("id"))
	if err != nil {
		return writeSuspendError(c, err)
	}
	return c.JSON(http.StatusOK, toUserJSON(user))
}

// 利用者を利用中に戻す
func (a *UserAPI) Resume(c echo.Context) error {
	user, err := a.resume.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), c.Param("id"))
	if err != nil {
		return writeResumeError(c, err)
	}
	return c.JSON(http.StatusOK, toUserJSON(user))
}

// 利用者の削除を受ける
func (a *UserAPI) Delete(c echo.Context) error {
	user, err := a.remove.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), c.Param("id"))
	if err != nil {
		return writeDeleteError(c, err)
	}
	return c.JSON(http.StatusOK, toUserJSON(user))
}

// 利用者の役割の変更を受ける
func (a *UserAPI) ChangeRole(c echo.Context) error {
	var body changeRoleJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。役割を選んでください"})
	}
	user, err := a.role.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), c.Param("id"), body.Role, body.CustomerID)
	if err != nil {
		return writeRoleError(c, err)
	}
	return c.JSON(http.StatusOK, toUserJSON(user))
}

type changeRoleJSON struct {
	Role       string `json:"role"`
	CustomerID string `json:"customer_id"`
}

type linkCustomerJSON struct {
	CustomerID string `json:"customer_id"`
}

func writeCreateAgentError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, rule.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrInvalidUserName),
		errors.Is(err, rule.ErrInvalidEmail),
		errors.Is(err, rule.ErrInvalidPassword):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrEmailTaken):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "担当者を登録できませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeListUsersError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, rule.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "利用者の一覧をまとめられませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeLinkError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, rule.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrForbidden), errors.Is(err, rule.ErrNotApplicant):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrUserNotFound), errors.Is(err, rule.ErrCustomerNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrCustomerLink):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "申請者を顧客に結びつけられませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeSuspendError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, rule.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrForbidden), errors.Is(err, rule.ErrSuspendSelf):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrUserNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrAlreadySuspended):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "利用者を停止できませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeResumeError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, rule.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrUserNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrAlreadyActive):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "利用者を利用中に戻せませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeDeleteError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, rule.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrUserNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrDeleteOnlySuspended):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "利用者を削除できませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeRoleError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, rule.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrForbidden), errors.Is(err, rule.ErrChangeOwnRole):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrUserNotFound), errors.Is(err, rule.ErrCustomerNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrInvalidRole), errors.Is(err, rule.ErrApplicantNeedsCustomer):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	case errors.Is(err, rule.ErrRoleSame):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "役割を変えられませんでした。しばらくしてから、もう一度試してください"})
	}
}
