# SMS Backup Viewer (SBV)
#
# The fts5 build tag is required for both building and testing: without it,
# go-sqlite3 compiles without the FTS5 module and anything touching the
# database fails with "no such module: fts5".

.PHONY: build build-heic test vet clean

build:
	./build.sh

build-heic:
	./build.sh heic

# Scoped to the module's own packages: ./... would descend into Go code
# shipped inside frontend/node_modules
test:
	go test -tags fts5 . ./internal/...

vet:
	go vet -tags fts5 . ./internal/...

clean:
	rm -f sbv
