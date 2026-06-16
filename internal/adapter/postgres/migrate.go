package postgres

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/statoon54/mailhive/internal/config"
)

// RunMigrations exécute les migrations SQL vers le haut.
// Le paramètre migrations est un embed.FS contenant les fichiers SQL.
func RunMigrations(cfg config.DBConfig, migrations embed.FS) error {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode,
	)

	source, err := iofs.New(migrations, ".")
	if err != nil {
		return fmt.Errorf("erreur de chargement des migrations : %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return fmt.Errorf("erreur d'initialisation des migrations : %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("erreur d'exécution des migrations : %w", err)
	}

	return nil
}

// RollbackMigrations annule la dernière migration.
func RollbackMigrations(cfg config.DBConfig, migrations embed.FS) error {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode,
	)

	source, err := iofs.New(migrations, ".")
	if err != nil {
		return fmt.Errorf("erreur de chargement des migrations : %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return fmt.Errorf("erreur d'initialisation des migrations : %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("erreur de rollback : %w", err)
	}

	return nil
}
