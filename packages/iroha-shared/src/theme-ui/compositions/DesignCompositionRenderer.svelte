<script lang="ts">
  import type { Component } from "svelte";
  import {
    type DesignCompositionId,
    type DesignCompositionProps,
  } from "../../design-compositions";
  import Chronicle from "./Chronicle.svelte";
  import CommandCenter from "./CommandCenter.svelte";
  import CoverPage from "./CoverPage.svelte";
  import Editorial from "./Editorial.svelte";
  import FieldJournal from "./FieldJournal.svelte";
  import PersonalOS from "./PersonalOS.svelte";
  import Quiet from "./Quiet.svelte";

  const RENDERERS: Record<
    DesignCompositionId,
    Component<DesignCompositionProps>
  > = {
    editorial: Editorial,
    command: CommandCenter,
    chronicle: Chronicle,
    cover: CoverPage,
    workspace: PersonalOS,
    journal: FieldJournal,
    quiet: Quiet,
  };

  let {
    composition,
    today,
    readiness,
    links,
  }: DesignCompositionProps & {
    composition: DesignCompositionId;
  } = $props();

  const Renderer = $derived(RENDERERS[composition]);
</script>

<Renderer {today} {readiness} {links} />
