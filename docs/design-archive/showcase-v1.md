# Showcase v1 — day constellation

Status: archived reference, 2026-07-18

Source: retired route, preserved here as a decision record.

The adopted pieces now live in `apps/iroha-web/src/routes/+page.svelte`.

## Intent

Make a first visit feel like hearing a day take shape and seeing it bloom: the
day is a pattern to notice, not a spreadsheet to operate. The page uses a large
editorial promise, a central readiness signal, instrument cards, and a short
activity / recovery narrative.

## Keep

- A strong first-screen promise rather than a generic dashboard title.
- One dominant signal that explains the rest of the page.
- A small number of composable instruments with clear labels.
- Rich negative space, restrained color, and data-driven narrative copy.
- A graceful sample state when the private API is unavailable.

## Do not copy blindly

- The constellation is a concept surface, not the primary navigation model.
- Decorative orbit geometry must never obscure the actual value, date, or
  provenance of a measurement.
- “Explore” is not a replacement for task-oriented routes and filters.

## Production consequences

The production app uses the showcase tone for page hierarchy and visual
weight, while each domain remains task-first:

- Today answers “what should I notice now?”
- Patterns answers “what changed over time?”
- Motion answers “which session do I want to inspect?”
- Night answers “how did recovery behave?”
- Library answers “what am I continuing or choosing next?”
- Share answers “what can I safely show someone else?”
