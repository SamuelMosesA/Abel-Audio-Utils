<script lang="ts">
  let { 
    type = 'text',
    value = $bindable(),
    placeholder = '',
    class: className = '',
    id = '',
    ...props 
  } = $props();

  const baseClasses = "flex h-10 w-full rounded-md border border-input bg-muted/30 px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 transition-all duration-200 focus:bg-muted/50 focus:border-primary/50";
</script>

<input
  {id}
  {type}
  bind:value={value}
  placeholder={placeholder}
  class="{baseClasses} {className}"
  {...props}
/>
