module github.com/azusachino/iroha/apps/iroha-job

go 1.26.4

require (
	github.com/azusachino/iroha/apps/iroha-core v0.0.0-00010101000000-000000000000
	github.com/azusachino/iroha/apps/iroha-server v0.0.0-00010101000000-000000000000
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

replace github.com/azusachino/iroha/apps/iroha-server => ../iroha-server

replace github.com/azusachino/iroha/apps/iroha-core => ../iroha-core
