DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'iroha') THEN
        CREATE ROLE iroha LOGIN PASSWORD 'iroha_dev' SUPERUSER;
    END IF;
END
$$;

ALTER DATABASE iroha OWNER TO iroha;
