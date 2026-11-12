# Drifter feed contract copies

`reading-snapshot-v1.schema.json` and `reading-snapshot-v1.golden.json` are
copies of `schemas/reading-snapshot-v1.schema.json` and
`schemas/testdata/reading-snapshot-v1.golden.json` from drifter at commit
`d25815c` (azusachino/drifter PR 35). Tests read these copies, never the
drifter checkout. Refresh both files together when drifter changes the contract.
