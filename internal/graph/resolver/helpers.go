package resolver

import (
	"github.com/TechBuilder-360/Rental-API/internal/auth"
	"github.com/TechBuilder-360/Rental-API/internal/graph/model"
)

func toAuthPayload(res *auth.Result) *model.AuthPayload {
	return &model.AuthPayload{
		AccessToken:           res.AccessToken,
		AccessTokenExpiresAt:  res.AccessTokenExpiresAt,
		RefreshToken:          res.RefreshToken,
		RefreshTokenExpiresAt: res.RefreshTokenExpiresAt,
		TokenType:             "Bearer",
		User:                  res.User,
	}
}
