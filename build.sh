#!/bin/sh
# Cross-compile the Windows GUI exe from Linux/macOS (needs Go 1.21+).
set -e
export GOFLAGS=-mod=mod
go run github.com/akavel/rsrc -manifest app.manifest -o rsrc.syso
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H windowsgui" -o "MP3 to YouTube Video.exe" .
echo "Built: MP3 to YouTube Video.exe  (put ffmpeg.exe next to it)"
