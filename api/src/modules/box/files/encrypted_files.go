package files

import (
	"context"
	"database/sql"
	"io"

	"github.com/volatiletech/sqlboiler/boil"
	"gitlab.misakey.dev/misakey/msk-sdk-go/merror"

	"gitlab.misakey.dev/misakey/backend/api/src/modules/box/events"
	"gitlab.misakey.dev/misakey/backend/api/src/modules/box/repositories/sqlboiler"
)

type EncryptedFile struct {
	ID   string
	Size int64
}

type FileStorageRepo interface {
	Upload(context.Context, string, io.Reader) error
	Download(context.Context, string) ([]byte, error)
	Delete(context.Context, string) error
}

func Create(ctx context.Context, exec boil.ContextExecutor, encryptedFile EncryptedFile) error {
	toStore := sqlboiler.EncryptedFile{
		ID:   encryptedFile.ID,
		Size: encryptedFile.Size,
	}
	return toStore.Insert(ctx, exec, boil.Infer())
}

func Get(ctx context.Context, exec boil.ContextExecutor, fileID string) (*EncryptedFile, error) {
	dbEncryptedFile, err := sqlboiler.EncryptedFiles(sqlboiler.EncryptedFileWhere.ID.EQ(fileID)).One(ctx, exec)
	if err == sql.ErrNoRows {
		return nil, merror.NotFound().Detail("id", merror.DVNotFound)
	}
	if err != nil {
		return nil, err
	}

	encryptedFile := EncryptedFile{
		ID:   dbEncryptedFile.ID,
		Size: dbEncryptedFile.Size,
	}

	return &encryptedFile, nil
}

func Upload(ctx context.Context, repo FileStorageRepo, fileID string, encData io.Reader) error {
	return repo.Upload(ctx, fileID, encData)
}

func Download(ctx context.Context, repo FileStorageRepo, fileID string) ([]byte, error) {
	return repo.Download(ctx, fileID)
}

func Delete(ctx context.Context, exec boil.ContextExecutor, repo FileStorageRepo, fileID string) error {
	// delete the stored file
	if err := repo.Delete(ctx, fileID); err != nil {
		return err
	}

	// delete file entity (ignoring the no row affected error)
	if _, err := sqlboiler.EncryptedFiles(sqlboiler.EncryptedFileWhere.ID.EQ(fileID)).DeleteAll(ctx, exec); err != nil {
		return err
	}

	return nil
}

func IsOrphan(ctx context.Context, exec boil.ContextExecutor, fileID string) (bool, error) {
	// check that there is no saved file referring this file
	savedFiles, err := ListSavedFilesByFileID(ctx, exec, fileID)
	if err != nil {
		return false, err
	}
	if len(savedFiles) != 0 {
		return false, nil
	}

	// check that there is no box event referring this file
	boxEvents, err := events.FindByEncryptedFileID(ctx, exec, fileID)
	if err != nil && !merror.HasCode(err, merror.NotFoundCode) {
		return false, err
	}
	if len(boxEvents) != 0 {
		return false, nil
	}

	// the file is orphan
	return true, nil
}
