<script lang="ts">
  import { getAppContext } from "$lib/audioState.svelte";
  const { audio } = getAppContext();
  import Card from "$lib/components/ui/Card.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import LanguageSelector from "$lib/components/ai/LanguageSelector.svelte";
  import { goto } from "$app/navigation";
  import { Globe, ArrowRight, QrCode } from "lucide-svelte";
  import { onMount } from "svelte";
  import QRCode from "qrcode";

  let selectedLang = $state('');
  let qrCodeDataUrl = $state('');
  let wifiSSID = $state('');

  function goToAILiveAudio() {
    if (selectedLang) {
      goto(`/ai_live_audio/${selectedLang}`);
    }
  }

  onMount(async () => {
    // Fetch system connection info
    try {
      const res = await fetch("/api/system/connection");
      if (res.ok) {
        const data = await res.json();
        wifiSSID = data.ssid;
      }
    } catch (err) {
      console.error("Failed to fetch connection info", err);
    }

    try {
      const url = window.location.href;
      qrCodeDataUrl = await QRCode.toDataURL(url, {
        margin: 2,
        scale: 8,
        color: {
          dark: '#ffffff',
          light: '#00000000'
        }
      });
    } catch (err) {
      console.error("QR Code generation error", err);
    }
  });
</script>

<header class="flex items-center justify-between mb-12 border-b border-border/40 pb-6">
  <div class="flex items-center gap-3">
    <div class="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center font-black text-primary">
      AV
    </div>
    <div>
      <h1 class="text-xl font-bold tracking-tight">Audio Proxy</h1>
      <p class="text-xxs font-black uppercase tracking-widest text-muted-foreground">Broadcast Interface</p>
    </div>
  </div>
  <div class="flex items-center gap-3">
    <Button variant="ghost" size="sm" onclick={() => goto("/login")}>
      Admin
    </Button>
  </div>
</header>

<div class="max-w-4xl mx-auto py-12 px-6 space-y-12">
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
    <!-- AI Accessibility Section -->
    <div class="lg:col-span-2">
      <Card title="AI Live Translation & Accessibility">
        <div class="space-y-6">
          <div class="flex items-center gap-4">
            <div class="p-3 bg-secondary text-secondary-foreground rounded-full">
              <Globe class="w-6 h-6" />
            </div>
            <div>
              <h2 class="text-xl font-bold">Real-time Translation</h2>
              <p class="text-muted-foreground text-sm">Select your language to hear and read live AI-generated translations.</p>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4 items-end">
            <div class="space-y-2">
              <span class="text-xs font-bold uppercase text-muted-foreground ml-1">Choose Language</span>
              <LanguageSelector 
                selected={selectedLang} 
                onchange={(val: string) => selectedLang = val} 
              />
            </div>
            <Button 
              variant="secondary" 
              disabled={!selectedLang} 
              onclick={goToAILiveAudio}
              class="w-full"
            >
              Join AI Stream <ArrowRight class="ml-2 w-4 h-4" />
            </Button>
          </div>
          
          <div class="p-4 bg-muted/20 rounded-lg border border-border">
            <p class="text-xs text-muted-foreground italic">
              * Subtitles and dedicated audio streams are automatically enabled for all AI-assisted feeds.
            </p>
          </div>
        </div>
      </Card>
    </div>

    <!-- Scan to Join Section -->
    <Card title="Scan to Join">
      <div class="flex flex-col items-center justify-center space-y-6 py-4">
        <div class="relative group">
          <div class="absolute -inset-1 bg-gradient-to-r from-primary to-secondary rounded-xl blur opacity-25 group-hover:opacity-50 transition duration-1000 group-hover:duration-200"></div>
          <div class="relative bg-black rounded-lg p-2 border border-white/10">
            {#if qrCodeDataUrl}
              <img src={qrCodeDataUrl} alt="Join QR Code" class="w-32 h-32" />
            {:else}
              <div class="w-32 h-32 flex items-center justify-center bg-muted/20 rounded">
                <QrCode class="w-8 h-8 text-muted-foreground animate-pulse" />
              </div>
            {/if}
          </div>
        </div>
        <div class="text-center space-y-1">
          <p class="text-xxs font-black uppercase tracking-widest text-muted-foreground">Mobile Access</p>
          <p class="text-xs text-muted-foreground/60">Scan to open this page on your device</p>
          {#if wifiSSID && wifiSSID !== 'N/A'}
            <div class="pt-3 flex flex-col items-center gap-1">
              <span class="text-tiny font-bold uppercase tracking-extra text-primary/50">Network</span>
              <span class="text-sm font-black tracking-tight text-primary px-3 py-1 bg-primary/10 rounded-full border border-primary/20">{wifiSSID}</span>
            </div>
          {/if}
        </div>
      </div>
    </Card>
  </div>

  <div class="pt-12 text-center text-muted-foreground/40">
    <p class="text-xxs font-black uppercase tracking-ultra">
      Direct Interface • Low Latency Audio Proxy • AI Assisted Access
    </p>
  </div>
</div>
