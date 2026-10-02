export function formatTimecode(seconds: number): string {
    if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
    const tenths = Math.floor((seconds + 1e-9) * 10);
    const total = Math.floor(tenths / 10);
    const fraction = tenths % 10;
    const m = Math.floor(total / 60);
    const core = total >= 3600
        ? `${Math.floor(total / 3600)}:${String(m % 60).padStart(2, "0")}:${String(total % 60).padStart(2, "0")}`
        : `${m}:${String(total % 60).padStart(2, "0")}`;
    return core + (fraction > 0 ? `.${fraction}` : "");
}

export function parseTimecode(input: string): number | null {
    const parts = input.trim().split(":");
    if (parts.length !== 2 && parts.length !== 3) return null;
    const [first, middle, last] = parts.length === 3 ? parts : ["0", ...parts];
    if (!/^\d+$/.test(first) || !/^\d{1,2}$/.test(middle) || !/^\d{1,2}(?:\.\d{1,3})?$/.test(last)) return null;
    const hours = Number(first), minutes = Number(middle), seconds = Number(last);
    if (minutes >= 60 || seconds >= 60 || !Number.isSafeInteger(hours)) return null;
    return hours * 3600 + minutes * 60 + seconds;
}
