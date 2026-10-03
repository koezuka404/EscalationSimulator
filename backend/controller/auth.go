package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"escalator/domain"
	"escalator/middleware"
	"escalator/usecase"
)

// AuthAPI は会員登録とログインを HTTP で受ける入口。
type AuthAPI struct {
	signUp            *usecase.SignUp
	logIn             *usecase.LogIn
	logOut            *usecase.LogOut
	refreshCookieName string
	csrfCookieName    string
	cookieSecure      bool
	refreshTTL        int
}

func NewAuthAPI(signUp *usecase.SignUp, logIn *usecase.LogIn, logOut *usecase.LogOut, refreshCookieName, csrfCookieName string, cookieSecure bool, refreshTTLSeconds int) *AuthAPI {
	return &AuthAPI{
		signUp:            signUp,
		logIn:             logIn,
		logOut:            logOut,
		refreshCookieName: refreshCookieName,
		csrfCookieName:    csrfCookieName,
		cookieSecure:      cookieSecure,
		refreshTTL:        refreshTTLSeconds,
	}
}

func (a *AuthAPI) Register(c echo.Context) error {
	var body registerJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。名前、メールアドレス、パスワードを入力してください"})
	}
	user, err := a.signUp.Execute(c.Request().Context(), body.Name, body.Email, body.Password)
	if err != nil {
		return writeSignUpError(c, err)
	}
	return c.JSON(http.StatusCreated, toUserJSON(user))
}

func (a *AuthAPI) Login(c echo.Context) error {
	var body loginJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。メールアドレスとパスワードを入力してください"})
	}
	result, err := a.logIn.Execute(c.Request().Context(), body.Email, body.Password)
	if err != nil {
		return writeLoginError(c, err)
	}
	c.SetCookie(sessionCookie(a.refreshCookieName, result.RefreshToken, a.refreshTTL, true, a.cookieSecure))
	c.SetCookie(sessionCookie(a.csrfCookieName, result.CSRFToken, a.refreshTTL, false, a.cookieSecure))
	return c.JSON(http.StatusOK, loginResponseJSON{
		AccessToken: result.AccessToken,
		ExpiresIn:   int(time.Until(result.ExpiresAt).Seconds()),
		User:        toUserJSON(result.User),
	})
}

func (a *AuthAPI) Logout(c echo.Context) error {
	if err := middleware.Allow(c.Request(), a.csrfCookieName); err != nil {
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	}
	refreshToken := ""
	if cookie, err := c.Cookie(a.refreshCookieName); err == nil {
		refreshToken = cookie.Value
	}
	if err := a.logOut.Execute(c.Request().Context(), refreshToken); err != nil {
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "ログアウトできませんでした。しばらくしてから、もう一度試してください"})
	}
	c.SetCookie(sessionCookie(a.refreshCookieName, "", -1, true, a.cookieSecure))
	c.SetCookie(sessionCookie(a.csrfCookieName, "", -1, false, a.cookieSecure))
	return c.JSON(http.StatusOK, messageJSON{Message: "ログアウトしました"})
}

type registerJSON struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginJSON struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponseJSON struct {
	AccessToken string   `json:"access_token"`
	ExpiresIn   int      `json:"expires_in"`
	User        userJSON `json:"user"`
}

type userJSON struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	CustomerID string `json:"customer_id"`
	Status     string `json:"status"`
}

func toUserJSON(user domain.User) userJSON {
	return userJSON{
		ID:         user.ID,
		Name:       user.Name,
		Email:      user.Email,
		Role:       string(user.Role),
		CustomerID: user.CustomerID,
		Status:     string(user.Status),
	}
}

func writeSignUpError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidUserName),
		errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPassword):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	case errors.Is(err, domain.ErrEmailTaken):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "登録できませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeLoginError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrLoginFailed):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, domain.ErrLoginLocked):
		return c.JSON(http.StatusTooManyRequests, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "ログインできませんでした。しばらくしてから、もう一度試してください"})
	}
}

func sessionCookie(name, value string, maxAge int, httpOnly, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}
