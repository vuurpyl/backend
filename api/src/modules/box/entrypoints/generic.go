package entrypoints

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"gitlab.misakey.dev/misakey/msk-sdk-go/ajwt"
	"gitlab.misakey.dev/misakey/msk-sdk-go/merror"
)

type Request interface {
	BindAndValidate(echo.Context) error
}

func ResponseNoContent(eCtx echo.Context, _ interface{}) error {
	return eCtx.NoContent(http.StatusNoContent)
}

func ResponseOK(eCtx echo.Context, data interface{}) error {
	return eCtx.JSON(http.StatusOK, data)
}

func ResponseCreated(eCtx echo.Context, data interface{}) error {
	return eCtx.JSON(http.StatusCreated, data)
}

func ResponseBlob(eCtx echo.Context, data interface{}) error {
	return eCtx.Blob(http.StatusOK, "application/octet-stream", data.([]byte))
}

func NewHTTPEntrypoint(
	initReq func() Request,
	checkAccesses bool,
	appFunc func(context.Context, Request) (interface{}, error),
	responseFunc func(echo.Context, interface{}) error,
	afterOpts ...func(echo.Context, interface{}) error,
) func(eCtx echo.Context) error {
	return func(
		eCtx echo.Context,
	) error {
		req := initReq()
		ctx := eCtx.Request().Context()

		if checkAccesses {
			if acc := ajwt.GetAccesses(ctx); acc == nil {
				return merror.Forbidden()
			}
		}

		// bind - validate the request
		if err := req.BindAndValidate(eCtx); err != nil {
			return err
		}

		data, err := appFunc(ctx, req)
		if err != nil {
			return err
		}

		for _, opt := range afterOpts {
			if err := opt(eCtx, data); err != nil {
				return err
			}
		}

		return responseFunc(eCtx, data)
	}
}