-- Single-owner authentication (ADR-0008). The user record holds identity only;
-- the password hash and sessions live in separate auth records.
create table tb_users (
  id uuid primary key,
  username text not null,
  created_at timestamptz not null,
  constraint tb_users_username_key unique (username),
  constraint tb_users_username_check check (username ~ '^[a-z0-9][a-z0-9._-]{0,63}$')
);

-- At most one owner: setup is allowed only while this table is empty.
create unique index tb_users_single_owner on tb_users ((true));

create table tb_user_passwords (
  user_id uuid primary key references tb_users(id) on delete cascade,
  password_hash text not null,
  updated_at timestamptz not null,
  constraint tb_user_passwords_hash_check check (password_hash like '$argon2id$%')
);

-- Sessions are looked up by the SHA-256 of a random opaque identifier held in
-- an HttpOnly cookie; the identifier itself is never stored.
create table tb_sessions (
  id uuid primary key,
  user_id uuid not null references tb_users(id) on delete cascade,
  token_sha256 text not null,
  csrf_token text not null,
  created_at timestamptz not null,
  expires_at timestamptz not null,
  last_seen_at timestamptz not null,
  revoked_at timestamptz,
  constraint tb_sessions_token_sha256_key unique (token_sha256)
);
create index idx_tb_sessions_user on tb_sessions(user_id) where revoked_at is null;
