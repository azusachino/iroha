-- The single Health Auto Export intake credential. Only a verifier is stored;
-- the plaintext token is shown once at rotation and never persisted. The
-- singleton key makes rotation an atomic replace of the one row.
create table tb_health_intake_credential (
  singleton boolean primary key default true,
  token_sha256 text not null,
  rotated_at timestamptz not null,
  constraint tb_health_intake_credential_singleton_check check (singleton),
  constraint tb_health_intake_credential_sha_check check (token_sha256 <> '')
);
