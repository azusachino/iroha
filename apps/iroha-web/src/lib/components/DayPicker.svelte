<script lang="ts">
  import { Calendar } from "bits-ui";
  import { parseDate, type DateValue } from "@internationalized/date";

  // A compact month-grid day picker built on bits-ui Calendar. Days that
  // actually have data are dotted so scrubbing lands on real days. Emits the
  // chosen 'YYYY-MM-DD'.
  let {
    value,
    days,
    max,
    onselect,
  }: {
    value: string;
    days: Set<string>;
    max?: string;
    onselect: (day: string) => void;
  } = $props();

  function safeParse(iso: string | undefined): DateValue | undefined {
    if (!iso || !/^\d{4}-\d{2}-\d{2}$/.test(iso)) return undefined;
    try {
      return parseDate(iso);
    } catch {
      return undefined;
    }
  }

  const parsedValue = $derived(safeParse(value));
  const parsedMax = $derived(safeParse(max));

  function handleValueChange(next: DateValue | undefined) {
    if (!next) return;
    onselect(next.toString());
  }
</script>

<Calendar.Root
  type="single"
  class="picker"
  value={parsedValue}
  maxValue={parsedMax}
  preventDeselect
  weekdayFormat="narrow"
  calendarLabel="Choose a day"
  onValueChange={handleValueChange}
>
  {#snippet children({ months, weekdays })}
    <Calendar.Header class="pk-head">
      <Calendar.PrevButton class="pk-nav" aria-label="Previous month"
        >‹</Calendar.PrevButton
      >
      <Calendar.Heading class="pk-heading" />
      <Calendar.NextButton class="pk-nav" aria-label="Next month"
        >›</Calendar.NextButton
      >
    </Calendar.Header>
    {#each months as month (month.value.toString())}
      <Calendar.Grid class="pk-grid">
        <Calendar.GridHead>
          <Calendar.GridRow class="pk-row">
            {#each weekdays as day, index (`${day}-${index}`)}
              <Calendar.HeadCell class="dow">{day}</Calendar.HeadCell>
            {/each}
          </Calendar.GridRow>
        </Calendar.GridHead>
        <Calendar.GridBody>
          {#each month.weeks as weekDates (weekDates[0].toString())}
            <Calendar.GridRow class="pk-row">
              {#each weekDates as date (date.toString())}
                {@const iso = date.toString()}
                <Calendar.Cell {date} month={month.value} class="pk-cell">
                  <Calendar.Day
                    class={`day${days.has(iso) ? " has" : ""}`}
                    data-day={iso}
                  >
                    {#snippet child({ props })}
                      <button type="button" {...props}>{date.day}</button>
                    {/snippet}
                  </Calendar.Day>
                </Calendar.Cell>
              {/each}
            </Calendar.GridRow>
          {/each}
        </Calendar.GridBody>
      </Calendar.Grid>
    {/each}
  {/snippet}
</Calendar.Root>

<style>
  :global(.picker) {
    width: 15rem;
    padding: 0.75rem;
  }

  :global(.picker .pk-head) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.5rem;
    font-weight: 650;
    color: var(--text);
  }

  :global(.picker .pk-nav) {
    appearance: none;
    border: 0;
    background: transparent;
    color: var(--text-muted);
    font-size: 1.1rem;
    width: 1.8rem;
    height: 1.8rem;
    border-radius: var(--radius);
    cursor: pointer;
    display: inline-grid;
    place-items: center;
  }

  :global(.picker .pk-nav:hover:not(:disabled)) {
    color: var(--accent);
    background: color-mix(in srgb, var(--accent) 12%, transparent);
  }

  :global(.picker .pk-grid) {
    width: 100%;
    border-collapse: collapse;
    table-layout: fixed;
  }

  :global(.picker .dow) {
    text-align: center;
    font-size: 0.68rem;
    font-weight: 500;
    color: var(--text-muted);
    padding-bottom: 0.25rem;
  }

  :global(.picker .pk-cell) {
    padding: 1px;
    text-align: center;
  }

  :global(.picker .day) {
    display: grid;
    place-items: center;
    width: 100%;
    aspect-ratio: 1;
    border: 0;
    border-radius: var(--radius);
    background: transparent;
    color: var(--text);
    font-size: 0.8rem;
    cursor: pointer;
    position: relative;
  }

  :global(.picker .day:hover:not([data-disabled]):not([data-outside-month])) {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }

  :global(.picker .day[data-outside-month]) {
    opacity: 0.28;
  }

  :global(.picker .day[data-disabled]) {
    color: color-mix(in srgb, var(--text-muted) 45%, transparent);
    cursor: default;
  }

  /* dot marks days that actually have data */
  :global(.picker .day.has::after) {
    content: "";
    position: absolute;
    bottom: 3px;
    left: 50%;
    transform: translateX(-50%);
    width: 4px;
    height: 4px;
    border-radius: 50%;
    background: var(--accent);
  }

  :global(.picker .day[data-selected]) {
    background: var(--accent);
    color: #08131a;
    font-weight: 700;
    box-shadow: var(--accent-glow);
  }

  :global(.picker .day[data-selected].has::after) {
    background: #08131a;
  }
</style>
