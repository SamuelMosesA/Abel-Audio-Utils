import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import MeterPanel from './MeterPanel.svelte';
import { getAppContext } from '../audioState.svelte';

// Mock getAppContext
vi.mock('../audioState.svelte', async (importOriginal) => {
    const actual: any = await importOriginal();
    return {
        ...actual,
        getAppContext: vi.fn()
    };
});

describe('MeterPanel.svelte', () => {
    let mockVisuals: any;

    beforeEach(() => {
        vi.clearAllMocks();
        mockVisuals = {
            currentDb: { L: -100, R: -100 },
            currentMeters: { L: 0, R: 0 }
        };
        (getAppContext as any).mockReturnValue({ visuals: mockVisuals });
    });

    it('should render silence initially (minus infinity)', () => {
        render(MeterPanel);
        const labels = screen.getAllByText('−∞');
        expect(labels).toHaveLength(2);
    });

    it('should update display when currentDb changes', async () => {
        render(MeterPanel);
        
        mockVisuals.currentDb.L = -12.5;
        mockVisuals.currentDb.R = -45.2;

        // Force a re-render or wait for reactivity if possible, 
        // but since we are mocking the store object we need to be careful.
        // In this case, MeterPanel uses { visuals } = getAppContext() which is a reference to mockVisuals.
    });
});
