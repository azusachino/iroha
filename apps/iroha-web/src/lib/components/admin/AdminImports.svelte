<script lang="ts">
  import { onMount } from "svelte";
  import { RefreshCw, Upload } from "@lucide/svelte";
  import {
    UPLOAD_SOURCE_KINDS,
    createImportJob,
    listImportJobs,
    listRawFiles,
    uploadRawFile,
    type ImportJob,
    type RawFile,
  } from "$lib/api";
  import { formatDate } from "$lib/format";
  import { formatBytes } from "$lib/format-bytes";

  let imports = $state<ImportJob[]>([]);
  let files = $state<Map<string, RawFile>>(new Map());
  let loading = $state(true);
  let busy = $state(false);
  let error = $state<string | null>(null);
  let notice = $state<string | null>(null);
  let sourceKind = $state<string>(UPLOAD_SOURCE_KINDS[0].kind);
  let fileInput: HTMLInputElement;

  function tone(status: ImportJob["status"]): string {
    return status === "completed" ? "good" : status === "failed" ? "bad" : "";
  }

  async function load() {
    loading = true;
    error = null;
    try {
      const [jobs, raw] = await Promise.all([listImportJobs(), listRawFiles()]);
      imports = jobs;
      files = new Map(raw.map((file) => [file.id, file]));
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      loading = false;
    }
  }

  async function upload(event: SubmitEvent) {
    event.preventDefault();
    const file = fileInput?.files?.[0];
    if (!file) return;
    busy = true;
    error = null;
    notice = null;
    try {
      const raw = await uploadRawFile(file, sourceKind);
      await createImportJob(raw.id, raw.source_kind);
      notice = raw.duplicate
        ? `“${file.name}” was already stored; imported it again.`
        : `Uploaded “${file.name}” and queued its import.`;
      fileInput.value = "";
      await load();
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      busy = false;
    }
  }

  async function reprocess(job: ImportJob) {
    busy = true;
    error = null;
    notice = null;
    try {
      await createImportJob(job.raw_file_id, job.parser_kind);
      notice = "Queued a new import of the same file.";
      await load();
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      busy = false;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<div class="stack">
  <section class="admin-card" aria-labelledby="upload-title">
    <header>
      <div>
        <h2 id="upload-title">Upload</h2>
        <small
          >Stores the file as raw evidence, then imports it. Re-uploading the
          same file is detected.</small
        >
      </div>
    </header>
    <form class="upload" onsubmit={upload}>
      <label>
        Kind
        <select bind:value={sourceKind}>
          {#each UPLOAD_SOURCE_KINDS as option (option.kind)}
            <option value={option.kind}>{option.label}</option>
          {/each}
        </select>
      </label>
      <label>
        File
        <input bind:this={fileInput} type="file" required />
      </label>
      <button type="submit" class="admin-button primary" disabled={busy}>
        <Upload size={14} />
        {busy ? "Uploading…" : "Upload and import"}
      </button>
    </form>
    {#if notice}<p class="admin-muted" role="status">{notice}</p>{/if}
    {#if error}<p class="admin-error" role="alert">{error}</p>{/if}
  </section>

  <section class="admin-card" aria-labelledby="imports-title">
    <header>
      <div>
        <h2 id="imports-title">Import history</h2>
        <small>The latest 50 imports, newest first.</small>
      </div>
      <button
        type="button"
        class="admin-button"
        onclick={() => void load()}
        disabled={loading}><RefreshCw size={14} /> Refresh</button
      >
    </header>
    {#if imports.length}
      <ul class="admin-list">
        {#each imports as job (job.id)}
          {@const file = files.get(job.raw_file_id)}
          <li>
            <div>
              <strong>{file?.original_filename ?? job.raw_file_id}</strong>
              <span class={`status ${tone(job.status)}`}>{job.status}</span>
              <small>
                {job.parser_kind.replaceAll("_", " ")}
                {#if file}· {formatBytes(file.size_bytes)} · via {file.uploaded_via}{/if}
                · {formatDate(job.created_at)}
              </small>
              {#if job.error_message}<small class="admin-error"
                  >{job.error_message}</small
                >{/if}
            </div>
            <div class="admin-actions">
              <button
                type="button"
                class="admin-button"
                disabled={busy}
                onclick={() => void reprocess(job)}>Reprocess</button
              >
            </div>
          </li>
        {/each}
      </ul>
    {:else if !loading}
      <p class="admin-muted">No imports yet.</p>
    {/if}
  </section>
</div>

<style>
  .stack {
    display: grid;
    gap: 0.85rem;
  }

  .upload {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    gap: 0.75rem;
  }

  label {
    display: grid;
    min-width: 0;
    max-width: 100%;
    gap: 0.3rem;
    font-size: 0.85rem;
  }

  input[type="file"] {
    width: 100%;
    max-width: 100%;
    min-width: 0;
    box-sizing: border-box;
  }

  select,
  input[type="file"] {
    min-height: 2.4rem;
    padding: 0.3rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.9rem;
  }
</style>
