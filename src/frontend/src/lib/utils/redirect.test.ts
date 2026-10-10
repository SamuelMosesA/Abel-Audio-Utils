import { describe, it, expect } from "vitest";
import { resolveRedirect } from "./redirect";

describe("resolveRedirect", () => {
    it("returns valid relative paths", () => {
        expect(resolveRedirect("/admin")).toBe("/admin");
        expect(resolveRedirect("/admin/settings")).toBe("/admin/settings");
        expect(resolveRedirect("/admin?tab=audio#meters")).toBe("/admin?tab=audio#meters");
        expect(resolveRedirect("/")).toBe("/");
    });

    it("falls back to default when input is empty or null", () => {
        expect(resolveRedirect(null)).toBe("/admin");
        expect(resolveRedirect(undefined)).toBe("/admin");
        expect(resolveRedirect("")).toBe("/admin");
        expect(resolveRedirect("   ")).toBe("/admin");
    });

    it("uses custom fallback if provided", () => {
        expect(resolveRedirect(null, "/custom")).toBe("/custom");
        expect(resolveRedirect("", "/custom")).toBe("/custom");
    });

    it("rejects open redirect attacks and non-relative schemes", () => {
        expect(resolveRedirect("//evil.com")).toBe("/admin");
        expect(resolveRedirect("//evil.com/phish")).toBe("/admin");
        expect(resolveRedirect("/\\evil.com")).toBe("/admin");
        expect(resolveRedirect("https://evil.com")).toBe("/admin");
        expect(resolveRedirect("http://evil.com")).toBe("/admin");
        expect(resolveRedirect("javascript:alert(1)")).toBe("/admin");
        expect(resolveRedirect("/\tevil.com")).toBe("/admin");
        expect(resolveRedirect("/\nevil.com")).toBe("/admin");
    });
});
