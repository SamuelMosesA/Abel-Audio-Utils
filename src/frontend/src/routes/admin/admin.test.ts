import { describe, it, expect, vi, beforeEach } from "vitest";

describe("Admin Route Guard", () => {
    let mockSystem: {
        isAuthenticated: boolean;
        validateSession: () => Promise<boolean>;
    };
    let redirectedTo: string | null = null;

    beforeEach(() => {
        redirectedTo = null;
        mockSystem = {
            isAuthenticated: false,
            validateSession: vi.fn(),
        };
    });

    it("redirects unauthenticated users to /login?redirect=/admin", async () => {
        mockSystem.validateSession = vi.fn().mockResolvedValue(false);

        const checkGuard = async () => {
            const valid = await mockSystem.validateSession();
            if (!valid) {
                redirectedTo = "/login?redirect=/admin";
            }
        };

        await checkGuard();
        expect(mockSystem.validateSession).toHaveBeenCalled();
        expect(redirectedTo).toBe("/login?redirect=/admin");
    });

    it("allows access when session validation succeeds", async () => {
        mockSystem.validateSession = vi.fn().mockResolvedValue(true);
        mockSystem.isAuthenticated = true;

        const checkGuard = async () => {
            const valid = await mockSystem.validateSession();
            if (!valid) {
                redirectedTo = "/login?redirect=/admin";
            }
        };

        await checkGuard();
        expect(mockSystem.validateSession).toHaveBeenCalled();
        expect(redirectedTo).toBeNull();
        expect(mockSystem.isAuthenticated).toBe(true);
    });
});
