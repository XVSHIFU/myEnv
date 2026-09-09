package state

import (
	"context"
	"database/sql"
	"errors"
)

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func loadPythonEntry(ctx context.Context, query rowQuerier, g *Generation) error {
	err := query.QueryRowContext(ctx, `SELECT executable FROM generation_python WHERE generation_id=?`, g.ID).Scan(&g.PythonExecutable)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}
