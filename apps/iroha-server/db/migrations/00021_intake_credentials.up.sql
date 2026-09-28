-- Health Auto Export intake credentials, one per uploading device. Only a
-- SHA-256 verifier is stored; the plaintext token is shown once at issue.
-- Rotation issues a new row and revokes the old one, so both can overlap
-- while the device is updated.
create table tb_intake_credentials (
  id uuid primary key,
  name text not null,
  token_sha256 text not null,
  created_at timestamptz not null,
  last_used_at timestamptz,
  revoked_at timestamptz,
  constraint tb_intake_credentials_name_check check (name ~ '^[a-z0-9][a-z0-9-]*$'),
  constraint tb_intake_credentials_sha_check check (token_sha256 <> ''),
  constraint tb_intake_credentials_token_sha256_key unique (token_sha256)
);
