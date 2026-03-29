<script lang="ts">
    import { audioState } from "$lib/audioState.svelte";
    import { goto } from "$app/navigation";
    import { Label } from "$lib/components/ui/label/index.js";
    import { Lock, AlertCircle, ChevronLeft, Loader2, Shield } from "lucide-svelte";
    import SimpleCard from "./ui/SimpleCard.svelte";
    import SimpleButton from "./ui/SimpleButton.svelte";
    import SimpleInput from "./ui/SimpleInput.svelte";

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

<div class="max-w-lg mx-auto py-32 px-6 space-y-12 animate-in fade-in zoom-in-[0.99] duration-700">
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

    <SimpleCard class="space-y-12 py-12 px-10">
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
            {#if audioState.wasKicked || error}
                <div class="flex items-center gap-2 p-4 bg-destructive/10 border border-destructive/20 rounded-xl text-destructive text-sm font-medium animate-in slide-in-from-top-2">
                    <AlertCircle class="w-4 h-4 shrink-0" />
                    {audioState.wasKicked ? "Session ended by another administrator." : error}
                </div>
            {/if}

            <div class="space-y-3">
                <Label for="password" class="text-xs font-black uppercase tracking-[0.2em] text-muted-foreground ml-1">System Access Key</Label>
                <SimpleInput 
                    id="password" 
                    type="password" 
                    bind:value={password} 
                    oninput={() => audioState.wasKicked = false}
                    placeholder="••••••••"
                    class="h-14 bg-black text-lg tracking-widest placeholder:text-muted-foreground/20"
                    onkeydown={(e: KeyboardEvent) => e.key === "Enter" && handleLogin()}
                />
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
