package middleware

import (
	"context"
	"errors"

	"escalator/rule"
)

type RefreshLookup interface {
	FindUserIDByHash(ctx context.Context, tokenHash string) (string, error)
}

//再発行の印から利用者番号を返す。無効な印は空にする
func UserIDFromRefresh(sessions RefreshLookup) func(context.Context, string) (string, error) {
	return func(ctx context.Context, tokenHash string) (string, error) {
		userID, err := sessions.FindUserIDByHash(ctx, tokenHash)
		if errors.Is(err, rule.ErrInvalidRefresh) {
			return "", nil
		}
		return userID, err
	}
}
