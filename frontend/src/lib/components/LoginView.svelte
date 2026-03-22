<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    const { system } = getAppContext();
    import { goto } from "$app/navigation";
    import { Label } from "$lib/components/ui/label/index.js";
    import { Lock, AlertCircle, ChevronLeft, Loader2, Shield } from "lucide-svelte";
    import SimpleCard from "./ui/SimpleCard.svelte";
    import SimpleButton from "./ui/SimpleButton.svelte";
    import SimpleInput from "./ui/SimpleInput.svelte";

    let username = $state("");
    let password = $state("");
    let error = $state("");
    let isLoading = $state(false);

    const handleLogin = async () => {
        if (!username || !password) return;
        isLoading = true;
        error = "";
        
        const success = await system.login(username, password);
        if (success) {
            goto("/admin");
        } else {
            error = "Invalid administrator credentials.";
        }
        isLoading = false;
    };
</script>

<div class="max-w-lg mx-auto py-20 px-6 space-y-8 animate-in fade-in">
    <div class="flex items-center justify-between">
        <SimpleButton 
            onclick={() => goto("/")} 
            variant="ghost"
            class="px-0 hover:bg-transparent"
        >
            <ChevronLeft class="w-4 h-4 mr-1" />
            Back
        </SimpleButton>
    </div>

    <SimpleCard class="space-y-8 py-8 px-6">
        <div class="space-y-4">
            <div class="p-3 bg-primary/10 rounded-2xl w-fit">
                <Shield class="w-10 h-10 text-primary" />
            </div>
            <div class="space-y-1">
                <h1 class="text-3xl font-bold tracking-tight text-white">Administrator Access</h1>
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
                    <Label for="username" class="text-xs font-black uppercase tracking-[0.2em] text-muted-foreground ml-1">Username</Label>
                    <SimpleInput 
                        id="username" 
                        type="text" 
                        bind:value={username} 
                        placeholder="admin"
                        class="h-12 bg-muted/50 text-base"
                        onkeydown={(e: KeyboardEvent) => e.key === "Enter" && handleLogin()}
                    />
                </div>

                <div class="space-y-3">
                    <Label for="password" class="text-xs font-black uppercase tracking-[0.2em] text-muted-foreground ml-1">Access Key</Label>
                    <SimpleInput 
                        id="password" 
                        type="password" 
                        bind:value={password} 
                        placeholder="••••••••"
                        class="h-12 bg-muted/50 text-base"
                        onkeydown={(e: KeyboardEvent) => e.key === "Enter" && handleLogin()}
                    />
                </div>
            </div>

            <SimpleButton 
                class="w-full h-14"
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
            </SimpleButton>
        </div>
    </SimpleCard>
</div>
