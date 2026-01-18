# SMS Backup Viewer (SBV) - Technical Specification

A privacy-focused, self-hosted web application for viewing SMS, MMS, and call log backups from the Android "SMS Backup & Restore" application.

## Overview

SBV provides a modern conversation-based interface for browsing message backups while keeping all data local. Key design principles:

- **Privacy-first**: No telemetry, no remote servers, 100% self-hosted
- **Per-user isolation**: Each user has their own SQLite database
- **Familiar UI**: Conversation threading similar to native messaging apps
- **Media support**: Inline display of images, videos, and contact cards

## Technology Stack

### Backend
- **Go 1.25** with Echo v4 web framework
- **SQLite** with FTS5 for full-text search
- **bcrypt** for password hashing
- **libheif** (optional) for HEIC image conversion
- **ffmpeg** for 3GP video support

### Frontend
- **React 19** with React Router 7
- **Vite 7** for building
- **Bootstrap 5** / React Bootstrap for styling
- **Axios** for API communication

### Deployment
- **Docker** with multi-stage Alpine-based builds
- **Docker Compose** for easy deployment
- **GitHub Actions** for CI/CD to GitHub Container Registry

## Project Structure

```
sbv/
├── main.go                    # Entry point, route setup, server config
├── internal/                  # Core backend logic
│   ├── auth.go                # User/session management
│   ├── auth_handlers.go       # Auth API endpoints
│   ├── database.go            # SQLite initialization, queries
│   ├── handlers.go            # Message/call API endpoints
│   ├── parser.go              # XML backup file parsing
│   ├── models.go              # Data structures
│   ├── middleware.go          # Auth middleware
│   ├── settings.go            # User preferences
│   ├── heic_enabled.go        # HEIC conversion (with libheif)
│   └── heic_disabled.go       # HEIC fallback (without libheif)
├── frontend/
│   ├── src/
│   │   ├── App.jsx            # Main app with routing
│   │   ├── components/        # React components
│   │   │   ├── ConversationList.jsx   # Contact/conversation list
│   │   │   ├── MessageThread.jsx      # Message display
│   │   │   ├── Calls.jsx              # Call log view
│   │   │   ├── Search.jsx             # Full-text search
│   │   │   ├── Upload.jsx             # XML file upload
│   │   │   ├── Activity.jsx           # Timeline view
│   │   │   ├── LazyMedia.jsx          # Lazy-loaded media
│   │   │   ├── MediaGrid.jsx          # Media gallery
│   │   │   └── ...
│   │   └── contexts/
│   │       └── AuthContext.jsx        # Auth state management
│   └── dist/                  # Production build output
├── Dockerfile                 # Multi-stage build
├── docker-compose.yaml        # Deployment config
└── build.sh                   # Local build script
```

## Architecture

### Authentication Flow

1. User registers with username/password (bcrypt hashed)
2. Login creates a session stored in the auth database
3. Session ID stored in httpOnly cookie (`sbv_session`)
4. Middleware validates session on protected routes
5. Each user gets a unique database file: `sbv_[user-uuid].db`

### Data Model

**Auth Database (`sbv.db`)**:
- `users` - User accounts with hashed passwords
- `sessions` - Active sessions with expiration
- `settings` - Per-user JSON preferences

**Per-User Database (`sbv_[uuid].db`)**:
- `messages` - Unified table for SMS, MMS, and calls
  - `record_type`: 1=SMS, 2=MMS, 3=Call
  - `type`: Message direction (1=received, 2=sent, etc.)
  - `media_data`: BLOB storage for attachments
- `messages_fts` - FTS5 virtual table for search

### Message Import Pipeline

1. User uploads XML file via `/api/upload`
2. Server saves to temp file, returns immediately
3. Parser reads XML incrementally:
   - SMS messages parsed with metadata
   - MMS messages parsed with parts/attachments
   - Call logs parsed with duration/type
4. Records inserted with unique constraint (idempotent)
5. Client polls `/api/progress` for status

### Media Handling

Media is stored as base64-encoded BLOBs in the `messages` table:
- **Images**: JPEG, PNG, GIF, WebP, HEIC (converted to JPEG)
- **Videos**: MP4, 3GP (converted to MP4 via ffmpeg)
- **vCards**: Parsed and rendered as contact previews

The frontend uses lazy loading (`LazyMedia.jsx`) to fetch media on-demand.

## API Reference

### Authentication (Public)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/register` | Create account |
| POST | `/api/auth/login` | Login, sets cookie |
| POST | `/api/auth/logout` | Logout, clears cookie |

### User (Protected)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/auth/me` | Current user info |
| POST | `/api/auth/change-password` | Update password |
| GET | `/api/settings` | Get user settings |
| PUT | `/api/settings` | Update user settings |

### Messages & Data (Protected)

| Method | Endpoint | Query Params | Description |
|--------|----------|--------------|-------------|
| GET | `/api/conversations` | `start_date`, `end_date` | List conversations |
| GET | `/api/messages` | `address`, `start_date`, `end_date`, `limit`, `offset` | Messages for conversation |
| GET | `/api/activity` | `start_date`, `end_date`, `limit`, `offset` | Timeline of messages + calls |
| GET | `/api/calls` | `start_date`, `end_date` | Call log |
| GET | `/api/search` | `q`, `start_date`, `end_date` | Full-text search |
| GET | `/api/media` | `address` | All media for conversation |
| GET | `/api/media-items` | `address` | Media items only (no data) |
| GET | `/api/daterange` | - | Min/max dates in database |

### Upload (Protected)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/upload` | Upload XML backup (async) |
| GET | `/api/progress` | Check upload status |

### System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/health` | Health check |
| GET | `/api/version` | App version |

## Database Schema

### messages table

```sql
CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    record_type INTEGER NOT NULL,        -- 1=SMS, 2=MMS, 3=Call
    address TEXT,                        -- Phone number
    body TEXT,                           -- Message text
    type INTEGER,                        -- Direction (1=recv, 2=sent)
    date INTEGER,                        -- Unix timestamp (ms)
    read INTEGER,                        -- Read status
    thread_id INTEGER,                   -- Conversation thread
    contact_name TEXT,                   -- Resolved contact name

    -- MMS-specific
    msg_id TEXT,                         -- MMS message ID
    m_type INTEGER,                      -- MMS type
    media_type TEXT,                     -- MIME type
    media_data BLOB,                     -- Binary media

    -- Call-specific
    duration INTEGER,                    -- Call duration (seconds)

    -- Additional metadata
    service_center TEXT,
    protocol INTEGER,
    subject TEXT,
    ...
);

-- Key indexes
CREATE UNIQUE INDEX idx_message_unique ON messages(...);
CREATE INDEX idx_address ON messages(address);
CREATE INDEX idx_date ON messages(date);
CREATE INDEX idx_thread ON messages(thread_id);
CREATE INDEX idx_record_type ON messages(record_type);

-- Full-text search
CREATE VIRTUAL TABLE messages_fts USING fts5(
    body, address, contact_name,
    content='messages', content_rowid='id'
);
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8081` | Server port |
| `DB_PATH_PREFIX` | `.` | Database directory |
| `PUID` | `1000` | Docker user ID |
| `PGID` | `1000` | Docker group ID |

### Build Tags

| Tag | Description |
|-----|-------------|
| `fts5` | Enable full-text search (always used) |
| `heic` | Enable HEIC image conversion (requires libheif) |

### User Settings

Stored per-user as JSON:
```json
{
  "conversations": {
    "show_calls": true
  }
}
```

## Build & Development

### Local Development

```bash
# Backend (with hot reload)
air

# Frontend (with hot reload)
cd frontend && npm run dev
```

Frontend dev server runs on `:5173` and proxies API calls to `:8081`.

### Production Build

```bash
# With HEIC support
./build.sh

# Without HEIC
go build -tags fts5 -o sbv .

# Frontend
cd frontend && npm run build
```

### Docker

```bash
docker compose up -d
```

The multi-stage Dockerfile:
1. Builds frontend with Node.js 22
2. Compiles Go binary with FTS5 + HEIC
3. Creates minimal Alpine runtime with ffmpeg/libheif

## Security Considerations

- **httpOnly cookies**: Session tokens not accessible to JavaScript
- **SameSite=Lax**: CSRF protection
- **bcrypt hashing**: Secure password storage
- **Per-user isolation**: Users cannot access other users' data
- **No internet exposure recommended**: Designed for local/trusted networks

## Known Limitations

1. **Large imports**: Processing slows with many media attachments
2. **MMS group sender**: Contact names unavailable (XML limitation)
3. **100k message limit**: Use date filters for older conversations
4. **Android only**: Requires SMS Backup & Restore app (no iOS support)
