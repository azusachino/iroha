module github.com/azusachino/iroha/apps/iroha-server

go 1.27

require (
	github.com/azusachino/iroha/apps/iroha-core v0.1.0
	github.com/azusachino/iroha/apps/iroha-imports v0.1.0
	github.com/azusachino/iroha/apps/iroha-providers v0.1.0
	github.com/azusachino/iroha/apps/iroha-runtime v0.1.0
	github.com/descope/virtualwebauthn v1.0.5
	github.com/go-chi/chi/v5 v5.3.2
	github.com/go-chi/cors v1.2.2
	github.com/go-chi/httprate v0.16.0
	github.com/go-webauthn/webauthn v0.18.2
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.10.0
	golang.org/x/crypto v0.57.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/postgres v1.6.3
	gorm.io/gorm v1.31.2
)

replace github.com/azusachino/iroha/apps/iroha-core => ../iroha-core

replace github.com/azusachino/iroha/apps/iroha-imports => ../iroha-imports

replace github.com/azusachino/iroha/apps/iroha-providers => ../iroha-providers

replace github.com/azusachino/iroha/apps/iroha-runtime => ../iroha-runtime

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.4 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/go-webauthn/x v0.3.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/redis/go-redis/v9 v9.22.0 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
