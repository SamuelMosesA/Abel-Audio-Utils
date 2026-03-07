<script lang="ts">
    import { type Snippet } from 'svelte';
    
    let { 
        children, 
        onclick, 
        variant = "primary", 
        class: className = "",
        disabled = false
    }: { 
        children: Snippet, 
        onclick?: () => void, 
        variant?: "primary" | "secondary" | "destructive" | "ghost",
        class?: string,
        disabled?: boolean
    } = $props();

    const variants = {
        primary: "bg-primary text-primary-foreground hover:brightness-110 shadow-[0_0_20px_rgba(34,197,94,0.15)]",
        secondary: "bg-secondary text-secondary-foreground border border-border/40 hover:bg-muted",
        destructive: "bg-destructive text-destructive-foreground hover:brightness-110 shadow-[0_0_20px_rgba(239,68,68,0.15)]",
        ghost: "text-muted-foreground hover:text-white hover:bg-muted border-none bg-transparent"
    };
</script>

<button
    {onclick}
    {disabled}
    class="h-10 md:h-12 px-5 md:px-6 font-bold rounded-lg transition-all active:scale-[0.98] flex items-center justify-center gap-2 text-xs md:text-sm tracking-widest uppercase {variants[variant]} {className}"
>
    {@render children()}
</button>
