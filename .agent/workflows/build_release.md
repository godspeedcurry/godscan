---
description: Build godscan for all supported platforms (Mac, Linux, Windows)
---
1. Clean previous builds
// turbo
rm -f godscan_linux_amd64 godscan_windows_amd64.exe godscan_darwin_amd64 godscan_darwin_arm64

2. Build for Linux AMD64
// turbo
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o godscan_linux_amd64

3. Build for Windows AMD64
// turbo
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o godscan_windows_amd64.exe

4. Build for MacOS AMD64
// turbo
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o godscan_darwin_amd64

5. Build for MacOS ARM64
// turbo
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o godscan_darwin_arm64

6. List generated binaries
ls -lh godscan_*
