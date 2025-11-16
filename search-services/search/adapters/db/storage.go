package db

import (
	"context"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/search/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {

	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Read(ctx context.Context) ([]core.DBComic, error) {
	query := `
			SELECT (id, url, words) FROM comics
	`
	var comics []struct {
		ID    int            `db:"id"`
		URL   string         `db:"url"`
		Words map[string]int `db:"words"`
	}

	err := db.conn.SelectContext(ctx, &comics, query)
	if err != nil {
		db.log.Error("db.read error", "error", err)
		return nil, err
	}
	var res []core.DBComic
	for _, c := range comics {
		res = append(res, core.DBComic{
			ID:    c.ID,
			URL:   c.URL,
			Words: c.Words,
		})
	}
	return res, nil
}
