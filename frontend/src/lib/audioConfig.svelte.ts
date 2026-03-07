import { audioState } from "./audioState.svelte";
import { fetchWithSync } from "./utils/api";

class AudioConfig {
    async connectDevice(id: number) {
        await fetchWithSync("/api/control", {
            method: "POST",
            body: JSON.stringify({ action: "connect", DeviceID: id })
        });
    }

    async toggleRecording() {
        const action = audioState.isRecording ? "stop" : "start";
        await fetchWithSync("/api/control", {
            method: "POST",
            body: JSON.stringify({ action })
        });
    }

    async updateConfig() {
        await fetchWithSync("/api/control", {
            method: "POST",
            body: JSON.stringify({
                action: "update",
                chL: parseInt(audioState.chL.toString()),
                chR: parseInt(audioState.chR.toString()),
                Boost: parseFloat(audioState.boost.toString())
            })
        });
    }
}

export const audioConfig = new AudioConfig();
