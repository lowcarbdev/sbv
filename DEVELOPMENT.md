## Development

- Go 1.25 or higher
- Node.js 22 or higher
- Air (for Go hot reload): `go install github.com/air-verse/air@latest`
- SQLite built with FTS5 support (for full-text search)
- libheif, for conversion of images
  - If not installed, HEIC images will display as placeholder images
- ffmpeg, for conversion of videos
  - If not installed, 3gp videos won't play

Build with `go build -tags "fts5 heic"`

To run the latest development version:
```bash
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data:/data \
  -e DB_PATH_PREFIX=/data \
  ghcr.io/lowcarbdev/sbv:latest
```

## Tests

Run all tests:
```bash
go test -tags "fts5 heic"
```

## Backend Setup

1. Install Go dependencies:
   ```bash
   go mod download
   ```

2. Run the backend with hot reload:
   ```bash
   air
   ```

   Or without hot reload:
   ```bash
   go run -tags "fts5 heic" .
   ```

The backend will start on `http://localhost:8081`

## Frontend Setup

1. Navigate to the frontend directory:
   ```bash
   cd frontend
   ```

2. Install npm dependencies:
   ```bash
   npm install
   ```

3. Run the development server:
   ```bash
   npm run dev
   ```

The frontend will start on `http://localhost:5173`

## Backend Hot Reload

The backend uses [Air](https://github.com/air-verse/air) for hot reload. When you save a `.go` file, Air will automatically rebuild and restart the server.

## Frontend Hot Reload

The frontend uses Vite's built-in hot module replacement (HMR). Changes to React components will be reflected instantly without a full page reload.

# Usage

1. **Start both servers** - Run the backend and frontend development servers
2. **Open the app** - Navigate to `http://localhost:5173` in your browser
3. **Upload backup** - Click "Upload Backup" and select your XML file
4. **Browse messages** - Click on conversations to view message threads
5. **Filter by date** - Use the date pickers to filter messages by date range
