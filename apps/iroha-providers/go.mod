module github.com/azusachino/iroha/apps/iroha-providers

go 1.26.4

require github.com/azusachino/iroha/apps/iroha-core v0.1.0

require github.com/google/uuid v1.6.0 // indirect

replace github.com/azusachino/iroha/apps/iroha-core => ../iroha-core
