// Package blobstore fournit des implémentations de port.BlobStore pour le
// stockage du contenu des pièces jointes (backend PostgreSQL ou object store S3).
package blobstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/statoon54/mailhive/internal/port"
)

// ErrBlobNotFound est retourné quand un blob demandé n'existe pas.
var ErrBlobNotFound = errors.New("blob introuvable")

// PostgresStore implémente port.BlobStore via la table attachment_blobs.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore crée un BlobStore adossé à PostgreSQL.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Put stocke le contenu sous (tenantID, sha256). Idempotent via ON CONFLICT.
func (s *PostgresStore) Put(ctx context.Context, tenantID uuid.UUID, sha256, content []byte) error {
	query := `
		INSERT INTO attachment_blobs (tenant_id, sha256, data)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, sha256) DO NOTHING`
	if _, err := s.pool.Exec(ctx, query, tenantID, sha256, content); err != nil {
		return fmt.Errorf("erreur de stockage du blob : %w", err)
	}
	return nil
}

// Get retourne le contenu stocké sous (tenantID, sha256).
func (s *PostgresStore) Get(ctx context.Context, tenantID uuid.UUID, sha256 []byte) ([]byte, error) {
	query := `SELECT data FROM attachment_blobs WHERE tenant_id = $1 AND sha256 = $2`
	var data []byte
	if err := s.pool.QueryRow(ctx, query, tenantID, sha256).Scan(&data); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBlobNotFound
		}
		return nil, fmt.Errorf("erreur de lecture du blob : %w", err)
	}
	return data, nil
}

// Delete supprime le contenu sous (tenantID, sha256). L'absence n'est pas une erreur.
func (s *PostgresStore) Delete(ctx context.Context, tenantID uuid.UUID, sha256 []byte) error {
	query := `DELETE FROM attachment_blobs WHERE tenant_id = $1 AND sha256 = $2`
	if _, err := s.pool.Exec(ctx, query, tenantID, sha256); err != nil {
		return fmt.Errorf("erreur de suppression du blob : %w", err)
	}
	return nil
}

// Exists indique si un contenu est présent sous (tenantID, sha256).
func (s *PostgresStore) Exists(ctx context.Context, tenantID uuid.UUID, sha256 []byte) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM attachment_blobs WHERE tenant_id = $1 AND sha256 = $2)`
	var exists bool
	if err := s.pool.QueryRow(ctx, query, tenantID, sha256).Scan(&exists); err != nil {
		return false, fmt.Errorf("erreur de vérification du blob : %w", err)
	}
	return exists, nil
}

// Vérifie à la compilation que PostgresStore satisfait port.BlobStore.
var _ port.BlobStore = (*PostgresStore)(nil)
