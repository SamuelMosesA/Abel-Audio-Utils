import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import TranslationAdmin from './TranslationAdmin.svelte';
import { getAppContext } from '../audioState.svelte';

// Mock getAppContext
vi.mock('../audioState.svelte', async (importOriginal) => {
    const actual: any = await importOriginal();
    return {
        ...actual,
        getAppContext: vi.fn()
    };
});

describe('TranslationAdmin.svelte', () => {
    let mockAi: any;
    let mockAudio: any;

    beforeEach(() => {
        vi.clearAllMocks();
        mockAi = {
            geminiMasterEnabled: true,
            translations: [],
            setGeminiMaster: vi.fn(),
            stopTranslation: vi.fn()
        };
        mockAudio = {};
        (getAppContext as any).mockReturnValue({ ai: mockAi, audio: mockAudio, ui: {}, visuals: {} });
    });

    it('should show "No active translation sessions" when translations list is empty', () => {
        render(TranslationAdmin);
        expect(screen.getByText(/No active translation sessions/i)).toBeInTheDocument();
    });

    it('should render active translation sessions', () => {
        mockAi.translations = [
            { language: 'tamil', listeners: 1, subtitles: true }
        ];
        render(TranslationAdmin);
        expect(screen.getByText(/tamil/i)).toBeInTheDocument();
        expect(screen.getByText(/Subtitles ON/i)).toBeInTheDocument();
    });

    it('should call setGeminiMaster when Disable Gemini button is clicked', async () => {
        render(TranslationAdmin);
        const button = screen.getByText(/Disable Gemini/i);
        await fireEvent.click(button);
        expect(mockAi.setGeminiMaster).toHaveBeenCalledWith(false);
    });

    it('should call stopTranslation when Kill button is clicked and confirmed', async () => {
        const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);
        mockAi.translations = [
            { language: 'tamil', listeners: 1, subtitles: true }
        ];
        render(TranslationAdmin);
        const killButton = screen.getByText(/Kill/i);
        await fireEvent.click(killButton);
        expect(confirmSpy).toHaveBeenCalled();
        expect(mockAi.stopTranslation).toHaveBeenCalledWith('tamil');
    });
});
