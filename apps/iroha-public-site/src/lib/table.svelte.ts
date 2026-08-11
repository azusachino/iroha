// A thin Svelte 5 (runes) reactive wrapper around @tanstack/table-core.
//
// @tanstack/svelte-table isn't usable here: v8 doesn't support Svelte 5 (this
// project forces runes mode), and the v9 adapter with native Svelte 5 support
// is alpha and explicitly documented upstream as changing too frequently to
// depend on. table-core itself is the stable, framework-agnostic half of
// TanStack Table -- this file is the same "wire it to runes directly"
// approach used by the community Svelte 5 references for table-core.
import {
  createTable,
  type RowData,
  type TableOptions,
  type TableOptionsResolved,
} from "@tanstack/table-core";

function mergeObjects(...objects: object[]): object {
  return Object.assign({}, ...objects);
}

export function createSvelteTable<TData extends RowData>(
  options: TableOptions<TData>,
) {
  const resolvedOptions = mergeObjects(
    {
      state: {},
      onStateChange: () => {},
      renderFallbackValue: null,
    },
    options,
  ) as TableOptionsResolved<TData>;

  const table = createTable(resolvedOptions);
  let state = $state(table.initialState);

  function updateOptions() {
    table.setOptions(
      (previous) =>
        mergeObjects(previous, options, {
          state: mergeObjects(state, options.state ?? {}),
          onStateChange: (updater: unknown) => {
            state =
              typeof updater === "function"
                ? (updater as (old: typeof state) => typeof state)(state)
                : (updater as typeof state);
            options.onStateChange?.(updater as never);
          },
        }) as TableOptionsResolved<TData>,
    );
  }

  updateOptions();
  $effect.pre(() => {
    for (const value of Object.values(options.state ?? {})) value;
    updateOptions();
  });

  return table;
}
