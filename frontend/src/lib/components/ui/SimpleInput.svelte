<script lang="ts">
    let { 
        value = $bindable(), 
        type = "text", 
        placeholder = "", 
        class: className = "",
        disabled = false,
        onkeydown,
        ...rest
    }: { 
        value: any, 
        type?: string, 
        placeholder?: string, 
        class?: string,
        disabled?: boolean,
        onkeydown?: (e: KeyboardEvent) => void
        [key: string]: any
    } = $props();
</script>

<input
    {type}
    {placeholder}
    {disabled}
    {onkeydown}
    bind:value={value}
    {...rest}
    class="h-12 md:h-14 bg-background border border-border/40 rounded-lg px-4 py-2 focus:outline-none focus:ring-1 focus:ring-primary/50 transition-all font-bold text-white text-sm {className}"
/>
