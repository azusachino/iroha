module github.com/azusachino/iroha/apps/iroha-imports

go 1.26.4

require (
	github.com/azusachino/iroha/apps/iroha-core v0.1.0
	github.com/azusachino/iroha/apps/iroha-providers v0.1.0
	github.com/azusachino/iroha/apps/iroha-server v0.1.0
	github.com/google/uuid v1.6.0
	gorm.io/gorm v1.31.1
)

replace github.com/azusachino/iroha/apps/iroha-core => ../iroha-core
replace github.com/azusachino/iroha/apps/iroha-providers => ../iroha-providers
replace github.com/azusachino/iroha/apps/iroha-server => ../iroha-server
