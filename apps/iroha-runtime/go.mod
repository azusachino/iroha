module github.com/azusachino/iroha/apps/iroha-runtime

go 1.26.4

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/azusachino/iroha/apps/iroha-core v0.1.0
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.21.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.9.2 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/text v0.39.0 // indirect
)

replace github.com/azusachino/iroha/apps/iroha-core => ../iroha-core
