package domain

import "github.com/google/uuid"

// AttachmentMeta décrit une pièce jointe dédupliquée stockée (content-addressed).
// Le contenu lui-même n'y figure pas : il vit dans le BlobStore (backend postgres
// ou s3) sous la clé SHA256.
type AttachmentMeta struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	ContentType string
	Storage     string // "postgres" | "s3"
	SHA256      []byte
	Size        int64
}

// AttachmentLink lie un mail à une pièce jointe dédupliquée. Le filename est
// propre au mail (deux mails peuvent référencer le même contenu sous des noms
// différents) ; la déduplication porte sur le contenu, pas sur le nom.
type AttachmentLink struct {
	AttachmentID uuid.UUID
	Filename     string
	Position     int
}

// AttachmentRef est une pièce jointe résolue côté lecture : ses métadonnées
// (taille, type) plus le filename du mail. Sert au worker pour recharger le
// contenu via le BlobStore avant l'envoi SMTP.
type AttachmentRef struct {
	AttachmentID uuid.UUID `json:"attachment_id"`
	ContentType  string    `json:"content_type"`
	Filename     string    `json:"filename"`
	Storage      string    `json:"-"`
	SHA256       []byte    `json:"-"`
	Size         int64     `json:"size"`
}

// Storage backends pour les pièces jointes.
const (
	AttachmentStoragePostgres = "postgres"
	AttachmentStorageS3       = "s3"
)
