<script lang="ts">
  import { page } from "$app/state";
  import StreamingView from "$lib/components/views/StreamingView.svelte";
</script>

<StreamingView lang={page.params.lang} />
