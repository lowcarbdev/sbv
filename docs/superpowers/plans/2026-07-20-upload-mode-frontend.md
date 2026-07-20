# Upload Mode Frontend Selector Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two inline radio buttons to the Upload modal so users can select `tempfile` or `pipe` upload mode, which is sent as the `upload_mode` form field on each upload.

**Architecture:** Single file change to `frontend/src/components/Upload.jsx`. Add one new `uploadMode` state variable defaulted to `'tempfile'`, render two `Form.Check` radio buttons below the file list, and append the value to `FormData` in `uploadSingleFile`. No new files, no new dependencies.

**Tech Stack:** React 19, React Bootstrap 2, Vite 7.

## Global Constraints

- Radio button values must be exactly `"tempfile"` and `"pipe"` (lowercase, verbatim — these are the strings the backend accepts)
- Default selection: `"tempfile"`
- Both radios must be disabled when `uploading === true`
- Radios appear below the drop zone / file list, always visible
- No new files, no new dependencies

---

### Task 1: Add upload mode radio buttons to Upload.jsx

**Files:**
- Modify: `frontend/src/components/Upload.jsx`

**Interfaces:**
- Consumes: existing `uploading` state (boolean), existing `uploadSingleFile` function, existing `FormData` construction in `uploadSingleFile`
- Produces: `uploadMode` state (`'tempfile'` | `'pipe'`), radio UI, `upload_mode` form field on each POST

- [ ] **Step 1: Add `uploadMode` state**

In `Upload.jsx`, add the new state variable alongside the existing `useState` declarations (around line 8–17):

```jsx
const [uploadMode, setUploadMode] = useState('tempfile')
```

- [ ] **Step 2: Append `upload_mode` to FormData in `uploadSingleFile`**

In `uploadSingleFile` (around line 116–117), update the FormData construction:

```jsx
const uploadSingleFile = async (file) => {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('upload_mode', uploadMode)
  // ... rest unchanged
```

- [ ] **Step 3: Add the radio buttons to the JSX**

After the file list block (the closing `</Form.Group>` tag around line 272) and before the closing `</div>` of the `mb-3` wrapper (around line 273), add:

```jsx
          <Form.Group className="mt-3">
            <Form.Label className="small text-muted fw-semibold mb-1">Upload mode</Form.Label>
            <div className="d-flex gap-3">
              <Form.Check
                type="radio"
                id="mode-tempfile"
                name="uploadMode"
                label="Standard (temp file)"
                value="tempfile"
                checked={uploadMode === 'tempfile'}
                onChange={() => setUploadMode('tempfile')}
                disabled={uploading}
              />
              <Form.Check
                type="radio"
                id="mode-pipe"
                name="uploadMode"
                label="Streaming (pipe)"
                value="pipe"
                checked={uploadMode === 'pipe'}
                onChange={() => setUploadMode('pipe')}
                disabled={uploading}
              />
            </div>
          </Form.Group>
```

- [ ] **Step 4: Lint**

```bash
cd /path/to/project/frontend && npm run lint
```

Expected: exits 0, no errors.

- [ ] **Step 5: Manual smoke test**

Start the dev server:
```bash
cd frontend && npm run dev
```

Open the app in a browser, trigger the Upload modal, and verify:
- Two radio buttons appear below the file picker: "Standard (temp file)" (selected) and "Streaming (pipe)"
- Clicking "Streaming (pipe)" selects it
- Both radios become disabled while an upload is in progress
- Selecting a file and uploading with "Streaming (pipe)" selected sends `upload_mode=pipe` in the form data (verify in browser DevTools → Network → the upload POST request payload)

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/Upload.jsx
git commit -m "feat: add upload mode radio buttons to upload modal"
```
