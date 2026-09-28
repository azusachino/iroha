-- Optional display name shown in the user menu instead of the username.
alter table tb_users add column display_name text;
alter table tb_users add constraint tb_users_display_name_check
  check (display_name is null or (char_length(display_name) between 1 and 64));
