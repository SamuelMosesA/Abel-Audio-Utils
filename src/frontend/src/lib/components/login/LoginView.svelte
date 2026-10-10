<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    const { system } = getAppContext();
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { resolveRedirect } from "$lib/utils/redirect";
    import { Lock, AlertCircle, ChevronLeft, Loader2 } from "lucide-svelte";
    import Card from "../ui/Card.svelte";
    import Button from "../ui/Button.svelte";
    import Input from "../ui/Input.svelte";

    let username = $state("");
    let password = $state("");
    let error = $state("");
    let isLoading = $state(false);

    const handleLogin = async () => {
        if (!username || !password) return;
        isLoading = true;
        error = "";
        
        try {
            const success = await system.login(username, password);
            if (success) {
                const target = resolveRedirect(page.url.searchParams.get("redirect"));
                goto(target);
            } else {
                error = "Invalid administrator credentials.";
            }
        } catch (e) {
            error = "Authentication service unavailable.";
            console.error(e);
        } finally {
            isLoading = false;
        }
    };
</script>

<div class="max-w-md mx-auto py-8 sm:py-16 space-y-6">
    <div>
        <Button 
            onclick={() => goto("/")} 
            variant="ghost"
            size="sm"
            class="text-muted-foreground hover:text-foreground -ml-2"
        >
            <ChevronLeft class="w-4 h-4 mr-1" />
            Back to Broadcast
        </Button>
    </div>

    <Card 
        title="Admin Sign In" 
        description="Enter credentials to configure audio devices and processing."
    >
        <div class="space-y-4">
            {#if error}
                <div class="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm font-medium">
                    <AlertCircle class="w-4 h-4 shrink-0" />
                    <span>{error}</span>
                </div>
            {/if}

            <div class="space-y-4">
                <div class="space-y-1.5">
                    <label for="username" class="text-sm font-medium text-foreground">Username</label>
                    <Input 
                        id="username" 
                        type="text" 
                        bind:value={username} 
                        placeholder="admin"
                        autocomplete="username"
                        onkeydown={(e: KeyboardEvent) => e.key === "Enter" && handleLogin()}
                    />
                </div>

                <div class="space-y-1.5">
                    <label for="password" class="text-sm font-medium text-foreground">Password</label>
                    <Input 
                        id="password" 
                        type="password" 
                        bind:value={password} 
                        placeholder="••••••••"
                        autocomplete="current-password"
                        onkeydown={(e: KeyboardEvent) => e.key === "Enter" && handleLogin()}
                    />
                </div>
            </div>

            <Button 
                class="w-full mt-2"
                onclick={handleLogin}
                disabled={isLoading}
            >
                {#if isLoading}
                    <Loader2 class="w-4 h-4 animate-spin mr-2" />
                    Signing in...
                {:else}
                    <Lock class="w-4 h-4 mr-2" />
                    Sign In
                {/if}
            </Button>
        </div>
    </Card>
</div>
