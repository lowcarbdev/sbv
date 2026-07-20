package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// BlobStore abstracts where media blobs are persisted.
type BlobStore interface {
	// Write persists data and returns a relative file path, or "" if stored inline.
	Write(data []byte, mediaType string) (filePath string, err error)
	// Read retrieves data by the relative file path returned by Write.
	Read(filePath string) (data []byte, mediaType string, err error)
}

// globalBlobStoreMu guards GlobalBlobStore for concurrent access.
var globalBlobStoreMu sync.RWMutex

// GlobalBlobStore is the shared store for all users.
var GlobalBlobStore BlobStore = &DBBlobStore{}

// GetGlobalBlobStore returns the current global BlobStore safely.
func GetGlobalBlobStore() BlobStore {
	globalBlobStoreMu.RLock()
	defer globalBlobStoreMu.RUnlock()
	return GlobalBlobStore
}

// SetGlobalBlobStore sets the global BlobStore safely.
func SetGlobalBlobStore(s BlobStore) {
	globalBlobStoreMu.Lock()
	defer globalBlobStoreMu.Unlock()
	GlobalBlobStore = s
}

// DBBlobStore is a no-op: blobs are stored inline in media_data.
type DBBlobStore struct{}

func (d *DBBlobStore) Write(_ []byte, _ string) (string, error) { return "", nil }
func (d *DBBlobStore) Read(_ string) ([]byte, string, error)    { return nil, "", nil }

var mimeToExt = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"image/heic":      ".heic",
	"image/heif":      ".heif",
	"video/mp4":       ".mp4",
	"video/3gpp":      ".3gp",
	"video/quicktime": ".mov",
	"audio/mpeg":      ".mp3",
	"audio/amr":       ".amr",
	"audio/ogg":       ".ogg",
	"audio/aac":       ".aac",
	"text/vcard":      ".vcf",
	"text/x-vcard":    ".vcf",
}

var extToMime = map[string]string{
	".jpg":  "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".heic": "image/heic",
	".heif": "image/heif",
	".mp4":  "video/mp4",
	".3gp":  "video/3gpp",
	".mov":  "video/quicktime",
	".mp3":  "audio/mpeg",
	".amr":  "audio/amr",
	".ogg":  "audio/ogg",
	".aac":  "audio/aac",
	".vcf":  "text/vcard",
}

var safeExt = regexp.MustCompile(`[^a-z0-9._-]`)

func mediaTypeToExt(mediaType string) string {
	if mediaType == "" {
		return ".bin"
	}
	// Strip parameters (e.g. "image/jpeg; charset=utf-8" -> "image/jpeg")
	base := strings.ToLower(strings.SplitN(mediaType, ";", 2)[0])
	base = strings.TrimSpace(base)
	if ext, ok := mimeToExt[base]; ok {
		return ext
	}
	// Derive from subtype
	parts := strings.SplitN(base, "/", 2)
	if len(parts) != 2 || parts[1] == "" {
		return ".bin"
	}
	ext := "." + safeExt.ReplaceAllString(parts[1], "")
	if ext == "." {
		return ".bin"
	}
	return ext
}

func extToMediaType(ext string) string {
	// Accept both ".jpg" and "path/to/file.jpg"
	if !strings.HasPrefix(ext, ".") {
		ext = filepath.Ext(ext)
	}
	ext = strings.ToLower(ext)
	if mt, ok := extToMime[ext]; ok {
		return mt
	}
	return "application/octet-stream"
}

// DiskBlobStore stores blobs as <sha256-hash>.<ext> files under baseDir.
type DiskBlobStore struct {
	baseDir string
}

// NewDiskBlobStore creates a DiskBlobStore using sha256 for content-addressable filenames.
func NewDiskBlobStore(baseDir string) *DiskBlobStore {
	return &DiskBlobStore{baseDir: baseDir}
}

func (d *DiskBlobStore) Write(data []byte, mediaType string) (string, error) {
	if err := os.MkdirAll(d.baseDir, 0755); err != nil {
		return "", fmt.Errorf("blobstore: mkdir %s: %w", d.baseDir, err)
	}

	h := sha256.New()
	h.Write(data)
	hexHash := hex.EncodeToString(h.Sum(nil))
	ext := mediaTypeToExt(mediaType)
	filename := hexHash + ext
	fullPath := filepath.Join(d.baseDir, filename)

	// Dedup: if file already exists, skip write
	if _, err := os.Stat(fullPath); err == nil {
		return filename, nil
	}

	// Write atomically: temp file in same dir, then rename
	tmp, err := os.CreateTemp(d.baseDir, "blob-*.tmp")
	if err != nil {
		return "", fmt.Errorf("blobstore: create temp: %w", err)
	}
	tmpName := tmp.Name()

	_, writeErr := tmp.Write(data)
	closeErr := tmp.Close()
	if writeErr != nil || closeErr != nil {
		os.Remove(tmpName)
		if writeErr != nil {
			return "", fmt.Errorf("blobstore: write temp: %w", writeErr)
		}
		return "", fmt.Errorf("blobstore: close temp: %w", closeErr)
	}

	if err := os.Rename(tmpName, fullPath); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("blobstore: rename to %s: %w", fullPath, err)
	}

	slog.Debug("BlobStore wrote file", "path", fullPath, "size", len(data))
	return filename, nil
}

// SetModTime sets the modification time of a blob file to match the message timestamp.
func (d *DiskBlobStore) SetModTime(filePath string, t time.Time) error {
	return os.Chtimes(filepath.Join(d.baseDir, filePath), t, t)
}

func (d *DiskBlobStore) Read(filePath string) ([]byte, string, error) {
	fullPath := filepath.Join(d.baseDir, filePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, "", fmt.Errorf("blobstore: read %s: %w", fullPath, err)
	}
	ext := filepath.Ext(filePath)
	mediaType := extToMediaType(ext)
	return data, mediaType, nil
}

// GetUserBlobStore returns the shared BlobStore for all users.
func GetUserBlobStore(_ string) BlobStore {
	return GetGlobalBlobStore()
}
