<script lang="ts">
    import { type Snippet } from 'svelte';
    
    let { children, class: className = "" }: { children: Snippet, class?: string } = $props();
</script>

<div class="bg-card border border-border/40 rounded-[var(--radius)] p-4 md:p-6 transition-all {className}">
    {@render children()}
</div>
