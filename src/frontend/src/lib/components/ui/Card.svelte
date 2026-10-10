<script lang="ts">
  let { title = '', description = '', class: className = '', children } = $props();
</script>

<div class="rounded-lg border border-border bg-card text-card-foreground shadow-sm {className}">
  {#if title || description}
    <div class="flex flex-col space-y-1.5 p-6 pb-4 border-b border-border/40">
      {#if title}
        <h3 class="text-lg font-semibold leading-none tracking-tight">{title}</h3>
      {/if}
      {#if description}
        <p class="text-sm text-muted-foreground">{description}</p>
      {/if}
    </div>
  {/if}
  <div class="p-6">
    {@render children()}
  </div>
</div>
