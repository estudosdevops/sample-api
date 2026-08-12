package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/estudosdevops/sample-api/internal/domain"
	"github.com/estudosdevops/sample-api/internal/infra"
	_ "github.com/lib/pq"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type PostgresRepo struct {
	db      *sql.DB
	metrics *infra.BusinessMetrics
}

type txKey struct{}

func NewPostgres(dsn string) (*PostgresRepo, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	// configure pool (placeholders)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return &PostgresRepo{
		db:      db,
		metrics: infra.InitBusinessMetrics(),
	}, nil
}

func (p *PostgresRepo) Close() error {
	return p.db.Close()
}

// Ping checks connectivity to Postgres.
func (p *PostgresRepo) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// GetByCEP queries the addresses table for a given CEP.
func (p *PostgresRepo) GetByCEP(ctx context.Context, cep string) (*domain.Address, error) {
	tracer := otel.Tracer("postgres-repo")
	ctx, span := tracer.Start(ctx, "GetByCEP", trace.WithAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.statement", "SELECT_FROM_addresses_by_cep"),
	))
	defer span.End()

	var a domain.Address
	q := `SELECT cep, street, city, state FROM addresses WHERE cep = $1 LIMIT 1`

	// use transaction if present
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		row := tx.QueryRowContext(ctx, q, cep)
		if err := row.Scan(&a.CEP, &a.Street, &a.City, &a.State); err != nil {
			if err == sql.ErrNoRows {
				span.SetStatus(codes.Error, "not found")
				p.metrics.RecordDBOp(ctx, "select", err)
				return nil, domain.ErrNotFound
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			p.metrics.RecordDBOp(ctx, "select", err)
			infra.LoggerFromContext(ctx).Warn("db query error", "error", err, "cep", cep)
			return nil, fmt.Errorf("query address: %w", err)
		}
		p.metrics.RecordDBOp(ctx, "select", nil)
		return &a, nil
	}

	row := p.db.QueryRowContext(ctx, q, cep)
	if err := row.Scan(&a.CEP, &a.Street, &a.City, &a.State); err != nil {
		if err == sql.ErrNoRows {
			span.SetStatus(codes.Error, "not found")
			p.metrics.RecordDBOp(ctx, "select", err)
			return nil, domain.ErrNotFound
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		p.metrics.RecordDBOp(ctx, "select", err)
		infra.LoggerFromContext(ctx).Warn("db query error", "error", err, "cep", cep)
		return nil, fmt.Errorf("query address: %w", err)
	}
	p.metrics.RecordDBOp(ctx, "select", nil)
	return &a, nil
}

// Insert inserts an address into the addresses table.
func (p *PostgresRepo) Insert(ctx context.Context, a *domain.Address) error {
	tracer := otel.Tracer("postgres-repo")
	ctx, span := tracer.Start(ctx, "Insert", trace.WithAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.statement", "INSERT_INTO_addresses"),
	))
	defer span.End()

	q := `INSERT INTO addresses (cep, street, city, state) VALUES ($1, $2, $3, $4) ON CONFLICT (cep) DO UPDATE SET street=EXCLUDED.street, city=EXCLUDED.city, state=EXCLUDED.state`

	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		if _, err := tx.ExecContext(ctx, q, a.CEP, a.Street, a.City, a.State); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			p.metrics.RecordDBOp(ctx, "insert", err)
			infra.LoggerFromContext(ctx).Warn("db insert error", "error", err, "cep", a.CEP)
			return fmt.Errorf("insert address: %w", err)
		}
		p.metrics.RecordDBOp(ctx, "insert", nil)
		infra.LoggerFromContext(ctx).Info("db insert (tx)", "cep", a.CEP)
		return nil
	}

	if _, err := p.db.ExecContext(ctx, q, a.CEP, a.Street, a.City, a.State); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		p.metrics.RecordDBOp(ctx, "insert", err)
		infra.LoggerFromContext(ctx).Warn("db insert error", "error", err, "cep", a.CEP)
		return fmt.Errorf("insert address: %w", err)
	}
	p.metrics.RecordDBOp(ctx, "insert", nil)
	infra.LoggerFromContext(ctx).Info("db insert", "cep", a.CEP)
	return nil
}

// BeginTx starts a DB transaction and returns a context containing it.
func (p *PostgresRepo) BeginTx(ctx context.Context) (context.Context, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, txKey{}, tx), nil
}

// Commit commits a transaction stored in the context.
func (p *PostgresRepo) Commit(ctx context.Context) error {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx.Commit()
	}
	return nil
}

// Rollback rolls back a transaction stored in the context.
func (p *PostgresRepo) Rollback(ctx context.Context) error {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx.Rollback()
	}
	return nil
}
