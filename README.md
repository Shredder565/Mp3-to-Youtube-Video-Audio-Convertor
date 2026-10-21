# MP3 to YouTube Video

A small Windows program that turns a batch of MP3s into YouTube-ready MP4s.
Pick any number of MP3 files, pick one image, click **Convert**. Each
`song.mp3` becomes `song.mp4` (1920x1080, H.264 + AAC 192 kbps, faststart)
showing your image for the whole track. Originals are never modified.

## Download (no install)

Download [`dist/MP3_to_YouTube_Video.zip`](dist/MP3_to_YouTube_Video.zip)
(click it, then **Download raw file**), unzip it, and double-click
**MP3 to YouTube Video.exe**. Keep `ffmpeg.exe` in the same folder.

`dist/` also has the bare `MP3 to YouTube Video.exe` if you already have
ffmpeg.exe.

> Windows SmartScreen may warn the first time because the exe is unsigned.
> Click **More info → Run anyway**.

## Using it

1. **Add MP3s...** (Ctrl/Shift-click to pick many) or **Add folder...**
2. **Browse...** for the image (JPG/PNG; 1920x1080 looks best, other sizes are letterboxed).
3. Optionally choose a **Save to** folder. Leave it blank to put each MP4 next to its MP3.
4. **Convert**. A progress bar tracks the batch; a summary pops up at the end.

**Fast mode** (on by default) encodes the still image at 2 fps instead of 30,
about 4x faster. Nothing is lost since the picture never changes, and YouTube
accepts it. Untick it for a conventional 30 fps file.

## Copyrighted music inside the recordings

Upload first, then let YouTube's Content ID find the songs. In YouTube Studio:
**Content → click the Copyright notice → View details → Take Action →
Trim out segment** (or *Erase song* / *Mute*). Each claim lists the exact
timestamps. See [YouTube's help page](https://support.google.com/youtube/answer/2902117).

## Building from source

Written in Go using [lxn/walk](https://github.com/lxn/walk) for the native Win32 UI.

- On Windows: install Go, then run `build.bat`.
- On Linux/macOS (cross-compile): run `./build.sh`.

Then place a Windows `ffmpeg.exe` (e.g. from
[BtbN/FFmpeg-Builds](https://github.com/BtbN/FFmpeg-Builds/releases)) next to the exe.

## How it encodes

```
ffmpeg -loop 1 -framerate 2 -i image.jpg -i song.mp3 \
  -vf "scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2,format=yuv420p" \
  -c:v libx264 -preset veryfast -tune stillimage -crf 20 \
  -c:a aac -b:a 192k -ar 44100 -shortest -movflags +faststart -r 2 song.mp4
```
