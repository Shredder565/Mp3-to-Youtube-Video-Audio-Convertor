@echo off
rem Build the Windows GUI exe on Windows (needs Go 1.21+ from https://go.dev/dl/).
set GOFLAGS=-mod=mod
go run github.com/akavel/rsrc -manifest app.manifest -o rsrc.syso
go build -ldflags="-s -w -H windowsgui" -o "MP3 to YouTube Video.exe" .
echo Built: "MP3 to YouTube Video.exe"  (put ffmpeg.exe next to it)
pause
