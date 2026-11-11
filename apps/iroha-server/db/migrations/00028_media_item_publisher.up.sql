-- A consumed edition can name its publisher (a Drifter/WeRead book, a
-- Hardcover edition). The composer/publisher stays on the item, not the work:
-- two editions of one work may be published by different houses.
alter table tb_media_items add column publisher text not null default '';
