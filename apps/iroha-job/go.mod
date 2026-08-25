module github.com/azusachino/iroha/apps/iroha-job

go 1.27

require (
	github.com/azusachino/iroha/apps/iroha-core v0.1.0
	github.com/azusachino/iroha/apps/iroha-imports v0.1.0
	github.com/azusachino/iroha/apps/iroha-providers v0.1.0
	github.com/azusachino/iroha/apps/iroha-runtime v0.1.0
	github.com/azusachino/iroha/apps/iroha-server v0.1.0
	github.com/google/uuid v1.6.0
	gorm.io/gorm v1.31.2
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/redis/go-redis/v9 v9.22.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	gorm.io/driver/postgres v1.6.2 // indirect
)

replace github.com/azusachino/iroha/apps/iroha-core => ../iroha-core

replace github.com/azusachino/iroha/apps/iroha-imports => ../iroha-imports

replace github.com/azusachino/iroha/apps/iroha-providers => ../iroha-providers

replace github.com/azusachino/iroha/apps/iroha-runtime => ../iroha-runtime

replace github.com/azusachino/iroha/apps/iroha-server => ../iroha-server
