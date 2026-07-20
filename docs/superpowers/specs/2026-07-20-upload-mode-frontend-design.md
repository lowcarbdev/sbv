# Upload Mode Frontend Selector Design

**Date:** 2026-07-20  
**Status:** Approved

## Summary

Add two inline radio buttons to the Upload modal (`Upload.jsx`) so users can choose between `tempfile` (default) and `pipe` upload mode. The selected value is appended to the multipart form data on each upload, using the existing backend `upload_mode` form field.

## UI

Below the drop zone / file list, add a `Form.Group` with two `Form.Check` inline radio buttons:

```
Upload mode
● Standard (temp file)   ○ Streaming (pipe)
```

- Default selection: `tempfile`
- Both radios disabled while `uploading === true`
- Visible at all times (not collapsed, not hidden)

## State

Add one new piece of local state:

```js
const [uploadMode, setUploadMode] = useState('tempfile')
```

State is local to the modal — resets to `'tempfile'` each time the modal is opened.

## Data Flow

In `uploadSingleFile`, append the mode to the form data before the POST:

```js
formData.append('upload_mode', uploadMode)
```

The backend already reads `upload_mode` from the multipart form and routes accordingly (see `internal/handlers.go`).

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/components/Upload.jsx` | Add `uploadMode` state, radio buttons UI, `formData.append` |

## Non-Goals

- No persistence of the user's mode preference across sessions
- No explanation or help text for the two modes
- No server round-trip to fetch the server default
