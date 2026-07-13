module github.com/azusachino/iroha/apps/iroha-imports

go 1.26.4

require (
	github.com/azusachino/iroha/apps/iroha-core v0.1.0
	github.com/azusachino/iroha/apps/iroha-providers v0.1.0
	github.com/azusachino/iroha/apps/iroha-runtime v0.1.0
	github.com/google/uuid v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/redis/go-redis/v9 v9.21.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

replace github.com/azusachino/iroha/apps/iroha-core => ../iroha-core

replace github.com/azusachino/iroha/apps/iroha-providers => ../iroha-providers

replace github.com/azusachino/iroha/apps/iroha-runtime => ../iroha-runtime
