# WeRead → BookOrbit companion feasibility

Date: 2026-10-12. Parent: [#142](https://github.com/azusachino/iroha/issues/142); related contract spike: [#143](https://github.com/azusachino/iroha/issues/143).

## Question, scope and verdict

Can a one-way WeRead companion populate BookOrbit sufficiently that Iroha needs only a BookOrbit connector, without losing the personal history/provenance promised in #142?

The owner approved a bounded, single-session investigation and **disposable local BookOrbit tests with synthetic WeRead fixtures**. No real WeRead credentials/requests, persistent deployment, migration, production companion, upstream patch, or Iroha application change was authorized or performed. Work used the workstation-task playbook directly, without Asobi or agents. Existing preparation was merged in [PR #147](https://github.com/azusachino/iroha/pull/147); this report adds evidence rather than accepting the new topology automatically.

**Verdict: NO-GO for lossless history through unmodified BookOrbit 3.3.0's tested ordinary application APIs. Conditional feasibility for a deliberately limited status mirror of already-matched local books.**

A companion can write metadata, status/date facts and percent for a book backed by a local file. That does not make BookOrbit a faithful store of WeRead source history. Native provenance, safe stale-write arbitration, fileless shelf entries and unlocated annotations remain blockers. Generic custom metadata can hold JSON but is shared book metadata, not private per-reader evidence.

This does not reject BookOrbit as a reader. Its cross-device reliability, actual WeRead gateway behavior, quotas/service terms and backup restoration remain unverified here. A source-specific upstream/import API might change this verdict; none was implemented.

## Executed environment and pins

| Input | Actual test value |
| --- | --- |
| BookOrbit source | Release `v3.3.0`, commit `e767358443419253eeed782518d6fc4f675d69fd` |
| App image tag | `ghcr.io/bookorbit/bookorbit:3.3.0` |
| App platform manifest digest | `sha256:7f92fca7d924c0c82f902ec87658ec96e367405b32da2f996ce35f8206e0b23f` |
| App image ID | `f840c217268c51d0bb522258ac8eb7085d70e7abc150bfbd7060595258c0bf1f` |
| DB image | `docker.io/pgvector/pgvector:pg18`, platform manifest `sha256:ad249f9fb9572979643a47694e59b9a8cdcdf6abc6accebf49bb80ad5f20ff9b` |
| Runtime | Podman 5.8.7; Python standard-library HTTP/ZIP/subprocess probe through `uv run --no-project` |
| Network | Dedicated internal container network; app published only on loopback port 18943; DB port not published |
| Storage/data | New per-run scratch directories, separate DB; one authored synthetic DRM-free EPUB; synthetic admin and library-viewer accounts |
| Observed version | `/api/v1/app-info` returned `v3.3.0`; optional Swagger JSON captured successfully |

The source tag and running app report the same release. The images had no source-revision labels; this is version/digest recording, not an attested exact-source build claim. Both index and platform digests are retained in the raw result. No API key, JWT, password or account reset link is retained in the published evidence.

The sandbox used root inside a rootless container (`PUID/PGID=0`) for scratch bind-mount permissions, without host privileges. It was not a hardened Compose deployment, restore test, or recommendation for production configuration. The source contains the same relevant DTOs as the earlier inspected `main`; results below belong specifically to the tested release image.

## Measured results

The probe completed with exit 0 after recording all 12 checks. Six positive capabilities were observed and six required capabilities were absent in the tested path. A successful probe execution is **not** a successful companion acceptance gate.

| Capability | Actual observation | Consequence |
| --- | --- | --- |
| API inventory | Swagger JSON returned 200; `/api/v1/books` lists DELETE, not a standalone create operation | No documented create route was established for fileless WeRead books. |
| Metadata-only create | `POST /api/v1/books` returned 404, `Cannot POST /api/v1/books` | Cannot treat a WeRead shelf entry as a normal new BookOrbit book through this route. This one request is not proof that every possible migration path is impossible. |
| Local book creation | Library scanner ingested the authored EPUB; one book with a primary file was returned | A local file provides the working baseline. Do not create fake book files as a production workaround. |
| Metadata write | PATCH existing book title/authors/ISBN returned 200 and round-tripped | Metadata mirroring to an already-matched local book is feasible. |
| Status/date write | PATCH reading status and `startedAt: 2024-01-02` returned 200; date survived | Returned source was `manual`, with current BookOrbit `updatedAt`, not WeRead provenance. |
| Percent write | File progress POST returned 201 and 20% | Basic percent mirroring works on the selected local file; does not preserve WeRead chapter/range identity. |
| Native source provenance | Additional `sourceProvider`, `sourceRecordId`, `sourceUpdatedAt` fields returned 400 with “should not exist” | There is no typed external-source slot in this tested progress write DTO. |
| Replay semantics | Replaying 20% changed `updatedAt` from `01:14:29.245Z` to `01:14:29.377Z` | Same visible percent is not timestamp-idempotent; ingestion time must not become reading time. |
| Stale/local conflict | Wrote 70%, then a later-arriving 10%; GET returned 10% | This API does not arbitrate the caller's stale source timestamp. It accepts deliberate rewinds too, so a blanket max-percent rule would be wrong. No truly concurrent race was executed. |
| Unlocated annotation | Text/note without CFI/PDF returned 400, `Provide exactly one of cfi or pdf` | WeRead chapter/range highlights cannot be copied to the ordinary annotation API without a verified locator translation or an upstream unlocated-source contract. |
| Generic evidence storage | A text custom field round-tripped a synthetic JSON evidence string; a title-only PATCH retained it | A custom field can carry opaque current JSON, but this did not establish append-only history, replay/deletion semantics or a native source model. |
| Per-reader privacy | Second ordinary library viewer saw the first account's custom evidence while its own `readStatus` was null | Book metadata is shared across library users. Do not put personal progress, account identity, source history or notes in these fields. This is normal metadata visibility, not a demonstrated BookOrbit vulnerability. |

Writes were tested with the sandbox superuser, not a proven least-privileged companion account. Metadata/field edits affect shared books and require the corresponding permissions; viewer testing exercised reads only. Credential scoping remains an implementation prerequisite.

No WeRead text extraction, DRM transfer, fake historical session creation or fabricated EPUB locator was used. The no-location annotation test intentionally checked the boundary rather than guessing a cross-edition CFI.

## Evidence and reproducibility

- [Executed probe](evidence/2026-10-12-weread-bookorbit-probe.py): throwaway standard-library test code, not an operational companion. SHA-256 `60dac26edfe8ad00853f351e5bc8c4d7f4f07c71a44e54795e96eb228f37c34e`.
- [Complete synthetic results](evidence/2026-10-12-weread-bookorbit-results.json): includes statuses, errors, timestamps, image identity and teardown exits. Original capture SHA-256 `6a7eb01c22256a09d5b862f5e1c8490913dca1c16e3be83e9b845c3356b3171c` before any documentation JSON formatting.
- Swagger capture SHA-256 `876731e541b31b67c1540156af8105a12828ac3dba807847391663d662685492`; the large generated schema is scratch, not the sole record of relevant API claims. Its selected book routes are preserved in the result, and source references below identify their semantics.

For an owner-authorized repeat from the workstation root, copy the probe into `.tmp/bookorbit-companion/spike.py` so all generated state stays in scratch. Obtain `source/` over SSH at the recorded release commit and load the recorded image digests under the two tags used by the probe. Verify `podman image inspect` platform digests before execution; moving tags are not a reproducible pin. Run:

```sh
uv run --no-project python .tmp/bookorbit-companion/spike.py
```

The probe refuses existing containers/network names and a busy loopback port; it creates fresh storage, scans the synthetic EPUB, calls the APIs, creates a second viewer and reports capabilities. It always attempts teardown. Review `completed`, each capability and every cleanup exit in `results.json`; exit 0 alone is not GO. It has no real WeRead network client. Do not run the evidence file in place under `docs/`, because it deliberately uses its containing directory as the scratch root.

### Failed attempts retained honestly

1. Initial pull used `:v3.3.0` and failed `manifest unknown`. Upstream release notes/workflow specify `:3.3.0`; that source-backed tag was used, never `latest`.
2. First API attempt ran the early checks but stopped on a harness assertion that expected 200/201 for password reset. Source declares 204. The harness was corrected, the first run's containers/network/storage removed, and the entire test repeated with fresh storage. Final results are not a mixture of old and new sessions.

The final run's app, DB, network and disposable storage cleanup each exited 0. A subsequent container listing was empty; only the pre-existing default Podman network remained. Pulled images remain cached; no global prune or unrelated service cleanup occurred. Existing Iroha data was never mounted or addressed.

## Why an external ledger changes the architecture

A companion could maintain its own mappings, native payloads, account-scoped history, last-exported values and conflict decisions. That can make a status mirror safer, but if Iroha reads only BookOrbit's ordinary current-state APIs, the companion's evidence remains invisible to Iroha.

Writing that ledger into generic shared book fields fails the demonstrated privacy boundary and still needs versioning, retention, pagination and replay rules. Directly editing BookOrbit SQL or masquerading WeRead imports as web/KOReader activity bypasses application contracts and misattributes evidence. Neither is the smallest safe solution.

For Iroha's original goal, choose one of these explicitly:

1. **Independent source connectors:** BookOrbit for its own reading, official WeRead for its own sourced state/history. This remains the evidence-preserving default recommendation; account testing and implementation are not done.
2. **Limited one-way mirror:** only already-matched local books, explicitly lossy status/progress, a conservative companion ledger/attention policy, no sessions or annotations and no claim that old WeRead history reaches Iroha. Requires the owner to accept reduced goals.
3. **BookOrbit source-import extension:** a supported user-scoped external evidence/import surface, fileless items, source IDs/times/revisions, idempotency and stale/conflict rules, unlocated annotations and measured-versus-aggregate duration. Iroha could then have one BookOrbit connector while retaining original WeRead provenance. This is substantial upstream/contract work, not a small deployment companion.

Do not deploy the companion topology yet. Keep reader evaluation independent and #143 open: this test answers the write/provenance seam, not cross-device sync, WeRead account access, backups, or all of the spike's acceptance criteria. No changes to #144–#146 implementation scope are implied until the owner chooses an architecture.

## Source references at the tested release

All BookOrbit links below are pinned to `e767358443419253eeed782518d6fc4f675d69fd`:

- [Release image publication workflow](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/.github/workflows/release.yml): semantic version image tag versus `APP_VERSION=v…`.
- [Book controller](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/modules/book/book.controller.ts) and [book service](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/modules/book/book.service.ts): query/read routes, file-scoped progress, current server time, metadata/status writes.
- [Progress DTO](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/modules/book/dto/save-progress.dto.ts) and [status DTO](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/modules/user-book-status/dto/set-status.dto.ts): supported fields; no generic WeRead source write contract.
- [Annotation create DTO](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/modules/annotation/dto/create-annotation.dto.ts): exactly one locator; no external source ID/time inputs. [Reader schema](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/db/schema/reader.ts) constrains annotation origin to web/KOReader/Kobo.
- [Custom-metadata schema](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/db/schema/custom-metadata.ts): book-level values, no reader-user ownership. [Metadata update DTO](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/modules/book/dto/update-book-metadata.dto.ts) permits generic custom values.
- [Migration source DTO](https://github.com/bookorbit/bookorbit/tree/e767358443419253eeed782518d6fc4f675d69fd/server/src/modules/migration/dto) and adapters: existing Booklore/Grimmory/Audiobookshelf/Calibre-Web-Automated workflows are not a demonstrated official WeRead import contract. They were inspected, not executed or ruled out for every possible extension.
- [Auth controller](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/modules/auth/auth.controller.ts): reset-password success is 204.

The prior [BookOrbit/official WeRead research](bookorbit-research.md) pins the Tencent gateway contract, its 0–100 percent/seconds semantics and unresolved hosted-service terms. It is still source evidence only for WeRead: synthetic BookOrbit tests do not validate a real WeRead account or unrestricted automation.
