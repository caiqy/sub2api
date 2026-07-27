package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type backupImageStorageTestEncryptor struct{}

func (backupImageStorageTestEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (backupImageStorageTestEncryptor) Decrypt(ciphertext string) (string, error) {
	plaintext, ok := strings.CutPrefix(ciphertext, "enc:")
	if !ok {
		return "", errors.New("not encrypted")
	}
	return plaintext, nil
}

type backupImageStorageTestStore struct{}

func (backupImageStorageTestStore) Save(context.Context, string, string, []byte) (string, error) {
	return "", nil
}

func TestBackupHandlerUpdateS3ConfigRefreshesReusedImageStorage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newTestSettingRepo()
	encryptor := backupImageStorageTestEncryptor{}
	backup := service.NewBackupService(repo, &config.Config{
		Totp: config.TotpConfig{EncryptionKeyConfigured: true},
	}, encryptor, nil, nil)
	var built []config.ImageStorageConfig
	imageStorage := service.NewImageStorageSettingService(repo, encryptor, backup,
		func(_ context.Context, cfg *config.ImageStorageConfig) (service.ImageStorage, error) {
			built = append(built, *cfg)
			return backupImageStorageTestStore{}, nil
		}, config.ImageStorageConfig{})

	_, err := backup.UpdateS3Config(context.Background(), service.BackupS3Config{
		Endpoint: "https://old.example.test", Bucket: "backup", AccessKeyID: "old-ak", SecretAccessKey: "old-sk",
	})
	require.NoError(t, err)
	_, err = imageStorage.Update(context.Background(), service.ImageStorageSettings{Enabled: true, ReuseBackupS3: true})
	require.NoError(t, err)
	_, enabled := imageStorage.Resolver()()
	require.True(t, enabled)
	require.Len(t, built, 1)
	require.Equal(t, "https://old.example.test", built[0].Endpoint)
	require.Equal(t, "old-ak", built[0].AccessKeyID)
	require.Equal(t, "old-sk", built[0].SecretAccessKey)

	handler := NewBackupHandler(backup, nil, imageStorage)
	router := gin.New()
	router.PUT("/backup/s3", handler.UpdateS3Config)
	req := httptest.NewRequest(http.MethodPut, "/backup/s3", strings.NewReader(`{"endpoint":"https://new.example.test","bucket":"backup","access_key_id":"new-ak","secret_access_key":"new-sk"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	_, enabled = imageStorage.Resolver()()
	require.True(t, enabled)
	require.Len(t, built, 2, "the handler mutation must invalidate the cached uploader")
	require.Equal(t, "https://new.example.test", built[1].Endpoint)
	require.Equal(t, "new-ak", built[1].AccessKeyID)
	require.Equal(t, "new-sk", built[1].SecretAccessKey)
}
