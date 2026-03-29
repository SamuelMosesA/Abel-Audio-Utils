<script lang="ts">
  let { 
    variant = 'primary', 
    size = 'md', 
    class: className = '', 
    disabled = false,
    children, 
    ...props 
  } = $props();

  const variants = {
    primary: 'bg-primary text-primary-foreground hover:bg-primary/90 shadow',
    secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80 shadow-sm',
    destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90 shadow-sm',
    outline: 'border border-input bg-background hover:bg-accent hover:text-accent-foreground shadow-sm',
    ghost: 'hover:bg-accent hover:text-accent-foreground',
    link: 'text-primary underline-offset-4 hover:underline'
  };

  const sizes = {
    sm: 'h-8 px-3 text-xs',
    md: 'h-10 px-4 py-2',
    lg: 'h-12 px-8 text-base',
    icon: 'h-10 w-10'
  };

  const currentClasses = $derived(`inline-flex items-center justify-center rounded-md font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 ${variants[variant as keyof typeof variants] || ''} ${sizes[size as keyof typeof sizes] || ''} ${className}`);
</script>

<button
  {disabled}
  class={currentClasses}
  {...props}
>
  {@render children()}
</button>
