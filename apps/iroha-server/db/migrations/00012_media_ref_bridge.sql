-- +goose Up

-- Replaces the two ConfigMap-mounted JSON files (bangumi_to_mal.json,
-- mal_to_anilist.json) iroha-job used to load TwoHopMediaRefBridge from at
-- startup (apps/iroha-imports/media_resolution.go). One table for both hops
-- via a discriminator, not two tables, since the app treats this as one
-- Bangumi -> MAL -> AniList chain, not a generic per-provider table set.
create table tb_media_ref_bridge (
  hop text not null,
  source_id text not null,
  target_id text not null,
  updated_at timestamptz not null default now(),
  primary key (hop, source_id)
);

-- +goose Down

drop table tb_media_ref_bridge;
