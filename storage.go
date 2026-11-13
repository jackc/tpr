package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tpr/backend/data"
	"github.com/jackc/tpr/backend/dualstore"
	"github.com/jackc/tpr/backend/jedstore"
	"github.com/urfave/cli/v3"
)

func openStore(c *cli.Command, logger *slog.Logger) (data.DB, func(), error) {
	mode := c.String("store")
	if mode != "postgres" && mode != "jed" && mode != "dual" {
		return nil, nil, fmt.Errorf("--store must be postgres, jed, or dual")
	}
	if mode != "postgres" && os.Getenv("TEST_ENDPOINTS") == "true" {
		return nil, nil, fmt.Errorf("TEST_ENDPOINTS requires PostgreSQL; use store integration tests for JED and dual mode")
	}
	if mode == "postgres" {
		p, e := newPool(c, logger)
		if e != nil {
			return nil, nil, e
		}
		return p, p.Close, nil
	}
	path := c.String("jed-path")
	if path == "" {
		return nil, nil, fmt.Errorf("--jed-path is required for %s storage", mode)
	}
	j, e := jedstore.Open(path)
	if e != nil {
		return nil, nil, e
	}
	if mode == "jed" {
		return j, func() { j.Close() }, nil
	}
	p, e := newPool(c, logger)
	if e != nil {
		j.Close()
		return nil, nil, e
	}
	d := dualstore.New(p, j)
	if err := d.Verify(context.Background()); err != nil {
		p.Close()
		j.Close()
		return nil, nil, err
	}
	return d, func() { p.Close(); j.Close() }, nil
}

func copyToJED(ctx context.Context, c *cli.Command) error {
	logger, err := newLogger(c)
	if err != nil {
		return err
	}
	p, err := newPool(c, logger)
	if err != nil {
		return err
	}
	defer p.Close()
	if c.String("jed-path") == "" {
		return fmt.Errorf("--jed-path is required")
	}
	j, err := jedstore.Open(c.String("jed-path"))
	if err != nil {
		return err
	}
	defer j.Close()
	tx, err := p.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := data.CopySnapshot(ctx, tx, j); err != nil {
		return err
	}
	if err := dualstore.New(tx, j).Verify(ctx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
