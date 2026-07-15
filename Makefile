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

test:
	go test -tags fts5 ./...

vet:
	go vet -tags fts5 ./...

clean:
	rm -f sbv
