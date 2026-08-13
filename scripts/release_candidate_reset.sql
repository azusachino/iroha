do $$
declare
  tables text;
begin
  select string_agg(format('%I.%I', schemaname, tablename), ', ')
    into tables
    from pg_tables
   where schemaname = 'public'
     and tablename <> 'goose_db_version'
     and tablename <> 'spatial_ref_sys';
  if tables is not null then
    execute 'truncate table ' || tables || ' cascade';
  end if;
end
$$;
