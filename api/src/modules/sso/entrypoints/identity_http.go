package entrypoints

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"gitlab.misakey.dev/misakey/backend/api/src/modules/sso/application"
	"gitlab.misakey.dev/misakey/msk-sdk-go/merror"
)

type IdentityHTTP struct {
	service application.SSOService
}

func NewIdentityHTTP(service application.SSOService) IdentityHTTP {
	return IdentityHTTP{service: service}
}

// Handles PUT /identities/authable - retrieve authable identity information
func (entrypoint IdentityHTTP) RequireAuthableIdentity(ctx echo.Context) error {
	cmd := application.IdentityAuthableCmd{}
	if err := ctx.Bind(&cmd); err != nil {
		return merror.BadRequest().From(merror.OriBody).Describe(err.Error())
	}

	if err := cmd.Validate(); err != nil {
		return merror.Transform(err).From(merror.OriBody)
	}

	identity, err := entrypoint.service.RequireIdentityAuthable(ctx.Request().Context(), cmd)
	if err != nil {
		return merror.Transform(err).Describe("could not require authable identity").From(merror.OriBody)
	}
	return ctx.JSON(http.StatusOK, identity)
}