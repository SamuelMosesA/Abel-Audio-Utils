# Bug Assessment: WAV header generation and integrity

- **Slug**: wav-header-corruption
- **Created**: 2026-09-27
- **Source**: https://github.com/SamuelMosesA/Abel-Audio-Utils/issues/34 (pasted text + codebase triage)
- **Verdict**: valid
- **Severity**: medium

## Report (verbatim or summarized)

> Debug WAV header corruption issue
> https://github.com/SamuelMosesA/Abel-Audio-Utils/issues/34
>
> WAV header generation in `src/backend/lib/audioengine/wav.go` writes 44 bytes across 13 separate partial `Write` / `binary.Write` syscalls without memory buffering. Any interrupted write, non-atomic seek, or field calculation edge-case can lead to corrupted or unplayable WAV files.

## Symptom

WAV headers written during recording initialization (`WritePlaceholderHeader`) or finalized upon recording stop (`FinalizeWavHeader`) are constructed via 13 separate unbuffered I/O operations rather than a single atomic 44-byte block. If any error or interrupt occurs during header writing, the header becomes partially corrupted, leaving the file unplayable by standard audio players.

## Reproduction

1. Start recording and stop recording via `POST /api/recordings`.
2. Inspect `src/backend/lib/audioengine/wav.go:38-107` (`writeWavHeader`).
3. Note that `writeWavHeader` performs 13 distinct calls (`f.Write` and `binary.Write`) directly on `*os.File` without pre-assembling the 44-byte header buffer.
4. Under disk pressure or abrupt failure, partial header writes corrupt the RIFF/WAVE container.

## Suspected Code Paths

- `src/backend/lib/audioengine/wav.go:38-107` (`writeWavHeader`) — Writes individual fields across multiple syscalls instead of pre-allocating and writing a single 44-byte buffer.
- `src/backend/lib/audioengine/wav.go:23-28` (`WritePlaceholderHeader`) — Relies on `writeWavHeader`.
- `src/backend/lib/audioengine/wav.go:30-36` (`FinalizeWavHeader`) — Relies on `writeWavHeader` and does not check for 32-bit `uint32` data size overflow (> 4GB WAV limit).
- `src/backend/lib/audioengine/storage_test.go` — Needs full byte-by-byte 44-byte header verification tests.

## Root Cause Hypothesis

Confidence: high. WAV headers are 44 bytes in length with fixed offsets (`RIFF`, size, `WAVE`, `fmt `, format chunk, channel count, sample rate, byte rate, block align, bits per sample, `data`, data size). The current implementation writes each sub-field sequentially via unbuffered system calls (`f.Write` and reflection-based `binary.Write`), which is fragile, inefficient, and susceptible to partial writes. Constructing a complete 44-byte array in memory and executing a single `f.WriteAt` or `f.Seek(0, io.SeekStart)` + `f.Write` ensures atomic header persistence and prevents corruption.

## Proposed Remediation

**Preferred**:
1. Pre-build the entire 44-byte WAV header in a `[44]byte` array using `binary.LittleEndian.PutUint16` and `binary.LittleEndian.PutUint32`.
2. Validate parameters (channels, sample rate, data size) and ensure data size does not exceed the standard 4GB RIFF limit (or clamp/return error).
3. Write the 44-byte header atomically in a single `f.Write` operation after seeking to offset 0.
4. Add comprehensive unit tests verifying byte-by-byte header alignment, magic bytes (`RIFF`, `WAVE`, `fmt `, `data`), and edge cases.

**Alternatives**:
- Use an external WAV encoding library (adds external dependency with larger blast radius).

**Files likely to change**:
- `src/backend/lib/audioengine/wav.go`
- `src/backend/lib/audioengine/storage_test.go`

**Tests to add or update**:
- `TestWavHeaderByteLayout` — Asserts exact 44-byte header structure with varying channel counts and sample rates (44.1kHz, 48kHz, mono, stereo).
- `TestWriteWavHeaderSingleWrite` — Verifies header is written as a single continuous block.

## Risks & Considerations

- Must strictly adhere to standard 16-bit PCM WAV (RIFF/WAVE format) specifications so files remain playable across macOS, Windows, Linux, and mobile players.
- Must maintain backward compatibility with `WritePlaceholderHeader` and `FinalizeWavHeader` signatures.

## Open Questions

- None. Fix can proceed directly via `/speckit-bug-fix`.
