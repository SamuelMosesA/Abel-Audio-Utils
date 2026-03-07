<script lang="ts">
    import { audioState } from "$lib/audioState.svelte";
    import { goto } from "$app/navigation";
    import { Button } from "$lib/components/ui/button/index.js";
    import { Input } from "$lib/components/ui/input/index.js";
    import { Label } from "$lib/components/ui/label/index.js";
    import * as Card from "$lib/components/ui/card/index.js";
    import { Lock, AlertCircle, ChevronLeft, Loader2 } from "lucide-svelte";

    let password = $state("");
    let error = $state("");
    let isLoading = $state(false);

    const handleLogin = async () => {
        if (!password) return;
        isLoading = true;
        error = "";
        
        const success = await audioState.login(password);
        if (success) {
            goto("/admin");
        } else {
            error = "Invalid administrator password";
        }
        isLoading = false;
    };
</script>

<div class="max-w-md mx-auto py-20 space-y-8 animate-in fade-in zoom-in-95 duration-500">
    <Button 
        variant="ghost" 
        onclick={() => goto("/")} 
        class="gap-2 group/back text-muted-foreground hover:text-foreground transition-all duration-300"
    >
        <ChevronLeft class="w-4 h-4 group-hover/back:-translate-x-1 transition-transform duration-300" />
        Back
    </Button>

    <Card.Root class="bg-black/60 border-white/10 backdrop-blur-2xl shadow-[0_0_50px_rgba(0,0,0,0.5)] overflow-hidden pt-8 ring-1 ring-white/5">
        <Card.Header class="text-center space-y-4">
            <div class="mx-auto p-5 bg-amber-500/10 rounded-full w-fit relative group">
                <div class="absolute inset-0 bg-amber-500/20 rounded-full blur-xl group-hover:blur-2xl transition-all duration-700"></div>
                <Lock class="w-10 h-10 text-amber-500 relative animate-pulse duration-[3s]" />
            </div>
            <div>
                <Card.Title class="text-2xl font-black tracking-tighter text-white">ADMIN PORTAL</Card.Title>
                <Card.Description class="text-white/40 font-medium">Secure Access Only</Card.Description>
            </div>
        </Card.Header>
        <Card.Content class="space-y-6 px-8 pb-10">
            {#if audioState.wasKicked}
                <div class="flex items-center gap-2 p-3 bg-amber-500/10 border border-amber-500/20 rounded-xl text-amber-500 text-xs font-bold animate-in slide-in-from-top-2">
                    <AlertCircle class="w-4 h-4" />
                    Session ended by another administrator.
                </div>
            {/if}

            {#if error}
                <div class="flex items-center gap-2 p-3 bg-red-500/10 border border-red-500/20 rounded-xl text-red-500 text-xs font-bold animate-in slide-in-from-top-2">
                    <AlertCircle class="w-4 h-4" />
                    {error}
                </div>
            {/if}

            <div class="space-y-3">
                <Label for="password" class="text-[10px] font-black uppercase tracking-[0.2em] text-white/30 ml-1 leading-none">Access Key</Label>
                <Input 
                    id="password" 
                    type="password" 
                    bind:value={password} 
                    oninput={() => audioState.wasKicked = false}
                    placeholder="••••••••"
                    class="bg-white/5 border-white/10 h-14 text-lg tracking-widest focus:ring-amber-500/20 rounded-xl text-white placeholder:text-white/10"
                    onkeydown={(e: KeyboardEvent) => e.key === "Enter" && handleLogin()}
                />
            </div>

            <Button 
                class="w-full h-14 bg-amber-600 hover:bg-amber-500 text-white font-black text-sm tracking-widest shadow-[0_0_30px_rgba(217,119,6,0.2)] rounded-xl group/btn transition-all duration-500"
                onclick={handleLogin}
                disabled={isLoading}
            >
                {#if isLoading}
                    <Loader2 class="w-5 h-5 animate-spin mr-2" />
                    VERIFYING...
                {:else}
                    UNLOCK AUDIO ENGINE
                    <Lock class="w-4 h-4 ml-2 group-hover/btn:scale-110 group-hover/btn:rotate-12 transition-all duration-500" />
                {/if}
            </Button>
        </Card.Content>
    </Card.Root>
</div>

<style>
    :global(body) {
        background: radial-gradient(circle at top center, #111 0%, #000 100%);
    }
</style>
