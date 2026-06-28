<script lang="ts">
  let { title, class: className = '', children } = $props();
</script>

<div class="bg-card text-card-foreground border border-border rounded-lg shadow-sm overflow-hidden {className}">
  {#if title}
    <div class="px-6 py-4 border-b border-border">
      <h3 class="text-lg font-semibold leading-none tracking-tight">{title}</h3>
    </div>
  {/if}
  <div class="p-6">
    {@render children()}
  </div>
</div>
