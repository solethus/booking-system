package user

import (
	"context"
	"encore.dev/beta/auth"
)

type Data struct {
	Email string `json:"email"`
}

type AuthParams struct {
	Authorization string `header:"authorization"`
}

//encore:authhandler
func AuthHandler(ctx context.Context, p *AuthParams) (auth.UID, *Data, error) {
	if p.Authorization != "" {
		return "test", &Data{}, nil
	}
	return "", nil, nil
}
