# Upload Mode Selection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `tempfile` / `pipe` upload processing mode selectable via CLI flag, env var, or per-request form field.

**Architecture:** A package-level default mode in `internal/parser.go` is set at startup from CLI flag / env var. `HandleUpload` reads an optional `upload_mode` form field to override the default per-request, then branches: `tempfile` uses the existing temp-file path unchanged; `pipe` creates an `io.Pipe`, pumps the multipart reader into the writer in one goroutine, and calls the new `ProcessUploadedFileFromReader` from a second goroutine.

**Tech Stack:** Go stdlib (`io`, `os`), Echo v4, existing `ParseSMSBackupStreaming(io.Reader)` API.

## Global Constraints

- Go module: `github.com/lowcarbdev/sbv`
- Tests live in `internal/` as `package internal` (same package, white-box)
- Run tests with: `go test ./internal/...`
- Valid mode strings: `"tempfile"` and `"pipe"` (lowercase, exact)
- Default mode when nothing is configured: `"tempfile"`
- Invalid values at any config level are silently skipped; next level is tried
- Invalid CLI/env value exits the process with a clear error message (same pattern as `--blob-storage`)

---

### Task 1: Add mode accessors and `ProcessUploadedFileFromReader` to `internal/parser.go`

**Files:**
- Modify: `internal/parser.go` (after line 789, end of file)
- Test: `internal/parser_test.go`

**Interfaces:**
- Produces:
  - `SetDefaultUploadMode(mode string)` — sets package-level default
  - `GetDefaultUploadMode() string` — returns current default
  - `ProcessUploadedFileFromReader(userID, username string, r io.Reader)` — same contract as `ProcessUploadedFile` but reads from `r` directly; no temp file created or deleted

- [ ] **Step 1: Write failing tests**

Add to `internal/parser_test.go`:

```go
func TestDefaultUploadMode(t *testing.T) {
	// Default should be tempfile
	if got := GetDefaultUploadMode(); got != "tempfile" {
		t.Errorf("expected default 'tempfile', got %q", got)
	}
}

func TestSetDefaultUploadMode(t *testing.T) {
	original := GetDefaultUploadMode()
	defer SetDefaultUploadMode(original)

	SetDefaultUploadMode("pipe")
	if got := GetDefaultUploadMode(); got != "pipe" {
		t.Errorf("expected 'pipe', got %q", got)
	}

	SetDefaultUploadMode("tempfile")
	if got := GetDefaultUploadMode(); got != "tempfile" {
		t.Errorf("expected 'tempfile', got %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/... -run TestDefaultUploadMode -v
go test ./internal/... -run TestSetDefaultUploadMode -v
```

Expected: FAIL with `undefined: GetDefaultUploadMode`

- [ ] **Step 3: Add mode var and accessors to `internal/parser.go`**

Add after the existing package-level vars (near `uploadProgress` / `uploadProgressLock`, around line 619):

```go
var defaultUploadMode = "tempfile"

func SetDefaultUploadMode(mode string) {
	defaultUploadMode = mode
}

func GetDefaultUploadMode() string {
	return defaultUploadMode
}
```

- [ ] **Step 4: Run accessor tests to verify they pass**

```bash
go test ./internal/... -run TestDefaultUploadMode -v
go test ./internal/... -run TestSetDefaultUploadMode -v
```

Expected: PASS

- [ ] **Step 5: Write failing test for `ProcessUploadedFileFromReader`**

Add to `internal/parser_test.go`. This test uses an in-memory SQLite DB the same way other parser tests do — look at `TestSampleXMLParsing` for the XML fixture. Because `ProcessUploadedFileFromReader` runs asynchronously (it's called in a goroutine by the handler), we call it directly and synchronously here.

```go
func TestProcessUploadedFileFromReader(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	if err := InitUserDB("test-user", dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	r := strings.NewReader(sampleXML)
	// Call directly (not in goroutine) so we can observe results synchronously
	processUploadedFileFromReaderSync("test-user", "testuser", r, db)

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count); err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 messages, got %d", count)
	}
}
```

Note: `processUploadedFileFromReaderSync` is an unexported helper we'll add that accepts a `*sql.DB` directly — it lets us test the core logic without the `GetUserDB` lookup. `ProcessUploadedFileFromReader` (exported, no DB param) is what the handler calls.

Also add `"path/filepath"` and `"database/sql"` to imports in `parser_test.go` if not already present.

- [ ] **Step 6: Run test to verify it fails**

```bash
go test ./internal/... -run TestProcessUploadedFileFromReader -v
```

Expected: FAIL with `undefined: processUploadedFileFromReaderSync`

- [ ] **Step 7: Add `processUploadedFileFromReaderSync` and `ProcessUploadedFileFromReader` to `internal/parser.go`**

Add after `ProcessUploadedFile` (after line 789):

```go
// processUploadedFileFromReaderSync is the testable core: parses r into userDB.
func processUploadedFileFromReaderSync(userID, username string, r io.Reader, userDB *sql.DB) {
	slog.Info("Starting pipe-mode processing", "user", username)

	messageCount, callCount, err := ParseSMSBackupStreaming(userDB, userID, r, 1)
	if err != nil {
		slog.Error("Error processing file", "error", err)
		SetUploadProgress(0, 0, "error")
		uploadProgressLock.Lock()
		if uploadProgress != nil {
			uploadProgress.mu.Lock()
			uploadProgress.ErrorMessage = fmt.Sprintf("Failed to process file: %v", err)
			uploadProgress.mu.Unlock()
		}
		uploadProgressLock.Unlock()
		return
	}

	slog.Info("Completed pipe-mode processing", "messages", messageCount, "calls", callCount)
}

// ProcessUploadedFileFromReader processes r in the background without writing a temp file.
// Intended to be called in a goroutine; the caller is responsible for closing r on error.
func ProcessUploadedFileFromReader(userID, username string, r io.Reader) {
	slog.Info("Starting background pipe-mode processing", "user", username)

	userDB, err := GetUserDB(userID, username)
	if err != nil {
		slog.Error("Error getting user database", "error", err)
		SetUploadProgress(0, 0, "error")
		uploadProgressLock.Lock()
		if uploadProgress != nil {
			uploadProgress.mu.Lock()
			uploadProgress.ErrorMessage = fmt.Sprintf("Failed to get user database: %v", err)
			uploadProgress.mu.Unlock()
		}
		uploadProgressLock.Unlock()
		return
	}

	processUploadedFileFromReaderSync(userID, username, r, userDB)
}
```

- [ ] **Step 8: Run all parser tests**

```bash
go test ./internal/... -run TestProcessUploadedFileFromReader -v
go test ./internal/... -run TestDefaultUploadMode -v
go test ./internal/... -run TestSetDefaultUploadMode -v
```

Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add internal/parser.go internal/parser_test.go
git commit -m "feat: add upload mode accessors and ProcessUploadedFileFromReader"
```

---

### Task 2: Wire `--upload-mode` flag and `UPLOAD_MODE` env var in `main.go`

**Files:**
- Modify: `main.go`

**Interfaces:**
- Consumes: `internal.SetDefaultUploadMode(string)` from Task 1
- Produces: server starts with `defaultUploadMode` set to the resolved value

- [ ] **Step 1: Add flag, resolve, validate, and call `SetDefaultUploadMode`**

In `main.go`, after the `blobDir` flag (around line 23), add:

```go
uploadMode := flag.String("upload-mode", "", "Upload processing mode: 'tempfile' (default) or 'pipe'")
```

After the `blobDir` resolution block and before `internal.UseWALMode = ...`, add:

```go
resolvedUploadMode := *uploadMode
if resolvedUploadMode == "" {
    resolvedUploadMode = os.Getenv("UPLOAD_MODE")
}
if resolvedUploadMode == "" {
    resolvedUploadMode = "tempfile"
}
if resolvedUploadMode != "tempfile" && resolvedUploadMode != "pipe" {
    fmt.Fprintf(os.Stderr, "invalid --upload-mode value %q: must be 'tempfile' or 'pipe'\n", resolvedUploadMode)
    os.Exit(1)
}
internal.SetDefaultUploadMode(resolvedUploadMode)
```

After the blob storage logger calls, add:

```go
logger.Info("Upload mode", "mode", resolvedUploadMode)
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
go build ./...
```

Expected: exits 0, no output.

- [ ] **Step 3: Smoke-test flag parsing**

```bash
./sbv --upload-mode=pipe --help 2>&1 | grep upload-mode || true
go run . --upload-mode=invalid 2>&1 | grep "invalid --upload-mode"
```

Expected second line: prints `invalid --upload-mode value "invalid": must be 'tempfile' or 'pipe'`

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat: add --upload-mode flag and UPLOAD_MODE env var"
```

---

### Task 3: Branch on mode in `HandleUpload` in `internal/handlers.go`

**Files:**
- Modify: `internal/handlers.go`

**Interfaces:**
- Consumes:
  - `GetDefaultUploadMode() string` from Task 1
  - `ProcessUploadedFileFromReader(userID, username string, r io.Reader)` from Task 1
  - `SaveUploadedFile` and `ProcessUploadedFile` — unchanged, still used for `tempfile` mode

- [ ] **Step 1: Write a failing handler test for pipe mode**

Add to `internal/handlers_test.go`. Look at the existing upload test setup in that file for the multipart form builder pattern. Add:

```go
func TestHandleUploadPipeMode(t *testing.T) {
	// Arrange: set pipe mode as default, restore after
	SetDefaultUploadMode("pipe")
	defer SetDefaultUploadMode("tempfile")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "backup.xml")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = io.WriteString(part, sampleXML)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)
	c.Set("user_id", testUserID)
	c.Set("username", "testuser")

	err = HandleUpload(c)
	if err != nil {
		t.Fatalf("HandleUpload: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp UploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success=true, got false: %s", resp.Error)
	}
	if !resp.Processing {
		t.Errorf("expected processing=true")
	}
}
```

Check what `testUserID` and `testUsername` constants are called in `handlers_test.go` and use those exact names.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/... -run TestHandleUploadPipeMode -v
```

Expected: FAIL (pipe branch not yet implemented; likely falls through to tempfile path or compile error).

- [ ] **Step 3: Rewrite `HandleUpload` to branch on mode**

Replace the body of `HandleUpload` in `internal/handlers.go` from after the `defer file.Close()` line to the end of the function:

```go
	slog.Info("Receiving file", "filename", header.Filename, "size", header.Size)

	// Get user ID and username from context early — needed by both paths
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, UploadResponse{
			Success: false,
			Error:   "User not authenticated",
		})
	}
	username, ok := c.Get("username").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, UploadResponse{
			Success: false,
			Error:   "User not authenticated",
		})
	}

	// Resolve effective mode: form field overrides server default
	effectiveMode := GetDefaultUploadMode()
	if formMode := c.Request().FormValue("upload_mode"); formMode == "pipe" || formMode == "tempfile" {
		effectiveMode = formMode
	}
	slog.Info("Upload mode", "mode", effectiveMode, "filename", header.Filename)

	if effectiveMode == "pipe" {
		pr, pw := io.Pipe()
		go func() {
			_, err := io.Copy(pw, file)
			pw.CloseWithError(err)
		}()
		go ProcessUploadedFileFromReader(userID, username, pr)
	} else {
		tempFilePath, err := SaveUploadedFile(file, header.Filename)
		if err != nil {
			slog.Error("Error saving file", "error", err)
			return c.JSON(http.StatusInternalServerError, UploadResponse{
				Success: false,
				Error:   "Failed to save uploaded file: " + err.Error(),
			})
		}
		slog.Info("File saved", "path", tempFilePath)
		go ProcessUploadedFile(userID, username, tempFilePath)
	}

	return c.JSON(http.StatusOK, UploadResponse{
		Success:      true,
		MessageCount: 0,
		CallLogCount: 0,
		Processing:   true,
	})
```

Also add `"io"` to the imports in `handlers.go` if not already present.

- [ ] **Step 4: Run all tests**

```bash
go test ./internal/... -v 2>&1 | tail -30
```

Expected: all PASS, no FAIL lines.

- [ ] **Step 5: Verify form field override works by adding a quick test**

Add to `internal/handlers_test.go`:

```go
func TestHandleUploadFormFieldOverride(t *testing.T) {
	// Server default is tempfile; form field requests pipe
	SetDefaultUploadMode("tempfile")
	defer SetDefaultUploadMode("tempfile")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("upload_mode", "pipe")
	part, _ := writer.CreateFormFile("file", "backup.xml")
	_, _ = io.WriteString(part, sampleXML)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)
	c.Set("user_id", testUserID)
	c.Set("username", "testuser")

	if err := HandleUpload(c); err != nil {
		t.Fatalf("HandleUpload: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
```

```bash
go test ./internal/... -run TestHandleUploadFormFieldOverride -v
```

Expected: PASS

- [ ] **Step 6: Run full test suite one final time**

```bash
go test ./internal/... -count=1
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/handlers.go internal/handlers_test.go
git commit -m "feat: branch HandleUpload on pipe vs tempfile mode with form field override"
```
