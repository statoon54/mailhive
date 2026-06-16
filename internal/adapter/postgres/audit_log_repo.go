package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/statoon54/mailhive/internal/domain"
)

// AuditLogRepository implémente port.AuditLogRepository avec PostgreSQL.
type AuditLogRepository struct {
	pool *pgxpool.Pool
}

// NewAuditLogRepository crée un nouveau repository audit log.
func NewAuditLogRepository(pool *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{pool: pool}
}

// Create insère une entrée d'audit.
func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, tenant_id, action, resource_type, resource_id, status, status_code, error_message, details, method, path, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.pool.Exec(ctx, query,
		log.ID, log.TenantID, log.Action, log.ResourceType, log.ResourceID,
		log.Status, log.StatusCode, log.ErrorMessage, log.Details, log.Method, log.Path, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("erreur de création du log d'audit : %w", err)
	}
	return nil
}

// List retourne les logs d'audit paginés avec filtres (cross-tenant, JOIN tenants).
func (r *AuditLogRepository) List(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedList[domain.AuditLog], error) {
	offset := (filter.Page - 1) * filter.Limit

	// Comptage
	countQuery := `SELECT COUNT(*) FROM audit_logs a`
	countArgs := []any{}
	conditions := []string{}

	argN := 1
	if filter.TenantID != nil {
		conditions = append(conditions, fmt.Sprintf("a.tenant_id = $%d", argN))
		countArgs = append(countArgs, *filter.TenantID)
		argN++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argN))
		countArgs = append(countArgs, *filter.Status)
		argN++
	}
	if filter.ResourceType != nil {
		conditions = append(conditions, fmt.Sprintf("a.resource_type = $%d", argN))
		countArgs = append(countArgs, *filter.ResourceType)
		argN++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	err := r.pool.QueryRow(ctx, countQuery+whereClause, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("erreur de comptage des logs d'audit : %w", err)
	}

	// Liste avec JOIN tenants
	listQuery := `
		SELECT a.id, a.tenant_id, t.name, a.action, a.resource_type, a.resource_id,
			a.status, a.status_code, a.error_message, COALESCE(a.details, ''), a.method, a.path, a.created_at
		FROM audit_logs a
		JOIN tenants t ON t.id = a.tenant_id` + whereClause
	listArgs := make([]any, len(countArgs))
	copy(listArgs, countArgs)

	listQuery += fmt.Sprintf(" ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d", argN, argN+1)
	listArgs = append(listArgs, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("erreur de listage des logs d'audit : %w", err)
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var createdAt time.Time
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.TenantName, &l.Action, &l.ResourceType, &l.ResourceID,
			&l.Status, &l.StatusCode, &l.ErrorMessage, &l.Details, &l.Method, &l.Path, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("erreur de lecture du log d'audit : %w", err)
		}
		l.CreatedAt = createdAt
		logs = append(logs, l)
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &domain.PaginatedList[domain.AuditLog]{
		Items:      logs,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}
