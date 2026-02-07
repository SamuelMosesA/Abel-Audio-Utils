import { audioState, type MeterState } from "./audioState.svelte";

class AudioVisuals {
    currentMeters = $state<MeterState>({ L: 0, R: 0 });
    monitoring = $state(false);
    // Smoothed numeric dB values for display (dBFS, negative numbers where 0 = full scale)
    currentDb = $state<MeterState>({ L: -100, R: -100 });

    #targetMeters: MeterState = { L: 0, R: 0 };
    #targetDb: MeterState = { L: -100, R: -100 };
    #audioCtx: AudioContext | null = null;
    #nextStartTime = 0;
    #LATENCY_BUFFER = 0.1;
    #DECAY = 0.25;

    constructor() {
        audioState.onMessage = (dv) => this.processData(dv);
        this.runVisualLoop();
    }

    processData(dv: DataView) {
        const rL = dv.getFloat32(0, true);
        const rR = dv.getFloat32(4, true);

        // Convert linear peak (0..1) to dBFS. Use a floor to avoid -Infinity
        // for zero values. Map dB range [MIN_DB..0] to percent [0..100]
        const MIN_DB = -100;
        const toDb = (v: number) => {
            if (v <= 0) return MIN_DB;
            const db = 20 * Math.log10(v);
            return db < MIN_DB ? MIN_DB : db;
        };

        const dbL = toDb(rL);
        const dbR = toDb(rR);

        this.#targetDb.L = dbL;
        this.#targetDb.R = dbR;

        const dbToPercent = (db: number) => {
            const DISPLAY_MIN = -60;
            return Math.min(Math.max(((db - DISPLAY_MIN) / -DISPLAY_MIN) * 100, 0), 100);
        };

        this.#targetMeters.L = dbToPercent(dbL);
        this.#targetMeters.R = dbToPercent(dbR);

        if (this.monitoring && this.#audioCtx) {
            this.scheduleAudio(dv, 8);
        } else {
            this.#nextStartTime = 0;
        }
    }

    scheduleAudio(dataView: DataView, offset: number) {
        if (!this.#audioCtx) return;
        if (this.#audioCtx.state === 'suspended') this.#audioCtx.resume();

        const floatData = new Float32Array(dataView.buffer.slice(offset));
        const buffer = this.#audioCtx.createBuffer(2, floatData.length / 2, 48000);
        const chL = buffer.getChannelData(0);
        const chR = buffer.getChannelData(1);

        for (let i = 0; i < floatData.length / 2; i++) {
            chL[i] = floatData[i * 2];
            chR[i] = floatData[i * 2 + 1];
        }

        const now = this.#audioCtx.currentTime;
        if (this.#nextStartTime < now || this.#nextStartTime > now + 1.0) {
            this.#nextStartTime = now + this.#LATENCY_BUFFER;
        }

        const source = this.#audioCtx.createBufferSource();
        source.buffer = buffer;
        source.connect(this.#audioCtx.destination);
        source.start(this.#nextStartTime);
        this.#nextStartTime += buffer.duration;
    }

    async toggleMonitor() {
        if (this.monitoring) {
            // Disable monitoring
            this.monitoring = false;
            if (this.#audioCtx && this.#audioCtx.state !== 'closed') {
                await this.#audioCtx.suspend();
            }
            this.#nextStartTime = 0;
        } else {
            // Enable monitoring
            if (!this.#audioCtx) {
                const AudioContextClass = (window as any).AudioContext || (window as any).webkitAudioContext;
                this.#audioCtx = new AudioContextClass({
                    latencyHint: 'interactive',
                    sampleRate: 48000
                });
            }
            if (this.#audioCtx && this.#audioCtx.state === 'suspended') {
                await this.#audioCtx.resume();
            }
            this.monitoring = true;
        }
    }

    runVisualLoop() {
        const tick = () => {
            // Smooth percent meters
            this.currentMeters.L -= (this.currentMeters.L - this.#targetMeters.L) * this.#DECAY;
            this.currentMeters.R -= (this.currentMeters.R - this.#targetMeters.R) * this.#DECAY;

            // Smooth numeric dB readout using same decay
            this.currentDb.L -= (this.currentDb.L - this.#targetDb.L) * this.#DECAY;
            this.currentDb.R -= (this.currentDb.R - this.#targetDb.R) * this.#DECAY;

            if (this.currentMeters.L < 0.1) this.currentMeters.L = 0;
            if (this.currentMeters.R < 0.1) this.currentMeters.R = 0;
            if (this.currentDb.L < -99) this.currentDb.L = -100; // display as -∞ if needed
            if (this.currentDb.R < -99) this.currentDb.R = -100;

            requestAnimationFrame(tick);
        };
        requestAnimationFrame(tick);
    }
}

export const audioVisuals = new AudioVisuals();
