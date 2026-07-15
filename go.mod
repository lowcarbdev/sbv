module github.com/lowcarbdev/sbv

go 1.25.7

require (
	github.com/mattn/go-sqlite3 v1.14.48
	golang.org/x/time v0.15.0
)

require (
	github.com/google/uuid v1.6.0
	github.com/labstack/echo/v4 v4.15.4
	// Fork of github.com/strukturag/libheif-go with a fix for building
	// against libheif >= 1.19 (e.g. Alpine >= 3.21); switch back once
	// https://github.com/strukturag/libheif-go/pull/TODO is merged upstream.
	github.com/lowcarbdev/libheif-go v0.0.0-20260714060915-7cdd11ec893b
	golang.org/x/crypto v0.54.0
	golang.org/x/term v0.45.0
)

require (
	github.com/labstack/gommon v0.5.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)
