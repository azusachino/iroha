# Iroha semantic motion plan

Status: reviewed and ready for implementation (revised after fresh-context review — see "Review corrections" below)

Target release: `v0.4.5`

Task board: Asobi epic `iroha:0.4.5-semantic-motion`

## Objective

Make the private cockpit feel alive, not just correct. `v0.4.4` fixed accessibility and truthful-state defects; it deliberately touched no motion. The result is functionally sound and visually flat: data
appears with a hard cut, confirmations are silent, menus open with no acknowledgment, headline numbers snap instead of counting, and switching between the six design languages is an instant repaint with
no sense that something changed. This patch gives Iroha a small, named vocabulary of motion tied to state change — not decoration, and not a Three.js/D3 dependency — and applies it to the surfaces the
owner actually watches every day: Today's data arriving, a gauge filling, a number counting, a menu opening, a design language switching.

This is a personal cockpit, not a compliance tool. "Interesting" is a real product requirement here, not a nice-to-have layered on top of correctness — see the retrospective on why `v0.4.4` landed as a
disappointment despite being defect-free.

## Review corrections (2026-08-24, fresh-context Opus review)

The first draft of this plan was reviewed before implementation started, per this repo's own culture of catching scope problems on paper instead of mid-patch. Four findings changed the plan materially
enough to record here rather than silently rewrite history:

1. **The existing global reduced-motion rule contradicts this plan's own Decision 4.** `themes.css` and `app.css` both force `transition-duration: 0.01ms !important; animation-duration: 0.01ms
   !important` on `*, *::before, *::after`. `!important` beats any more-specific component override, so no task below could ever ship a deliberate, non-instant reduced-motion result even where one is
   warranted — Task 2 (new) fixes this before any motion lands.
2. **The original Task 5 (command palette confirmation) assumed a component structure that doesn't exist.** `CommandPalette.svelte`'s `activate()` calls `closePalette()` synchronously and then `await
   goto()` — the palette unmounts before anything could animate, every command is a navigation with no success/failure branch, and a held confirmation on the hot Cmd+K path would violate Decision 5
   (frequent keyboard actions stay instant). Replaced with a design-language switch transition (Task 6, new) — the single most dramatic, currently-instant state change in the app, per the review's own
   assessment of what's "begging for motion." A real command-confirmation task is recorded under Follow-up instead of shipped half-designed.
3. **"NavigationMenu already has an open/close transition" was wrong.** Only the chevron icon rotates (`NavigationMenu.svelte:179`); `.navigation-popover` itself has no transition. Corrected in the
   evidence table below, and given its own task (Task 7, new) since the Objective already promised "a menu opening" as a felt example.
4. **Task 4 (gauges) undercounted the real surface.** Three `theme-ui/<language>/` files draw rings inline via raw `<svg>`/`stroke-dasharray` instead of the shared `RingGauge` component
   (`field-journal/Today.svelte`, `phenology/Today.svelte`, `phenology/MediaDetail.svelte`) — one already has an untokened `0.3s` transition, two have no transition at all (instant snap). Split into 5a
   (shared components) / 5b (the three per-language inline rings), matching `v0.4.4`'s own pattern for a fix that spans all `theme-ui/*/` copies per `AGENTS.md`'s "check the filename across all six"
   rule.
5. **Today's headline numbers were a missing felt-motion opportunity, not just an infrastructure gap.** Grapher's `Today.svelte` formats every metric (Move kcal, Exercise min, Steps, sleep efficiency
   %) through a plain `number()`/`toLocaleString` text binding (`grapher/Today.svelte:20-26`) with no transition — changing the selected day or refetching snaps every number instantly. Added as Task 8,
   scoped to Grapher only (per Decision 6) with the five-language extension recorded as immediate follow-on work rather than silently expanded into this task's budget.

Task numbering below reflects these corrections; there is no separate "old" numbering to reconcile.

## Inputs

- the workstation research note [`ThreeUI and transitions.dev for Iroha and Felicia`](../../../../docs/research/2026-08/2026-08-22-threeui-and-transitions-dev-for-iroha-felicia.md), specifically its
  "actual Iroha seams" table and "bounded next-fix queue" — this plan executes that queue's Iroha half.
- [`v0.4.4`'s own deferral](2026-08-24-iroha-0.4.4-ui-ux-quality.md): Decision 4 ("No D3 or Three.js dependency is expected") and Follow-up epic 1 ("Semantic motion: inventory `state change | owner |
  channels | timing | reduced-motion result | route`, classify public/private parity, then define only repeated semantic tokens and adopt one bounded vertical slice. Frequent keyboard actions remain
  instant.").
- [the frontend design contract](../frontend-design-contract.md) and [theme architecture](../frontend-theme-architecture.md) — unchanged by this plan, but the six design languages' personalities
  constrain how loud any given transition family is allowed to be per language.
- transitions.dev's semantic sequence: `semantic state change -> affected channels -> transition family -> token values -> orchestration -> reduced motion`. Cited as a vocabulary, not a dependency —
  nothing from `refs/motion-design/transitions-dev` is copied verbatim.

## Current situation

### What already exists

- A global reduced-motion catch-all in `packages/iroha-shared/src/theme/themes.css` and a duplicate in `apps/iroha-web/src/routes/app.css` — currently too blunt (see Review correction 1), fixed by
  Task 2 rather than removed outright, since the intent (stop decorative/infinite motion) is right.
- Explicit reduced-motion handling already scoped correctly in MapLibre and the six ECharts-backed chart components (`BarChart`, `DailySmallMultiples`, `FusedActivityChart`,
  `SleepArchitectureChart`, `SleepTimelineChart`, `YearProgressChart` all pass `animation: !reducedMotion` into ECharts' own option object) — these are confirmed out of scope for the CSS token work
  below; their animation timing is ECharts-internal, not a CSS literal, and already respects motion preference.
- `RingGauge.svelte` already animates `stroke-dasharray` on value change (`transition: stroke-dasharray 0.6s ease`) — the best existing instance of felt motion in the app, but not yet a named,
  reusable pattern, and not applied consistently (see Review correction 4).
- `LoadingBoundary.svelte`'s shimmer already has its own local reduced-motion override (`animation: none` under its own `@media` block) — proof the per-component override pattern already works where
  it's been applied; it just hasn't been applied everywhere yet.

### Verified gaps (evidence: `rg` sweep across `packages/iroha-shared/src` and `apps/iroha-web/src`, 2026-08-24, spot-checked line-by-line)

| Priority | Finding | Evidence | Consequence |
| --- | --- | --- | --- |
| P0 | The global reduced-motion rule uses `!important` on `transition-duration`/`animation-duration`, which forecloses any deliberate (non-instant) reduced-motion result anywhere in the app. | `themes.css:53-61`, `app.css:627-635`. | Directly contradicts this plan's Decision 4; must be fixed before any other task's reduced-motion acceptance criterion is even checkable. |
| P0 | No shared motion token exists anywhere in the app. | `themes.css` defines zero `--motion-*` custom properties. | A future fix has nothing to reuse and will add yet another one-off value. |
| P0 | Durations are scattered with no semantic grouping: `140ms, 160ms(x3), 180ms(x3), 0.12s(x6), 0.2s(x2), 0.3s, 0.6s, 1.1s, 1.4s, 1.8s`. | `app.css:421-556`, `NavigationMenu.svelte:179`, `MonthNavigator.svelte:340`, `manual/+page.svelte:508-509`, `library/+page.svelte:376-622`, `RingGauge.svelte:102`, `phenology/Today.svelte:294`, `TodaySkeleton.svelte:95,131`, `LoadingBoundary.svelte:84`. | No way to tell which durations are load-bearing (chart pacing) versus accidental drift. |
| P1 | Three `theme-ui/<language>/` files draw rings by hand instead of using the shared `RingGauge`, with inconsistent (or absent) transitions. | `field-journal/Today.svelte:76` (no transition), `phenology/Today.svelte:67,75,294` (has `0.3s ease`, untokened), `phenology/MediaDetail.svelte:92` (no transition). | The one good motion pattern in the app isn't actually applied everywhere it visually should be; two languages snap their rings instantly. |
| P1 | `LoadingBoundary`'s loading -> ready transition is a hard `{#if}/{:else}` swap. | `LoadingBoundary.svelte:29-59` — no transition directive on either branch. | Every route's first paint is a snap-cut, the single most-seen state change in the app. |
| P1 | The "Updating…" pill and first-load overlay appear/disappear with no enter/exit motion. | `LoadingBoundary.svelte:48-57`. | Refetches (period change, filter change) look like layout jitter, not a status update. |
| P1 | `.navigation-popover` has no open/close transition; only its chevron icon rotates. | `NavigationMenu.svelte:179` (chevron only), `NavigationMenu.svelte:187-200` (popover, no `transition`). | Opening Domains/Analyze/More is a hard cut, despite the app bar sitting on every route. |
| P1 | Switching design languages (`.language-picker`) is an instant full repaint via `html:root[data-language]`. | `DesignLanguagePicker.svelte` + `ThemeFrame.svelte`'s `Shell` swap — no transition anywhere in the chain. | The single most dramatic, most-often-triggered state change in the app (six genuinely different visual identities) has zero acknowledgment. |
| P2 | Today's headline numbers (Move kcal, Exercise min, Steps, sleep efficiency %) snap instantly on day change or refetch — no count/roll motion. | `grapher/Today.svelte:20-26` (`number()`/`toLocaleString` plain text binding), `+page.svelte:237,285` (sleep efficiency, same pattern). | The one place the owner watches a number change every day has no felt update. |
| P2 | No action in the app has a success or error confirmation motion. | No `success`/`confirm`/`shake` hits under `apps/iroha-web/src`, `packages/iroha-shared/src` outside test/unrelated CSS. | Recorded as a real gap, but not fixed in this plan — see Review correction 2 and Follow-up. |

## Decisions and constraints

1. **Name the state change before choosing the easing.** Every task below follows transitions.dev's sequence: identify what the owner is seeing change, pick the smallest matching transition family, then
   assign token values. No task adds motion because a surface "feels plain" without naming the specific change it marks.
2. **Tokens live in `packages/iroha-shared/src/theme/themes.css`**, next to the existing reduced-motion rule and `--control-target-min`. This is shared design-token territory per the theme asset
   boundary hard rule; `apps/iroha-web` consumes the tokens, it does not define new ones locally.
3. **No new runtime dependency.** CSS custom properties, CSS transitions/animations, and Svelte's built-in `transition:`/`{#key}` primitives are sufficient for every task here. No D3, no Three.js, no
   animation library — this plan intentionally stays inside what `v0.4.4` explicitly ruled out adding, per its Decision 4.
4. **Reduced motion is a state-preserving behavior, not a shorter duration.** Every new transition must have an explicit reduced-motion result that still communicates the state change — never just
   "the same animation, crushed to near-zero" as the sole mechanism. Audit each task's reduced-motion path as its own acceptance item.
5. **Frequent keyboard actions remain instant**, carried over from `v0.4.4`'s own constraint. Motion here targets state *arrival* (data, disclosure, language switch), not input handling latency.
6. **Grapher is the reference language.** New tokens and every shared-component slice must look right in Grapher first, then get an explicit all-six-language pass — matching `v0.4.4`'s own decision 1.
7. **Each dispatched task changes at most five source, test, or documentation files.** Same discipline `v0.4.4` documented and then repeatedly broke — split before exceeding, and land the split as a new
   numbered task, not a same-task "fix" commit re-opening a closed one.
8. **Verify motion in a real browser, both motion preferences, before calling a task done.** A passing type-check does not prove a CSS transition fires, blocks focus, or respects `prefers-reduced-motion`.
   Where a task uses a Svelte `transition:`/`{#key}` primitive rather than a plain CSS `transition:` property, verify the reduced-motion check actually gates the JS-driven duration too — CSS-only fixes
   (Task 2) do not automatically cover Svelte's own transition engine.

## Delivery plan

### Task 1: Record the motion inventory and propose the token set

Produce the `state change | owner | channels | timing | reduced-motion result | route` inventory the `v0.4.4` follow-up epic asked for, covering every duration and gap in the evidence table above,
plus anything this pass finds that the initial sweep missed. Group into named semantic families and propose token values — keep an existing well-tuned literal's value and just name it; don't invent new
timing values speculatively.

Proposed families (confirm/adjust during the inventory):

- `--motion-micro` (~120-160ms, ease-out): hover/press feedback, disclosure toggles (`NavigationMenu`'s existing chevron 160ms is the anchor value).
- `--motion-quick-state` (~180-250ms, ease): loading -> ready swaps, pill enter/exit, popover reveal.
- `--motion-data-update` (~500-650ms, ease-in-out): gauge/number/chart value changes (`RingGauge`'s existing 0.6s is the anchor value).
- `--motion-language-switch` (~250-350ms, ease-in-out): the Shell cross-fade in Task 6 — kept separate from `--motion-quick-state` because it's a full-composition swap, not a small UI element.

Acceptance:

- the inventory table covers every file/line in the evidence table above, including the six ECharts components (recorded as "out of scope, ECharts-internal" rather than silently dropped);
- each existing literal is mapped to a proposed token or explicitly marked chart-specific with a one-line reason;
- token names describe intent, not duration;
- proposed values do not change any existing chart/map animation's felt timing without a named reason.

Expected files (maximum 2): one dated inventory doc under `docs/`, this plan file if scope needs a recorded adjustment.

Verify: manual cross-check against a fresh `rg` pass; no code changes in this task.

### Task 2: Scope the reduced-motion kill switch

Fix the `!important` conflict identified in Review correction 1 before any other task ships motion. Narrow `themes.css`'s and `app.css`'s blanket rule so it still stops truly decorative/infinite loops
(`animation-iteration-count: 1 !important`, `scroll-behavior: auto !important` stay) but no longer forces every `transition-duration`/`animation-duration` to `0.01ms` — that decision moves to each
component, following the pattern `LoadingBoundary`'s shimmer already uses. Audit every existing animation/transition in the app for one that currently relies solely on the blanket crush with no local
override; add one before narrowing the rule, so nothing regresses into full-speed decorative motion under reduced-motion preference.

Acceptance:

- `transition-duration`/`animation-duration` are no longer forced to `0.01ms !important` app-wide;
- every animation/transition that previously relied on the blanket crush now has an explicit local `@media (prefers-reduced-motion: reduce)` override, verified by toggling the OS/browser preference and
  checking each one individually — not just a type-check;
- infinite/looping decorative animations (the loading shimmer, any other found during the audit) still stop under reduced motion;
- `app.css`'s duplicate rule is reconciled with the shared one (either delegate to the shared rule or keep an intentionally-scoped app-local copy with a comment explaining why).

Expected files (maximum 4): `packages/iroha-shared/src/theme/themes.css`, `apps/iroha-web/src/routes/app.css`, one component found to need a new local override during the audit, this plan if the audit
finds more than fits the budget.

Verify: browser check at `prefers-reduced-motion: reduce` across Today, Overview, and one route per remaining chart type; confirm no full-speed decorative animation appears.

### Task 3: Land the token set

Add the confirmed `--motion-*` custom properties to `packages/iroha-shared/src/theme/themes.css`. Do not consume them anywhere yet — this task only makes them available.

Acceptance:

- tokens are defined once, in `:root`, with a one-line comment per family naming the state changes it covers;
- `make theme-boundary-check` and `make web-check` stay clean;
- no visual change anywhere in the app (tokens are unused after this task).

Expected files (maximum 2): `packages/iroha-shared/src/theme/themes.css`, one focused test if the package has a token-presence test convention.

Verify: `make theme-boundary-check`, `make web-check`, visual no-op check (screenshot diff of one route before/after).

### Task 4: Give `LoadingBoundary` an arrival transition

The vertical slice the follow-up epic named directly, and the single most-seen state change in the app. Replace the hard `{#if}/{:else}` cut with a `--motion-quick-state`-timed transition: the loading
surface fades/settles out as the data surface fades/settles in, and the "Updating…" pill and first-load overlay get real enter/exit motion instead of appearing mid-frame.

Acceptance:

- first paint (loading -> ready) transitions using `--motion-quick-state`, not a snap-cut;
- the "Updating…" pill and loading overlay each have enter and exit motion;
- `aria-live`/`aria-busy`/`inert` behavior is unchanged — this task changes presentation only, not the async contract `AGENTS.md`'s loading-state hard rule protects;
- reduced motion: the state change is still communicated (pill appears/disappears immediately, never stuck mid-transition-invisible) — verified with Task 2 already landed, so this can use a real
  (non-crushed) instant swap rather than fighting the old blanket rule;
- verified live on Today (full-boundary swap) and Metrics (`preserveLayout` update-pill path).

Expected files (maximum 3): `apps/iroha-web/src/lib/components/LoadingBoundary.svelte`, one focused test, this plan if scope shifts.

Verify: `make web-test`, `make web-check`, browser check at normal and reduced motion on Today and Metrics.

### Task 5a: Name and reuse the data-update motion on the shared gauge components

Move `RingGauge.svelte`'s existing `0.6s ease` to `--motion-data-update` (same felt value, now named and shared), and apply the same token to `DesignRingGauge.svelte`. Do not add a second, different
animation — this task turns one good instance into a reusable pattern.

Acceptance:

- both shared gauge components use `--motion-data-update` for their value-change transition;
- the felt animation is unchanged from `RingGauge`'s current behavior (verify via before/after screenshot at the same value);
- reduced motion: the ring still reaches its correct end state, without the sweep.

Expected files (maximum 3): `RingGauge.svelte`, `DesignRingGauge.svelte`, one focused test if present.

Verify: `make web-test`, `make theme-boundary-check`, browser check at normal/reduced motion across the languages that use these two components.

### Task 5b: Bring the three inline per-language rings onto the same token

`field-journal/Today.svelte`, `phenology/Today.svelte`, and `phenology/MediaDetail.svelte` each draw a ring by hand instead of using `RingGauge`. Give each ring's `stroke-dasharray` transition the same
`--motion-data-update` token — `phenology/Today.svelte` already has a transition (`0.3s ease`) to retarget; the other two currently snap instantly and need one added. This is not a refactor to the shared
component (that's a larger, separate change, out of scope here) — it's making the existing per-language rings feel the same as the shared one, per `AGENTS.md`'s rule that whatever's shared across
`theme-ui/*/` copies needs one canonical value even when the markup itself stays independent.

Acceptance:

- all three inline rings use `--motion-data-update`;
- `field-journal/Today.svelte` and `phenology/MediaDetail.svelte` no longer snap instantly on value change;
- reduced motion: all three reach the correct end state without the sweep, same as Task 5a's components;
- explicitly note in the PR/changelog that consolidating these three onto the shared `RingGauge` component remains a separate, undispatched refactor.

Expected files (maximum 4): the three `.svelte` files above, one focused test if feasible.

Verify: `make web-test`, `make theme-boundary-check`, browser check at normal/reduced motion for Field Journal, Phenology's Today, and Phenology's media detail route.

### Task 6: Cross-fade the design-language switch

The most dramatic, currently-instant state change in the app. When `theme.language()` changes in `ThemeFrame.svelte`, wrap the `<Shell>` mount so the outgoing language's shell transitions out as the
incoming one transitions in, using `--motion-language-switch`. Verify first whether `Shell` (the `$derived` component reference) already unmounts/remounts on language change as-is, or whether an
explicit `{#key theme.language()}` block is needed to guarantee ordered enter/exit — implement whichever the real behavior requires, don't assume.

Acceptance:

- switching languages via `DesignLanguagePicker` visibly cross-fades rather than hard-cutting, in both directions;
- the transition uses `--motion-language-switch`, driven through Svelte's transition primitives (not a raw CSS `transition:` that Task 2's fix wouldn't reach) — confirm the reduced-motion check gates
  the JS-driven duration directly (e.g. read `prefers-reduced-motion` once and pass a zero duration), not just via the CSS rule;
- reduced motion: the switch is instant with no fade, but never shows a layout flash or double-rendered shell mid-swap;
- the canonical `main` landmark and skip-link target from `v0.4.4`'s Task 2 remain valid through the transition — verify focus/landmark behavior isn't disturbed by the wrapper;
- verified across all six languages, both directions of at least one switch pair.

Expected files (maximum 3): `apps/iroha-web/src/lib/themes/ThemeFrame.svelte`, one focused test if feasible given it's a cross-language visual behavior.

Verify: `make web-check`, `make theme-boundary-check`, browser check at normal/reduced motion switching through all six languages, keyboard-focus check per the landmark acceptance item above.

### Task 7: Reveal motion for the navigation popover

`.navigation-popover` (Domains/Analyze/More) currently appears with no transition; only its chevron icon rotates. Give the popover itself a `--motion-quick-state` reveal (opacity + a small
translate/scale settle) on open and close, composed with the existing chevron rotation rather than replacing it.

Acceptance:

- the popover has enter and exit motion, not just the chevron;
- positioning logic (`updatePopoverPosition`, viewport clamping) is unaffected — this task adds presentation only;
- reduced motion: popover opens/closes immediately, chevron rotation may keep its existing (already-fine) 160ms or move to `--motion-micro`;
- keyboard open/close (native `<details>`/`<summary>` toggle) is not slowed down — no added latency before the menu is interactive, per Decision 5.

Expected files (maximum 2): `apps/iroha-web/src/lib/components/NavigationMenu.svelte`, one focused test if feasible.

Verify: `make web-check`, browser check at normal/reduced motion, keyboard open/close timing check.

### Task 8: Animate Today's headline numbers on change

Grapher's `Today.svelte` formats every metric (Move kcal, Exercise min, Steps, sleep efficiency %) through a plain text binding (`number()`/`toLocaleString`, `grapher/Today.svelte:20-26`) with no
transition. Add a small shared `AnimatedNumber`-style primitive under `packages/iroha-shared` using Svelte's own `svelte/motion` (`tweened`) — already part of Svelte, not a new dependency per Decision
3 — so headline numbers count toward their new value instead of snapping. Wire it into Grapher's `Today.svelte` only; the other five languages' Today equivalents are explicitly out of this task's budget
(see below).

Acceptance:

- Grapher's headline numbers (Move, Exercise, Steps, sleep efficiency) animate toward the new value on day change and on refetch, using a `--motion-data-update`-scaled duration;
- the primitive lives under `packages/iroha-shared` (theme asset boundary), not `apps/iroha-web`;
- reduced motion: the value updates instantly to the correct final number, no count animation, verified by toggling the OS preference (this is a `svelte/motion` tween, so Task 2's CSS fix does not
  reach it — the reduced-motion check must gate the tween directly, same caution as Task 6);
- values that go from a number to `—` (no data) or back do not animate through nonsense intermediate numbers — jump directly for `null`/`Infinity` transitions, tween only number-to-number changes;
- the five-language extension is filed as the immediate next task, not silently expanded into this one.

Expected files (maximum 4): the new shared primitive, `grapher/Today.svelte`, one focused test.

Verify: `make web-test`, `make theme-boundary-check`, browser check at normal/reduced motion on Grapher's Today across a day change and a refetch.

### Task 9: Public/private parity and regression check

Re-run the inventory's affected surfaces against `apps/iroha-public-site` where shared components render publicly (`RingGauge`/`DesignRingGauge` if used there; `ThemeFrame`'s language switch does not
apply publicly if the public site pins one language — confirm rather than assume). Add a lightweight regression check that would fail if a `--motion-*` token is deleted while still referenced, matching
`v0.4.4`'s pattern of extending an existing audit script rather than standing up a second stack.

Acceptance:

- every shared component touched in Tasks 4-8 is checked against its public-site consumer, if any exists;
- a token-usage regression check exists and fails on a deliberately-broken token reference (verify by breaking one locally, observing red, then reverting);
- confirmed (not assumed) whether the public site has a language switch to check at all.

Expected files (maximum 3): the regression check location, one focused test, `docs/frontend-theme-architecture.md` only if the token contract needs a one-line mention.

Verify: `make public-site-check`, `make theme-boundary-check`, the new regression check's red/green cycle.

### Task 10: Release the bounded v0.4.5 patch

Bump `VERSION`, add the changelog entry, and prepare the PR. Same release discipline as `v0.4.4`'s Task 10 — but keep later commits inside their originating task's file budget rather than reopening a
"done" task past its declared scope; if a review finding requires more, dispatch it as a new numbered task instead of extending an old commit's task.

Acceptance:

- `make check` and `make validate` pass;
- `VERSION`/`CHANGELOG.md` agree on `0.4.5`;
- the changelog names the token set and each felt change (loading arrival, gauge data-update including the three per-language rings, Grapher's animated headline numbers, language-switch cross-fade,
  popover reveal) as user-visible changes;
- a fresh-context review confirms no task's final diff exceeds its declared file budget without a recorded split;
- the feature branch is pushed and a PR opened; no deployment or release publication.

Expected files (maximum 4): `VERSION`, `CHANGELOG.md`, the final review record.

Verify: `make check`, `make validate`, `make release-candidate`, fresh-context review.

## Follow-up (not v0.4.5 scope)

- **A real command-confirmation motion**, deferred per Review correction 2. `CommandPalette.svelte`'s `activate()` closes the palette and navigates in the same tick, with no success/failure branch —
  before this can get motion, someone has to decide what "confirmation" even means for a component whose every action is a navigation (confirmation on route arrival, composing with Task 4's arrival
  transition, is the leading candidate; a held checkmark on the dying palette is not, per Decision 5).
- Consolidating `field-journal/Today.svelte`, `phenology/Today.svelte`, and `phenology/MediaDetail.svelte`'s inline rings onto the shared `RingGauge` component — a real refactor, not a motion task; Task
  5b intentionally left the duplication in place and only unified the transition.
- Extending Task 8's `AnimatedNumber` primitive to the other five languages' Today equivalents — the primitive and the pattern exist after this plan; wiring it into Atlas, Archive, Field Journal,
  Phenology, and Sound Map's own Today compositions is the immediate next task, dispatched separately so it gets its own file budget instead of inflating Task 8's.
- `RoutesMap`/`RouteFootprint` camera-movement semantics (arrive/reveal framing) — genuinely different problem shape (spatial, not state), deserves its own plan.
- The ThreeUI-style typed specimen catalog (`v0.4.4`'s Follow-up epic 3) — still explicitly out of scope; nothing here is a specimen platform, just applied tokens.

## Risks and mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| The `!important` reduced-motion bug blocks every other task's reduced-motion acceptance until fixed. | High | Task 2 is dispatched second, immediately after the inventory, before any consuming task lands. |
| Motion added for its own sake instead of a named state change. | High | Decision 1 and each task's acceptance criteria require naming the state change first; Task 1's inventory is a gate before any token is defined. |
| Reduced-motion users lose state feedback entirely instead of getting an instant equivalent. | High | Decision 4 and Task 2's per-component audit make reduced-motion an explicit, individually-verified acceptance item. |
| Six languages multiply every visual change. | Medium | Grapher first, then explicit all-language pass per Decision 6. |
| Task-budget discipline repeats `v0.4.4`'s own violation. | Medium | Decision 7 names the failure mode explicitly and requires a new task number instead of extending an old one. |
| The language-switch transition (Task 6) hides a landmark/focus regression during the swap. | Medium | Task 6's acceptance explicitly re-checks `v0.4.4`'s skip-link/main-landmark contract, not just the visual fade. |
| A tweened number animates through a nonsensical value on a `null`/`—` transition (Task 8). | Medium | Task 8's acceptance explicitly requires jumping (not tweening) across `null`/`Infinity` boundaries. |

## Definition of done

`v0.4.5` is complete only when:

- a named `--motion-*` token set exists in `packages/iroha-shared/src/theme/themes.css`, and the global reduced-motion rule no longer contradicts Decision 4;
- every duration touched by this plan traces to a named token;
- `LoadingBoundary`'s loading -> ready transition and update-pill enter/exit are no longer a hard cut;
- all gauge instances — shared components and the three per-language inline rings — share one named data-update motion;
- switching design languages visibly cross-fades instead of hard-cutting, with the `v0.4.4` landmark/skip-link contract verified intact through the transition;
- the navigation popover has real reveal motion, not just its chevron;
- Grapher's Today headline numbers animate toward their new value instead of snapping;
- every new transition has a verified, state-preserving reduced-motion result, checked by toggling the OS preference in a real browser;
- `make check`, `make validate`, `make release-candidate`, and a fresh-context review pass;
- `VERSION` and changelog identify `v0.4.5`;
- the feature branch is pushed and a GitHub PR is opened, with no deployment or release publication.

## Rollback

Tasks land as small conventional commits, same as `v0.4.4`. The token set (Task 3) is additive and inert until consumed; Task 2's reduced-motion fix is independently revertible (falls back to the old
blanket crush); each vertical slice (Tasks 4, 5a, 5b, 6, 7, 8) is independently revertible without touching the others. There is no database, API, or public/private boundary change, so rollback is
source-only.
