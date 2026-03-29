<script lang="ts">
  let { value = 0, label = '', class: className = '' } = $props();
  
  // value is expected 0-1
  const percent = $derived(Math.min(100, Math.max(0, value * 100)));
  
  // Color logic for audio levels
  const colorClass = $derived(
    percent > 90 ? 'bg-destructive' : 
    percent > 70 ? 'bg-yellow-500' : 
    'bg-primary'
  );
</script>

<div class="flex flex-col gap-1 {className}">
  {#if label}
    <div class="flex justify-between text-xxs uppercase font-bold text-muted-foreground">
      <span>{label}</span>
      <span>{Math.round(percent)}%</span>
    </div>
  {/if}
  <div class="h-2 w-full bg-secondary rounded-full overflow-hidden border border-border/50">
    <div 
      class="h-full {colorClass} transition-all duration-75" 
      style="width: {percent}%"
    ></div>
  </div>
</div>
