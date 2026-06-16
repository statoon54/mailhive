//go:build seaweedfs

// Package test — validation runtime de l'adaptateur S3 (port.BlobStore) contre une
// vraie passerelle S3 SeaweedFS.
//
// Exclu du `go test ./...` par défaut (build tag) car il démarre un conteneur
// SeaweedFS. Pour l'exécuter :
//
//	go test -tags seaweedfs ./internal/test/ -run TestS3Store -v
package test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/statoon54/mailhive/internal/adapter/blobstore"
	"github.com/statoon54/mailhive/internal/config"
)

const (
	s3TestAccessKey = "mailhive-test"
	s3TestSecretKey = "mailhive-test-secret"
)

// seaweedS3Identity est la config d'identité S3 montée dans SeaweedFS : sans
// elle, la passerelle S3 rejette les requêtes signées ("requires setting up
// SeaweedFS S3 authentication"). On accorde un accès Admin aux clés de test.
const seaweedS3Identity = `{
  "identities": [
    {
      "name": "mailhive-test",
      "credentials": [
        { "accessKey": "mailhive-test", "secretKey": "mailhive-test-secret" }
      ],
      "actions": ["Admin", "Read", "Write", "List", "Tagging"]
    }
  ]
}`

// startSeaweedFS démarre un conteneur SeaweedFS exposant sa passerelle S3 (port 8333)
// et retourne la config Blob pointant dessus.
func startSeaweedFS(t *testing.T) config.BlobConfig {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "chrislusf/seaweedfs:latest",
		Cmd:          []string{"server", "-s3", "-s3.port=8333", "-s3.config=/etc/seaweedfs/s3.json", "-dir=/tmp"},
		ExposedPorts: []string{"8333/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				Reader:            strings.NewReader(seaweedS3Identity),
				ContainerFilePath: "/etc/seaweedfs/s3.json",
				FileMode:          0o644,
			},
		},
		WaitingFor: wait.ForListeningPort("8333/tcp").
			WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "démarrage du conteneur SeaweedFS")
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "8333/tcp")
	require.NoError(t, err)

	return config.BlobConfig{
		Backend:     "s3",
		S3Endpoint:  fmt.Sprintf("%s:%s", host, port.Port()),
		S3Bucket:    "mailhive-attachments-test",
		S3AccessKey: s3TestAccessKey,
		S3SecretKey: s3TestSecretKey,
		S3Region:    "us-east-1",
		S3UseSSL:    false,
	}
}

func TestS3Store_RoundTrip(t *testing.T) {
	ctx := context.Background()
	cfg := startSeaweedFS(t)

	store, err := blobstore.NewS3Store(ctx, cfg)
	require.NoError(t, err, "création du S3Store (bucket auto-créé)")

	tenantID := uuid.New()
	content := []byte("contenu binaire stocké sur SeaweedFS")
	sum := sha256.Sum256(content)
	hash := sum[:]

	// Absent au départ.
	exists, err := store.Exists(ctx, tenantID, hash)
	require.NoError(t, err)
	assert.False(t, exists)

	// Put puis Get.
	require.NoError(t, store.Put(ctx, tenantID, hash, content))
	got, err := store.Get(ctx, tenantID, hash)
	require.NoError(t, err)
	assert.Equal(t, content, got)

	exists, err = store.Exists(ctx, tenantID, hash)
	require.NoError(t, err)
	assert.True(t, exists)

	// Put idempotent (dédup au niveau stockage : pas de réécriture).
	require.NoError(t, store.Put(ctx, tenantID, hash, content))

	// Delete puis absence.
	require.NoError(t, store.Delete(ctx, tenantID, hash))
	exists, err = store.Exists(ctx, tenantID, hash)
	require.NoError(t, err)
	assert.False(t, exists)

	_, err = store.Get(ctx, tenantID, hash)
	assert.Error(t, err, "Get après Delete doit échouer")
}

func TestS3Store_IsolatedPerTenant(t *testing.T) {
	ctx := context.Background()
	cfg := startSeaweedFS(t)

	store, err := blobstore.NewS3Store(ctx, cfg)
	require.NoError(t, err)

	content := []byte("même contenu, deux tenants")
	sum := sha256.Sum256(content)
	hash := sum[:]

	tenantA := uuid.New()
	tenantB := uuid.New()

	require.NoError(t, store.Put(ctx, tenantA, hash, content))

	// Le tenant B ne voit pas le blob du tenant A (clé scopée tenant_id/sha256).
	exists, err := store.Exists(ctx, tenantB, hash)
	require.NoError(t, err)
	assert.False(t, exists, "le blob doit être isolé par tenant")

	exists, err = store.Exists(ctx, tenantA, hash)
	require.NoError(t, err)
	assert.True(t, exists)
}
