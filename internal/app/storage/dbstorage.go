package storage

import "database/sql"

type DBRepo struct {
	pool *sql.DB
}

func NewDBRepo(pool *sql.DB) *DBRepo {
	return &DBRepo{pool}
}

func (D DBRepo) Create(id string, originalURL string) string {
	//TODO implement me
	panic("implement me")
}

func (D DBRepo) Read(id string) string {
	//TODO implement me
	panic("implement me")
}

func (D DBRepo) Ping() error {
	return D.pool.Ping()
}
