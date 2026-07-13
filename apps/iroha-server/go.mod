module github.com/azusachino/iroha/apps/iroha-server

go 1.26.4

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/go-chi/chi/v5 v5.2.3
	github.com/go-chi/cors v1.2.2
	github.com/go-chi/httprate v0.16.0
	github.com/google/uuid v1.6.0
	github.com/azusachino/iroha/apps/iroha-core v0.0.0-00010101000000-000000000000
	github.com/azusachino/iroha/apps/iroha-providers v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.21.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
)

replace github.com/azusachino/iroha/apps/iroha-core => ../iroha-core

replace github.com/azusachino/iroha/apps/iroha-providers => ../iroha-providers

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)
