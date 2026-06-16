package blobstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/statoon54/mailhive/internal/config"
	"github.com/statoon54/mailhive/internal/port"
)

// Au démarrage de la stack, la passerelle S3 (SeaweedFS/MinIO) peut n'être pas
// encore à l'écoute alors que mailhive démarre déjà (il n'attend que Postgres et
// Redis). On retente donc la connexion quelques secondes avant d'abandonner.
const (
	s3InitMaxAttempts = 15
	s3InitRetryDelay  = 2 * time.Second
)

// S3Store implémente port.BlobStore sur un object store compatible S3
// (SeaweedFS, MinIO, S3, R2). La clé objet est "<tenant_id>/<sha256 hex>".
type S3Store struct {
	client *minio.Client
	bucket string
}

// NewS3Store crée un BlobStore S3 et s'assure que le bucket existe.
func NewS3Store(ctx context.Context, cfg config.BlobConfig) (*S3Store, error) {
	client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
		Region: cfg.S3Region,
	})
	if err != nil {
		return nil, fmt.Errorf("erreur d'initialisation du client S3 : %w", err)
	}

	if err := ensureBucket(ctx, client, cfg); err != nil {
		return nil, err
	}

	return &S3Store{client: client, bucket: cfg.S3Bucket}, nil
}

// ensureBucket s'assure que le bucket existe (le crée au besoin), en retentant la
// connexion tant que la passerelle S3 n'est pas joignable (course au démarrage).
func ensureBucket(ctx context.Context, client *minio.Client, cfg config.BlobConfig) error {
	var lastErr error
	for attempt := 1; attempt <= s3InitMaxAttempts; attempt++ {
		exists, err := client.BucketExists(ctx, cfg.S3Bucket)
		if err == nil {
			if exists {
				return nil
			}
			if err := client.MakeBucket(ctx, cfg.S3Bucket, minio.MakeBucketOptions{Region: cfg.S3Region}); err != nil {
				return fmt.Errorf("erreur de création du bucket S3 : %w", err)
			}
			return nil
		}

		lastErr = err
		slog.Warn("passerelle S3 injoignable, nouvelle tentative",
			"endpoint", cfg.S3Endpoint, "attempt", attempt, "max", s3InitMaxAttempts, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s3InitRetryDelay):
		}
	}
	return fmt.Errorf("erreur de vérification du bucket S3 après %d tentatives : %w", s3InitMaxAttempts, lastErr)
}

// objectKey construit la clé d'objet déterministe pour (tenant, contenu).
func objectKey(tenantID uuid.UUID, sha256 []byte) string {
	return fmt.Sprintf("%s/%x", tenantID, sha256)
}

// Put stocke le contenu. Idempotent : si l'objet existe déjà, ne réécrit pas.
func (s *S3Store) Put(ctx context.Context, tenantID uuid.UUID, sha256, content []byte) error {
	key := objectKey(tenantID, sha256)

	exists, err := s.objectExists(ctx, key)
	if err != nil {
		return err
	}
	if exists {
		return nil // déduplication au niveau stockage
	}

	_, err = s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(content), int64(len(content)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return fmt.Errorf("erreur de stockage de l'objet S3 : %w", err)
	}
	return nil
}

// Get retourne le contenu stocké sous (tenantID, sha256).
func (s *S3Store) Get(ctx context.Context, tenantID uuid.UUID, sha256 []byte) ([]byte, error) {
	key := objectKey(tenantID, sha256)
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("erreur de lecture de l'objet S3 : %w", err)
	}
	defer func() { _ = obj.Close() }()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(obj); err != nil {
		if isNotFound(err) {
			return nil, ErrBlobNotFound
		}
		return nil, fmt.Errorf("erreur de lecture du contenu S3 : %w", err)
	}
	return buf.Bytes(), nil
}

// Delete supprime le contenu. L'absence n'est pas une erreur.
func (s *S3Store) Delete(ctx context.Context, tenantID uuid.UUID, sha256 []byte) error {
	key := objectKey(tenantID, sha256)
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("erreur de suppression de l'objet S3 : %w", err)
	}
	return nil
}

// Exists indique si un contenu est présent sous (tenantID, sha256).
func (s *S3Store) Exists(ctx context.Context, tenantID uuid.UUID, sha256 []byte) (bool, error) {
	return s.objectExists(ctx, objectKey(tenantID, sha256))
}

func (s *S3Store) objectExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("erreur de vérification de l'objet S3 : %w", err)
	}
	return true, nil
}

// isNotFound détecte l'erreur "objet introuvable" du client minio.
func isNotFound(err error) bool {
	var resp minio.ErrorResponse
	if errors.As(err, &resp) {
		return resp.Code == "NoSuchKey" || resp.StatusCode == 404
	}
	return false
}

// Vérifie à la compilation que S3Store satisfait port.BlobStore.
var _ port.BlobStore = (*S3Store)(nil)
