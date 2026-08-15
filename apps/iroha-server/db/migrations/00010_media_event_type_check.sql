-- +goose Up

-- Keep direct SQL writers from reintroducing provider state rows or an
-- unbounded event vocabulary. The application validates the same list, but
-- the canonical event table must enforce it at its own boundary as well.
alter table tb_media_consumption_events
  add constraint tb_media_consumption_events_allowed_type_check check (
    event_type in (
      'started', 'progressed', 'completed', 'finished', 'read', 'watched',
      'listened', 'reread', 'rewatched', 'abandoned', 'paused', 'reopened',
      'rated', 'noted', 'bookmarked'
    )
  );

-- +goose Down

alter table tb_media_consumption_events
  drop constraint if exists tb_media_consumption_events_allowed_type_check;
