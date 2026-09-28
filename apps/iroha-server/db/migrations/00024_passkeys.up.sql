-- Passkeys (ADR-0008 item 6). Only public-key credential state is stored,
-- never a private key. The user handle is an opaque random id sent to
-- authenticators instead of the internal user id.
alter table tb_users add column webauthn_handle bytea;
alter table tb_users add constraint tb_users_webauthn_handle_key unique (webauthn_handle);

-- Adding or removing a passkey needs a recent password confirmation.
alter table tb_sessions add column reauthenticated_at timestamptz;

create table tb_passkeys (
  id uuid primary key,
  user_id uuid not null references tb_users(id) on delete cascade,
  credential_id bytea not null,
  credential jsonb not null,
  name text not null,
  created_at timestamptz not null,
  last_used_at timestamptz,
  constraint tb_passkeys_credential_id_key unique (credential_id),
  constraint tb_passkeys_name_check check (char_length(name) between 1 and 64)
);
create index idx_tb_passkeys_user on tb_passkeys(user_id);
