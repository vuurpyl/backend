package box

import (
	"net/http"

	v "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/labstack/echo/v4"
	"gitlab.misakey.dev/misakey/msk-sdk-go/ajwt"
	"gitlab.misakey.dev/misakey/msk-sdk-go/merror"

	"gitlab.misakey.dev/misakey/backend/api/src/modules/box/events"
)

type uploadEncryptedFileCmd struct {
	boxID string
	size  int64

	MsgEncryptedContent string `form:"msg_encrypted_content"`
}

func (cmd uploadEncryptedFileCmd) Validate() error {
	return v.ValidateStruct(&cmd,
		v.Field(&cmd.boxID, v.Required, is.UUIDv4),
		v.Field(&cmd.MsgEncryptedContent, v.Required, is.Base64),
		v.Field(&cmd.size, v.Required, v.Max(8*1024*1000).Error("the maximum file size is 8MB")),
	)
}

func (h *handler) uploadEncryptedFile(eCtx echo.Context) error {
	ctx := eCtx.Request().Context()

	acc := ajwt.GetAccesses(ctx)
	if acc == nil {
		return merror.Forbidden()
	}

	// retrieve request data
	// pick blob manually to prevent it of being erase from context during the binding and validating phase
	encFile, err := eCtx.FormFile("encrypted_file")
	if err != nil {
		return merror.BadRequest().From(merror.OriBody).
			Detail("encrypted_file", merror.DVRequired).Describe(err.Error())
	}

	// retrieve the metadata and validate it
	cmd := uploadEncryptedFileCmd{}
	if err := eCtx.Bind(&cmd); err != nil {
		return err
	}
	cmd.size = encFile.Size
	cmd.boxID = eCtx.Param("id")
	if err := cmd.Validate(); err != nil {
		return err
	}

	// retrieve the raw []byte from the file
	encData, err := encFile.Open()
	if err != nil {
		return merror.Internal().Describef("opening encrypted file: %v", err)
	}
	defer encData.Close()

	// create the new msg file that will described the upload action
	e, fileID, err := events.NewMsgFile(ctx, cmd.boxID, acc.Subject, cmd.MsgEncryptedContent)
	if err != nil {
		return merror.Transform(err).Describe("creating msg file event")
	}

	// upload the encrypted data
	if err := h.files.Upload(ctx, e.BoxID, fileID, encData); err != nil {
		return merror.Transform(err).Describe("uploading file")
	}

	// persist the event in storage - on failure, we try to remove the uploaded file
	view, err := h.createEvent(ctx, e)
	if err != nil {
		err = merror.Transform(err).Describe("inserting event in DB")
		if delErr := h.files.Delete(ctx, e.BoxID, fileID); delErr != nil {
			return merror.Transform(err).Describef("deleting file: %v", delErr)
		}
		return err
	}

	return eCtx.JSON(http.StatusCreated, view)
}