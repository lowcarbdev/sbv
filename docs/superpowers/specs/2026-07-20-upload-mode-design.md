# Upload Mode Selection Design

**Date:** 2026-07-20  
**Status:** Approved

## Summary

Add a user-selectable upload processing mode: `tempfile` (current behavior — write to disk then process) or `pipe` (stream directly via `io.Pipe`, no disk write). The mode is configurable at the server level via CLI flag / env var, and overridable per-request via a multipart form field.

## Mode Descriptions

### `tempfile` (default)
Current behavior. The uploaded file is written to `os.TempDir()/sbv-uploads/backup-*.xml`, then a background goroutine opens and parses it, then deletes it.

### `pipe`
No disk write. The handler pumps the multipart reader into an `io.PipeWriter` in one goroutine; a second goroutine reads from the `io.PipeReader` and passes it to `ParseSMSBackupStreaming`. No temp file is created or deleted.

## Configuration

Priority (highest to lowest):

1. Form field `upload_mode=pipe|tempfile` in the multipart POST body
2. `--upload-mode` CLI flag
3. `UPLOAD_MODE` environment variable
4. Default: `tempfile`

Invalid values at any level are ignored and the next level is tried.

## Architecture

### `main.go`

- Add `--upload-mode` flag (string, default `""`)
- Resolve with env var fallback to `UPLOAD_MODE`, then default to `tempfile`
- Validate: must be `pipe` or `tempfile`
- Pass resolved value to `internal.SetDefaultUploadMode(mode)`

### `internal/handlers.go` — `HandleUpload`

After parsing the multipart form, read the `upload_mode` form value. If it is `pipe` or `tempfile`, use it. Otherwise use the server default from `GetDefaultUploadMode()`.

**`tempfile` path** (unchanged):
```
multipart reader → SaveUploadedFile → tempFilePath → go ProcessUploadedFile(userID, username, tempFilePath)
```

**`pipe` path** (new):
```
multipart reader
  └─ goroutine A: io.Copy(pipeWriter, file); pipeWriter.CloseWithError(err)
  └─ goroutine B: ProcessUploadedFileFromReader(userID, username, pipeReader)
```

Both paths return `HTTP 200` immediately with `Processing: true`. Progress is polled via `/api/progress` as before.

### `internal/parser.go`

- Add package-level `defaultUploadMode string` with `SetDefaultUploadMode` / `GetDefaultUploadMode` accessors.
- Add `ProcessUploadedFileFromReader(userID, username string, r io.Reader)` — same body as `ProcessUploadedFile` but skips the `os.Open` and `os.Remove` steps, calling `ParseSMSBackupStreaming` directly with `r`.

## Data Flow Diagram

```
POST /api/upload
       │
       ▼
ParseMultipartForm (32 MB in-memory buffer)
       │
       ├── read upload_mode form field → resolve effective mode
       │
       ├── [tempfile] ──→ SaveUploadedFile ──→ go ProcessUploadedFile(path)
       │                                              │
       │                                    os.Open → ParseSMSBackupStreaming → os.Remove
       │
       └── [pipe] ──→ io.Pipe()
                          ├── go: io.Copy(pw, file); pw.CloseWithError(err)
                          └── go: ProcessUploadedFileFromReader(pr)
                                        │
                                  ParseSMSBackupStreaming(pr)
```

## Error Handling

- **Pipe writer error** (e.g. client disconnect mid-upload): `pipeWriter.CloseWithError(err)` propagates to the reader; `xml.Decoder.Token()` returns the error; `ParseSMSBackupStreaming` returns it; progress is set to `error`.
- **Invalid form field value**: silently ignored, server default is used.
- **Invalid CLI/env value**: logged and process exits (same pattern as `--blob-storage`).

## Files Changed

| File | Change |
|------|--------|
| `main.go` | Add `--upload-mode` flag, resolve + validate, call `SetDefaultUploadMode` |
| `internal/handlers.go` | Read form field, branch on effective mode |
| `internal/parser.go` | Add `defaultUploadMode`, `SetDefaultUploadMode`, `GetDefaultUploadMode`, `ProcessUploadedFileFromReader` |

## Non-Goals

- No frontend UI to expose the mode selector (form field is available for power users / curl)
- No per-user persistence of mode preference
- No change to the progress polling mechanism
