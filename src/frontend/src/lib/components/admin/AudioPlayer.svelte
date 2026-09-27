<script lang="ts">
    import { Play, Pause } from "lucide-svelte";

    let { src, duration = 0, position = $bindable(0), selectionStart, selectionEnd }:
        { src: string; duration?: number; position?: number; selectionStart?: number; selectionEnd?: number } = $props();
    let audio: HTMLAudioElement;
    let playing = $state(false);
    let length = $state(0);
    let selectionMode = false;

    const stamp = (seconds: number) => {
        const n = Math.floor(Math.max(0, seconds || 0));
        const m = Math.floor(n / 60);
        return n >= 3600 ? `${Math.floor(n / 3600)}:${String(m % 60).padStart(2, '0')}:${String(n % 60).padStart(2, '0')}` : `${m}:${String(n % 60).padStart(2, '0')}`;
    };
    function pause() { audio?.pause(); playing = false; selectionMode = false; }
    async function toggle() {
        if (!audio) return;
        if (playing) { pause(); return; }
        selectionMode = false;
        audio.volume = 1;
        audio.muted = false;
        try { await audio.play(); } catch { playing = false; }
    }
    function seek(value: number) {
        if (!audio || !Number.isFinite(value)) return;
        audio.currentTime = Math.max(0, Math.min(value, length));
        position = audio.currentTime;
    }
    export async function playSelection(start: number, end: number) {
        if (!audio || !Number.isFinite(start) || !Number.isFinite(end) || end <= start) return;
        pause();
        seek(start);
        selectionMode = true;
        audio.volume = 1;
        audio.muted = false;
        try { await audio.play(); } catch { selectionMode = false; playing = false; }
    }
</script>

<audio bind:this={audio} {src} preload="metadata"
    onloadedmetadata={() => { length = audio.duration || duration; position = audio.currentTime; audio.volume = 1; audio.muted = false; }}
    ontimeupdate={() => { position = audio.currentTime; if (selectionMode && selectionEnd !== undefined && position >= selectionEnd) pause(); }}
    onplay={() => playing = true} onpause={() => playing = false} onended={pause}></audio>
<div class="flex items-center gap-2 min-w-0" role="group" aria-label="Audio preview">
    <button type="button" class="shrink-0 rounded-full bg-primary p-2 text-primary-foreground" onclick={toggle} aria-label={playing ? 'Pause audio' : 'Play audio'}>
        {#if playing}<Pause class="w-4 h-4" />{:else}<Play class="w-4 h-4" />{/if}
    </button>
    <span class="text-xs tabular-nums text-muted-foreground whitespace-nowrap">{stamp(position)}</span>
    <input type="range" class="min-w-0 flex-1 accent-primary" min="0" max={length || 1} step="0.1" value={position}
        aria-label="Audio position" oninput={e => seek(Number(e.currentTarget.value))} />
    <span class="text-xs tabular-nums text-muted-foreground whitespace-nowrap">{stamp(length)}</span>
</div>
