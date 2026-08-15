// A page loads one or more independent pieces of server data and needs to
// show a boundary-level loading/error/ready state without re-deriving that
// bookkeeping (and its race-condition guards) by hand on every route. This is
// the one shared definition every route's data loading should go through.
export type AsyncResource<T> = {
  readonly data: T | null;
  readonly loading: boolean;
  readonly error: string | null;
  // True once this resource has loaded successfully at least once, and
  // stays true for the life of the resource -- a later refetch (e.g. a
  // filter change) must not make already-visible content disappear again.
  readonly ready: boolean;
  run(fetcher: () => Promise<T>): Promise<T | undefined>;
  // Update `data` directly without going through a fetch -- e.g. appending
  // a "load more" page to an already-loaded list.
  mutate(updater: (current: T | null) => T): void;
};

export function createAsyncResource<T>(): AsyncResource<T> {
  let data = $state<T | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let ready = $state(false);
  let requestId = 0;

  async function run(fetcher: () => Promise<T>): Promise<T | undefined> {
    const id = ++requestId;
    loading = true;
    error = null;
    try {
      const result = await fetcher();
      if (id !== requestId) return undefined;
      data = result;
      ready = true;
      return result;
    } catch (cause) {
      if (id !== requestId) return undefined;
      error = cause instanceof Error ? cause.message : String(cause);
      return undefined;
    } finally {
      if (id === requestId) loading = false;
    }
  }

  function mutate(updater: (current: T | null) => T): void {
    data = updater(data);
  }

  return {
    get data() {
      return data;
    },
    get loading() {
      return loading;
    },
    get error() {
      return error;
    },
    get ready() {
      return ready;
    },
    run,
    mutate,
  };
}
