drop table tb_passkeys;
alter table tb_sessions drop column reauthenticated_at;
alter table tb_users drop constraint tb_users_webauthn_handle_key;
alter table tb_users drop column webauthn_handle;
