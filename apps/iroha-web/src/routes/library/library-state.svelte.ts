import { onMount } from "svelte";
import {
  getMediaAggregates,
  listMedia,
  type MediaAggregates,
  type MediaCompletionBucket,
  type MediaRow,
} from "$lib/api";
import { progressPercent } from "$lib/format";
import { mediaTypeFamily } from "$lib/media";
import { createAsyncResource } from "$lib/asyncResource.svelte";

// All state, derivations, and data loading for the Library route, kept out
// of the .svelte file so the template isn't interleaved with ~240 lines of
// business logic. `theme` (a Svelte context lookup) stays in the component.
export function createLibraryState() {
  const libraryResource = createAsyncResource<{
    aggregates: MediaAggregates;
    items: MediaRow[];
    nextCursor: string | null;
    hasMore: boolean;
    statusCounts: Record<string, number>;
    activeCount: number;
  }>();
  const aggregates = $derived(libraryResource.data?.aggregates ?? null);
  const items = $derived(libraryResource.data?.items ?? []);
  const hasMore = $derived(libraryResource.data?.hasMore ?? false);
  const statusCounts = $derived(libraryResource.data?.statusCounts ?? {});
  const activeCount = $derived(libraryResource.data?.activeCount ?? 0);
  let loadingMore = $state(false);
  let family = $state("");
  let status = $state("");
  let completedYear = $state("");
  let selectedYear = $state("");
  let yearSelect = $state<HTMLSelectElement>();
  let availableYears = $state<MediaCompletionBucket[]>([]);
  const EMPTY_AGGREGATES: MediaAggregates = {
    totals: {
      item_count: 0,
      completed_count: 0,
      current_completed_count: 0,
      this_year_completed: 0,
      average_rating: 0,
    },
    completions_by_year: [],
    score_distribution: [],
    type_split: [],
  };
  const aggregatesForView = $derived(aggregates ?? EMPTY_AGGREGATES);

  const FAMILIES = [
    { value: "", label: "All" },
    { value: "anime", label: "Anime" },
    { value: "manga_book", label: "Manga & light novels" },
    { value: "book", label: "Books" },
    { value: "game", label: "Games" },
  ];

  // Only actively-in-progress items belong in the "continue" strip; paused /
  // on-hold entries keep status=in_progress but carry hidden_from_continue.
  const isContinuing = (item: MediaRow) =>
    item.status === "in_progress" && !item.hidden_from_continue;
  const continueItems = $derived(items.filter(isContinuing).slice(0, 6));

  const STATUS_ORDER = [
    "paused",
    "completed",
    "planned",
    "abandoned",
    "unknown",
  ];
  const groupedItems = $derived(
    Object.entries(
      items
        .filter((item) => !isContinuing(item))
        .reduce(
          (groups, item) => {
            // Paused items share the in_progress status; give them a shelf.
            const key =
              item.status === "in_progress"
                ? "paused"
                : item.status || "unknown";
            (groups[key] ??= []).push(item);
            return groups;
          },
          {} as Record<string, MediaRow[]>,
        ),
    ).sort(
      ([a], [b]) =>
        (STATUS_ORDER.indexOf(a) + 1 || 99) -
        (STATUS_ORDER.indexOf(b) + 1 || 99),
    ),
  );

  // The API splits by raw media_type (anime_season, manga, movie, ova…);
  // collapse those into display families for the "By kind" chart.
  const typeFamilies = $derived(
    Object.entries(
      (aggregates?.type_split ?? []).reduce(
        (families, bucket) => {
          const key = mediaTypeFamily(bucket.type);
          families[key] = (families[key] ?? 0) + bucket.count;
          return families;
        },
        {} as Record<string, number>,
      ),
    )
      .map(([type, count]) => ({ type, count }))
      .sort((a, b) => b.count - a.count),
  );

  const completions = $derived(aggregates?.completions_by_year ?? []);
  const scores = $derived(aggregates?.score_distribution ?? []);
  const yearOptions = $derived(
    [...availableYears].sort((a, b) => b.year - a.year),
  );

  $effect(() => {
    selectedYear = completedYear;
    if (yearSelect && yearSelect.value !== completedYear) {
      yearSelect.value = completedYear;
    }
  });

  onMount(() => {
    void load();
  });

  function currentFilters() {
    return {
      family: family || undefined,
      status: status || undefined,
      completed_year: completedYear ? Number(completedYear) : undefined,
    };
  }

  async function load() {
    const filters = currentFilters();
    const result = await libraryResource.run(async () => {
      const [nextAggregates, page] = await Promise.all([
        getMediaAggregates(filters),
        listMedia({ limit: 100, ...filters }),
      ]);
      return {
        aggregates: nextAggregates,
        items: page.items,
        nextCursor: page.next_cursor,
        hasMore: page.has_more,
        statusCounts: page.status_counts ?? {},
        activeCount: page.active_count ?? 0,
      };
    });
    if (result && !filters.completed_year) {
      availableYears = result.aggregates.completions_by_year ?? [];
    }
  }

  async function selectFamily(value: string) {
    if (value === family) return;
    family = value;
    await load();
  }

  async function selectStatus(value: string) {
    if (value === status) return;
    status = value;
    await load();
  }

  async function selectYear(value: string) {
    if (value === completedYear) return;
    completedYear = value;
    await load();
  }

  async function loadMore() {
    const cursor = libraryResource.data?.nextCursor;
    if (!cursor || loadingMore) return;
    loadingMore = true;
    try {
      const page = await listMedia({
        limit: 100,
        cursor,
        ...currentFilters(),
      });
      libraryResource.mutate((current) => ({
        aggregates: current?.aggregates ?? EMPTY_AGGREGATES,
        statusCounts: current?.statusCounts ?? {},
        activeCount: current?.activeCount ?? 0,
        items: [...(current?.items ?? []), ...page.items],
        nextCursor: page.next_cursor,
        hasMore: page.has_more,
      }));
    } catch {
      // Load-more failures are retry-safe -- keep the rows already showing
      // rather than replacing a working view with an error.
    } finally {
      loadingMore = false;
    }
  }

  function statusLabel(status: string): string {
    return status
      .replaceAll("_", " ")
      .replace(/^./, (char) => char.toUpperCase());
  }
  function statusTone(status: string): string {
    if (status === "completed") return "completed";
    if (status === "planned") return "planned";
    if (status === "abandoned") return "abandoned";
    if (status === "paused") return "paused";
    return "unknown";
  }

  function progressValue(item: MediaRow): number {
    return progressPercent(
      item.status,
      item.position,
      item.total,
      item.progress_percent,
    );
  }

  // Default to the native (Japanese) title; keep the English/romaji as a
  // secondary line when it differs.
  function primaryTitle(item: MediaRow): string {
    return item.native_title || item.title;
  }
  function altTitle(item: MediaRow): string {
    return item.native_title && item.native_title !== item.title
      ? item.title
      : "";
  }
  function initial(item: MediaRow): string {
    return primaryTitle(item).slice(0, 1);
  }

  return {
    libraryResource,
    get aggregates() {
      return aggregates;
    },
    get items() {
      return items;
    },
    get hasMore() {
      return hasMore;
    },
    get statusCounts() {
      return statusCounts;
    },
    get activeCount() {
      return activeCount;
    },
    get loadingMore() {
      return loadingMore;
    },
    get family() {
      return family;
    },
    get status() {
      return status;
    },
    get completedYear() {
      return completedYear;
    },
    get selectedYear() {
      return selectedYear;
    },
    set selectedYear(value: string) {
      selectedYear = value;
    },
    get yearSelect() {
      return yearSelect;
    },
    set yearSelect(value: HTMLSelectElement | undefined) {
      yearSelect = value;
    },
    get aggregatesForView() {
      return aggregatesForView;
    },
    FAMILIES,
    get continueItems() {
      return continueItems;
    },
    get groupedItems() {
      return groupedItems;
    },
    get typeFamilies() {
      return typeFamilies;
    },
    get completions() {
      return completions;
    },
    get scores() {
      return scores;
    },
    get yearOptions() {
      return yearOptions;
    },
    selectFamily,
    selectStatus,
    selectYear,
    loadMore,
    statusLabel,
    statusTone,
    progressValue,
    primaryTitle,
    altTitle,
    initial,
  };
}
