package cache

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// PostgresStore stores cache entries in logged Postgres tables. Namespace
// generations make invalidation constant-time; old generations are disposable
// rows and can be cleaned up by a maintenance task.
type PostgresStore struct {
	db *gorm.DB
}

// NewPostgresStore builds a Store over the application's existing Postgres
// connection. The cache migration must be applied before use.
func NewPostgresStore(db *gorm.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, namespace, key string) ([]byte, bool, error) {
	var value []byte
	err := s.db.WithContext(ctx).Raw(`
		select e.value_json
		from tb_cache_entries e
		join tb_cache_namespaces n on n.namespace = e.namespace and n.generation = e.generation
		where e.namespace = ? and e.cache_key = ? and e.expires_at > now()
	`, namespace, key).Row().Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	return value, true, nil
}

func (s *PostgresStore) Set(ctx context.Context, namespace, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			insert into tb_cache_namespaces (namespace, generation, updated_at)
			values (?, 1, now())
			on conflict (namespace) do nothing
		`, namespace).Error; err != nil {
			return err
		}

		var generation int64
		if err := tx.Raw(`select generation from tb_cache_namespaces where namespace = ? for update`, namespace).Row().Scan(&generation); err != nil {
			return err
		}

		return tx.Exec(`
			insert into tb_cache_entries (namespace, cache_key, generation, value_json, expires_at, created_at, updated_at)
			values (?, ?, ?, ?::jsonb, ?, now(), now())
			on conflict (namespace, cache_key) do update set
				generation = excluded.generation,
				value_json = excluded.value_json,
				expires_at = excluded.expires_at,
				updated_at = excluded.updated_at
		`, namespace, key, generation, value, time.Now().UTC().Add(ttl)).Error
	})
}

func (s *PostgresStore) InvalidateNamespace(ctx context.Context, namespace string) error {
	return s.db.WithContext(ctx).Exec(`
		insert into tb_cache_namespaces (namespace, generation, updated_at)
		values (?, 1, now())
		on conflict (namespace) do update set
			generation = tb_cache_namespaces.generation + 1,
			updated_at = excluded.updated_at
	`, namespace).Error
}

func (s *PostgresStore) Close() error { return nil }
