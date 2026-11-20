# Go Shred

A file-shredding function written in Go. It overwrites a file three times with random bytes before removing it. The implementation supports both text and binary files and uses only the Go standard library.

## Implementation

```go
func Shred(path string) error
```

The function validates the supplied path, opens the file and overwrites its contents using a 32 KiB buffer and random data from `crypto/rand`. Each pass starts at the beginning of the file and is synchronised using `file.Sync()`.

The file is closed and removed only after all three passes have completed successfully. Errors are returned to the caller.

## Running

Developed and tested on Ubuntu using Go 1.27.4.

```bash
git clone https://github.com/Smalera93/go-shred.git
cd go-shred
go run .
```

The demonstration programme creates and shreds a temporary test file, then verifies its removal.

## Testing

Run the automated tests:

```bash
go test -count=1 -v ./...
```

The project includes 12 automated tests covering:

- Regular, empty and large binary files.
- Missing files, directories and symbolic links.
- File contents, byte counts and seek positions across three overwrite passes.
- Insufficient random data.
- Seek and write failures.
- Incomplete writes.
- Synchronisation failures.

The overwrite logic is tested independently using a recording file interface and predetermined byte sequences.

Statement coverage recorded during development:

| Function | Coverage |
|---|---:|
| `Shred()` | 72.4% |
| `overwriteFile()` | 96.3% |
| `main()` | 0.0% |
| Overall | 58.0% |

Generate a coverage report using:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Advantages

- Fixed-size buffer keeps memory usage independent of file size.
- Supports arbitrary file contents.
- Uses cryptographically secure random data.
- Stops and returns an error when an operation fails.
- Separates file management from the overwrite logic, allowing independent testing.
- No external dependencies.

## Drawbacks and Limitations

- Three overwrite passes increase execution time and storage activity.
- Failed or interrupted operations may leave files partially or completely overwritten.
- Hard links may expose changes through other paths referencing the same file.
- File identity checks reduce but do not eliminate filesystem race conditions.
- Overwriting does not guarantee permanent data erasure on SSDs, copy-on-write filesystems or storage containing snapshots and backups.

The implementation assumes that the target file and its path are not being concurrently modified. It should not be treated as a guaranteed secure-erasure tool.

## Additional Test Considerations

The following scenarios could be covered by further testing:

- Insufficient file permissions.
- File closing and removal failures.
- Hard links and concurrent file replacement.
- Interrupted overwrite operations.
- File sizes immediately below, equal to and above the buffer size.
- Performance and memory usage when processing very large files.
