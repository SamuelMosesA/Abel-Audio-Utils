<script lang="ts">
  import { getAppContext } from "$lib/audioState.svelte";
  const { audio } = getAppContext();
  import Card from "$lib/components/ui/Card.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import LanguageSelector from "$lib/components/ai/LanguageSelector.svelte";
  import LiveAudioPlayer from "$lib/components/audio/LiveAudioPlayer.svelte";
  import { goto } from "$app/navigation";
  import { Globe, ArrowRight, QrCode, Volume2, Wifi } from "lucide-svelte";
  import { onMount } from "svelte";
  import QRCode from "qrcode";

  let selectedLang = $state('');
  let qrCodeDataUrl = $state('');
  let wifiSSID = $state('');
  let displayEndpoint = $state('');
  let serverUrl = $state('');

  function goToAILiveAudio() {
    if (selectedLang) {
      goto(`/ai_live_audio/${selectedLang}`);
    }
  }

  onMount(async () => {
    let targetUrl = window.location.href;
    try {
      const res = await fetch("/api/system/connection");
      if (res.ok) {
        const data = await res.json();
        wifiSSID = data.ssid || '';
        displayEndpoint = data.displayEndpoint || '';
        serverUrl = data.serverUrl || '';
        if (data.serverUrl) {
          targetUrl = data.serverUrl;
        }
      }
    } catch (err) {
      console.error("Failed to fetch connection info", err);
    }

    try {
      qrCodeDataUrl = await QRCode.toDataURL(targetUrl, {
        margin: 1,
        scale: 6,
        color: {
          dark: '#000000',
          light: '#ffffff'
        }
      });
    } catch (err) {
      console.error("QR Code generation error", err);
    }
  });
</script>

<div class="space-y-8">
  <div class="space-y-1">
    <h1 class="text-2xl sm:text-3xl font-bold tracking-tight">Audio Proxy Broadcast</h1>
    <p class="text-sm text-muted-foreground">
      Listen to real-time low-latency audio feeds with AI-assisted translation and subtitles.
    </p>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- AI Translation Section -->
    <div class="lg:col-span-2">
      <Card 
        title="AI Live Translation & Accessibility"
        description="Select your preferred language to listen to translated audio and follow real-time subtitles."
      >
        <div class="space-y-6">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 items-end">
            <div class="space-y-2">
              <label for="lang-select" class="text-xs font-medium text-muted-foreground">Select Language</label>
              <LanguageSelector 
                selected={selectedLang} 
                onchange={(val: string) => selectedLang = val} 
              />
            </div>
            <div>
              <Button 
                variant="primary" 
                disabled={!selectedLang} 
                onclick={goToAILiveAudio}
                class="w-full"
              >
                Join AI Stream <ArrowRight class="w-4 h-4 ml-1" />
              </Button>
            </div>
          </div>
          
          <div class="p-3 bg-muted/40 rounded-md border border-border/60">
            <p class="text-xs text-muted-foreground">
              Subtitles and dedicated low-latency audio streams are automatically generated for all supported feeds.
            </p>
          </div>
        </div>
      </Card>
    </div>

    <!-- Scan to Join Section -->
    <Card 
      title="Scan to Join"
      description="Scan with your phone camera to open on mobile."
    >
      <div class="flex flex-col items-center justify-center space-y-3 py-2">
        <div class="p-2 bg-white rounded-lg border border-border shadow-sm">
          {#if qrCodeDataUrl}
            <img src={qrCodeDataUrl} alt="Join QR Code" class="w-32 h-32" />
          {:else}
            <div class="w-32 h-32 flex items-center justify-center bg-muted rounded">
              <QrCode class="w-8 h-8 text-muted-foreground animate-pulse" />
            </div>
          {/if}
        </div>

        {#if displayEndpoint}
          <div class="text-center">
            <span class="text-xs font-mono font-medium text-muted-foreground bg-muted/50 px-2 py-0.5 rounded border border-border/50">
              {displayEndpoint}
            </span>
          </div>
        {/if}

        {#if wifiSSID && wifiSSID !== 'N/A'}
          <div class="flex items-center gap-1.5 px-3 py-1 rounded-md bg-secondary text-secondary-foreground border border-border text-xs font-medium">
            <Wifi class="w-3.5 h-3.5 text-muted-foreground" />
            <span>Wi-Fi: <strong class="font-semibold text-foreground">{wifiSSID}</strong></span>
          </div>
        {/if}
      </div>
    </Card>

    <!-- Direct Broadcast Section (Untranslated) -->
    <div class="lg:col-span-3">
      <Card 
        title="Direct Source Broadcast"
        description="Listen to the original audio feed directly without translation."
      >
        <div class="space-y-4">
          <div class="flex items-center gap-3">
            <div class="p-2.5 bg-primary/10 text-primary rounded-md border border-primary/20">
              <Volume2 class="w-5 h-5" />
            </div>
            <div>
              <h4 class="text-sm font-semibold text-foreground">Original Master Feed</h4>
              <p class="text-xs text-muted-foreground">Default unprocessed broadcast stream</p>
            </div>
          </div>

          <LiveAudioPlayer
            src="/api/audio/stream"
            label="Original live audio"
          />
        </div>
      </Card>
    </div>
  </div>
</div>
