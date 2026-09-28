# Bug Fix: WAV header generation and integrity

- **Slug**: wav-header-corruption
- **Fixed**: 2026-09-27
- **Assessment**: ./assessment.md
- **Status**: applied

## Summary

Refactored WAV header generation to build the complete 44-byte standard PCM header in a pre-allocated memory buffer using `GenerateWavHeader` and persist it via a single atomic `Write` call, with boundary validation and size overflow protection.

## Changes

| File | Change | Notes |
|------|--------|-------|
| `src/backend/lib/audioengine/wav.go` | modified | Added `GenerateWavHeader` to assemble a 44-byte buffer in memory; refactored `writeWavHeader` to write atomically; added 4GB boundary checks in `FinalizeWavHeader`. |
| `src/backend/lib/audioengine/storage_test.go` | modified | Added `TestGenerateWavHeaderByteLayout` and `TestFinalizeWavHeaderEdgeCases` verifying exact byte layout, fallbacks, negative samples, and boundary clamping. |

## Diff Highlights

```go
// src/backend/lib/audioengine/wav.go
func GenerateWavHeader(ch uint16, dataSize uint32, sampleRate int) [wavHeaderSize]byte {
    if ch == 0 { ch = 2 }
    if sampleRate <= 0 { sampleRate = defaultWavSampleRate }

    byteRate := uint32(sampleRate) * uint32(ch) * wavBytesPerSample
    blockAlign := uint16(ch * wavBytesPerSample)

    var header [wavHeaderSize]byte
    copy(header[0:4], "RIFF")
    binary.LittleEndian.PutUint32(header[4:8], 36+dataSize)
    copy(header[8:12], "WAVE")
    copy(header[12:16], "fmt ")
    binary.LittleEndian.PutUint32(header[16:20], 16)
    binary.LittleEndian.PutUint16(header[20:22], 1)
    binary.LittleEndian.PutUint16(header[22:24], ch)
    binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
    binary.LittleEndian.PutUint32(header[28:32], byteRate)
    binary.LittleEndian.PutUint16(header[32:34], blockAlign)
    binary.LittleEndian.PutUint16(header[34:36], uint16(wavBitsPerSample))
    copy(header[36:40], "data")
    binary.LittleEndian.PutUint32(header[40:44], dataSize)

    return header
}

func writeWavHeader(f *os.File, ch uint16, dataSize uint32, sampleRate int) error {
    header := GenerateWavHeader(ch, dataSize, sampleRate)
    if _, err := f.Seek(0, io.SeekStart); err != nil {
        return err
    }
    _, err := f.Write(header[:])
    return err
}
```

## Tests Added or Updated

- `src/backend/lib/audioengine/storage_test.go::TestGenerateWavHeaderByteLayout` — Pins down byte-by-byte correctness of the 44-byte WAV header for stereo/mono and custom sample rates.
- `src/backend/lib/audioengine/storage_test.go::TestFinalizeWavHeaderEdgeCases` — Pins down nil file safety, negative sample handling, and 4GB maximum RIFF data size clamping.

## Local Verification

- Commands run:
  - `go test -v ./lib/audioengine -run "Test.*Wav.*|Test.*Header.*"` → PASS
  - `go test -race ./lib/audioengine` → PASS (1.531s)
  - `go test ./...` → PASS (all backend tests passed)

## Deviations from Assessment

None. The fix was applied as planned in the assessment.

## Follow-ups

- None.
