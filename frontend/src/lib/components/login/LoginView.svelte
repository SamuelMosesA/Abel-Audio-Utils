<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    const { system } = getAppContext();
    import { goto } from "$app/navigation";
    import { Lock, AlertCircle, ChevronLeft, Loader2, Shield } from "lucide-svelte";
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
                goto("/admin");
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

<div class="max-w-lg mx-auto py-20 px-6 space-y-8 animate-in fade-in">
    <div class="flex items-center justify-between">
        <Button 
            onclick={() => goto("/")} 
            variant="ghost"
            class="px-2 h-9 text-muted-foreground hover:text-white"
        >
            <ChevronLeft class="w-4 h-4 mr-1" />
            Back Home
        </Button>
    </div>

    <Card title="Administrator Access" class="glass border-primary/20 p-6 space-y-8">
        <div class="space-y-4">
            <div class="p-3 bg-primary/10 rounded-2xl w-fit">
                <Shield class="w-10 h-10 text-primary" />
            </div>
            <div class="space-y-1">
                <p class="text-muted-foreground text-sm">Secure authorization required to control the audio engine.</p>
            </div>
        </div>

        <div class="space-y-8">
            {#if error}
                <div class="flex items-center gap-2 p-4 bg-destructive/10 border border-destructive/20 rounded-xl text-destructive text-sm font-medium animate-in slide-in-from-top-2">
                    <AlertCircle class="w-4 h-4 shrink-0" />
                    {error}
                </div>
            {/if}

            <div class="space-y-6">
                <div class="space-y-3">
                    <label for="username" class="text-xxs font-black uppercase tracking-extra text-muted-foreground ml-1">Username</label>
                    <Input 
                        id="username" 
                        type="text" 
                        bind:value={username} 
                        placeholder="admin"
                        class="h-12 bg-muted/50 text-base"
                        onkeydown={(e: KeyboardEvent) => e.key === "Enter" && handleLogin()}
                    />
                </div>

                <div class="space-y-3">
                    <label for="password" class="text-xxs font-black uppercase tracking-extra text-muted-foreground ml-1">Access Key</label>
                    <Input 
                        id="password" 
                        type="password" 
                        bind:value={password} 
                        placeholder="••••••••"
                        class="h-12 bg-muted/50 text-base"
                        onkeydown={(e: KeyboardEvent) => e.key === "Enter" && handleLogin()}
                    />
                </div>
            </div>

            <Button 
                class="w-full h-14 font-bold uppercase tracking-widest text-xs"
                onclick={handleLogin}
                disabled={isLoading}
            >
                {#if isLoading}
                    <Loader2 class="w-5 h-5 animate-spin mr-2" />
                    Authenticating...
                {:else}
                    Unlock Audio Console
                    <Lock class="w-4 h-4 ml-2" />
                {/if}
            </Button>
        </div>
    </Card>
</div>
