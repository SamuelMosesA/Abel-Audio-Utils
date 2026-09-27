import { describe, expect, it } from "vitest";
import { formatTimecode, parseTimecode } from "./timecode";

describe("audio timecodes", () => {
    it("formats minutes, hours and fractional positions", () => {
        expect(formatTimecode(2450)).toBe("40:50");
        expect(formatTimecode(3670.5)).toBe("1:01:10.5");
        expect(formatTimecode(1.94)).toBe("0:01.9");
    });
    it("parses minute and hour inputs", () => {
        expect(parseTimecode("40:50")).toBe(2450);
        expect(parseTimecode("1:01:10.5")).toBe(3670.5);
    });
    it("rejects ambiguous and invalid inputs", () => {
        for (const input of ["42", "1:60", "1:12:60", "-1:20", "1:2:3:4"]) expect(parseTimecode(input)).toBeNull();
    });
});
