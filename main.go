// MP3 to YouTube Video - native Windows GUI.
// Picks MP3 files + one image, writes a YouTube-ready MP4 per MP3 with the same name.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

const appTitle = "MP3 to YouTube Video"

type app struct {
	mw        *walk.MainWindow
	list      *walk.ListBox
	listModel *fileModel
	imgEdit   *walk.LineEdit
	outEdit   *walk.LineEdit
	fastCB    *walk.CheckBox
	progress  *walk.ProgressBar
	status    *walk.Label
	convert   *walk.PushButton
	cancelBtn *walk.PushButton
	files     []string
	cancel    bool
	running   bool
}

type fileModel struct {
	walk.ListModelBase
	items []string
}

func (m *fileModel) ItemCount() int         { return len(m.items) }
func (m *fileModel) Value(i int) interface{} { return filepath.Base(m.items[i]) }

func findFFmpeg() string {
	exe, _ := os.Executable()
	cands := []string{filepath.Join(filepath.Dir(exe), "ffmpeg.exe"), "ffmpeg.exe"}
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	return ""
}

func ffmpegArgs(mp3, img, out string, fast bool) []string {
	fps := "30"
	if fast {
		fps = "2"
	}
	return []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-loop", "1", "-framerate", fps, "-i", img,
		"-i", mp3,
		"-vf", "scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2,format=yuv420p",
		"-c:v", "libx264", "-preset", "veryfast", "-tune", "stillimage", "-crf", "20",
		"-c:a", "aac", "-b:a", "192k", "-ar", "44100",
		"-shortest", "-movflags", "+faststart", "-r", fps,
		out,
	}
}

func main() {
	a := &app{listModel: &fileModel{}}

	err := MainWindow{
		AssignTo: &a.mw,
		Title:    appTitle,
		MinSize:  Size{Width: 640, Height: 480},
		Size:     Size{Width: 760, Height: 560},
		Layout:   VBox{Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}},
		Children: []Widget{
			GroupBox{
				Title:  "MP3 files",
				Layout: HBox{},
				Children: []Widget{
					ListBox{
						AssignTo:       &a.list,
						Model:          a.listModel,
						MultiSelection: true,
					},
					Composite{
						Layout: VBox{MarginsZero: true},
						Children: []Widget{
							PushButton{Text: "Add MP3s...", OnClicked: a.addFiles},
							PushButton{Text: "Add folder...", OnClicked: a.addFolder},
							PushButton{Text: "Remove selected", OnClicked: a.removeSelected},
							PushButton{Text: "Clear all", OnClicked: a.clearAll},
							VSpacer{},
						},
					},
				},
			},
			Composite{
				Layout: Grid{Columns: 3, MarginsZero: true},
				Children: []Widget{
					Label{Text: "Image:"},
					LineEdit{AssignTo: &a.imgEdit},
					PushButton{Text: "Browse...", OnClicked: a.pickImage},

					Label{Text: "Save to:"},
					LineEdit{AssignTo: &a.outEdit, ToolTipText: "Leave blank to save each MP4 next to its MP3"},
					PushButton{Text: "Browse...", OnClicked: a.pickOutput},
				},
			},
			Label{Text: "Leave \"Save to\" blank to put each MP4 next to its MP3.", TextColor: walk.RGB(110, 110, 110)},
			CheckBox{
				AssignTo: &a.fastCB,
				Text:     "Fast mode (2 fps still image, about 4x quicker, YouTube accepts it; untick for standard 30 fps)",
				Checked:  true,
			},
			ProgressBar{AssignTo: &a.progress},
			Label{AssignTo: &a.status, Text: "Add MP3 files and choose an image to begin."},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &a.cancelBtn, Text: "Cancel", Enabled: false, OnClicked: func() {
						a.cancel = true
						a.status.SetText("Cancelling after current file...")
					}},
					PushButton{AssignTo: &a.convert, Text: "Convert", OnClicked: a.start},
				},
			},
		},
	}.Create()
	if err != nil {
		walk.MsgBox(nil, appTitle, err.Error(), walk.MsgBoxIconError)
		return
	}

	if findFFmpeg() == "" {
		walk.MsgBox(a.mw, appTitle,
			"ffmpeg.exe was not found.\n\nKeep ffmpeg.exe in the same folder as this program.",
			walk.MsgBoxIconError)
	}
	a.mw.Run()
}

// ---------- pickers ----------

func (a *app) addFiles() {
	dlg := new(walk.FileDialog)
	dlg.Title = "Choose MP3 files"
	dlg.Filter = "MP3 files (*.mp3)|*.mp3|All audio (*.mp3;*.wav;*.m4a;*.flac;*.ogg)|*.mp3;*.wav;*.m4a;*.flac;*.ogg|All files (*.*)|*.*"
	if ok, _ := dlg.ShowOpenMultiple(a.mw); ok {
		a.add(dlg.FilePaths)
	}
}

func (a *app) addFolder() {
	dlg := new(walk.FileDialog)
	dlg.Title = "Choose a folder of MP3s"
	if ok, _ := dlg.ShowBrowseFolder(a.mw); ok && dlg.FilePath != "" {
		entries, _ := os.ReadDir(dlg.FilePath)
		var paths []string
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".mp3") {
				paths = append(paths, filepath.Join(dlg.FilePath, e.Name()))
			}
		}
		a.add(paths)
	}
}

func (a *app) add(paths []string) {
	for _, p := range paths {
		dup := false
		for _, f := range a.files {
			if strings.EqualFold(f, p) {
				dup = true
				break
			}
		}
		if !dup {
			a.files = append(a.files, p)
		}
	}
	a.refreshList()
}

func (a *app) refreshList() {
	a.listModel.items = a.files
	a.listModel.PublishItemsReset()
	a.status.SetText(fmt.Sprintf("%d file(s) queued.", len(a.files)))
}

func (a *app) removeSelected() {
	sel := map[int]bool{}
	for _, i := range a.list.SelectedIndexes() {
		sel[i] = true
	}
	var keep []string
	for i, f := range a.files {
		if !sel[i] {
			keep = append(keep, f)
		}
	}
	a.files = keep
	a.refreshList()
}

func (a *app) clearAll() {
	a.files = nil
	a.refreshList()
}

func (a *app) pickImage() {
	dlg := new(walk.FileDialog)
	dlg.Title = "Choose the image for the videos"
	dlg.Filter = "Images (*.jpg;*.jpeg;*.png;*.bmp;*.webp)|*.jpg;*.jpeg;*.png;*.bmp;*.webp|All files (*.*)|*.*"
	if ok, _ := dlg.ShowOpen(a.mw); ok {
		a.imgEdit.SetText(dlg.FilePath)
	}
}

func (a *app) pickOutput() {
	dlg := new(walk.FileDialog)
	dlg.Title = "Choose output folder"
	if ok, _ := dlg.ShowBrowseFolder(a.mw); ok {
		a.outEdit.SetText(dlg.FilePath)
	}
}

// ---------- conversion ----------

func (a *app) start() {
	if a.running {
		return
	}
	ff := findFFmpeg()
	if ff == "" {
		walk.MsgBox(a.mw, appTitle, "ffmpeg.exe not found. Keep it next to this program.", walk.MsgBoxIconError)
		return
	}
	if len(a.files) == 0 {
		walk.MsgBox(a.mw, appTitle, "Add at least one MP3 file.", walk.MsgBoxIconWarning)
		return
	}
	img := strings.TrimSpace(a.imgEdit.Text())
	if st, err := os.Stat(img); err != nil || st.IsDir() {
		walk.MsgBox(a.mw, appTitle, "Choose an image file.", walk.MsgBoxIconWarning)
		return
	}
	outDir := strings.TrimSpace(a.outEdit.Text())
	if outDir != "" {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			walk.MsgBox(a.mw, appTitle, "Can't create output folder:\n"+err.Error(), walk.MsgBoxIconError)
			return
		}
	}
	fast := a.fastCB.Checked()
	files := append([]string(nil), a.files...)

	a.running, a.cancel = true, false
	a.convert.SetEnabled(false)
	a.cancelBtn.SetEnabled(true)
	a.progress.SetRange(0, len(files))
	a.progress.SetValue(0)

	go func() {
		ok, failed := 0, []string{}
		for i, mp3 := range files {
			if a.cancel {
				break
			}
			name := strings.TrimSuffix(filepath.Base(mp3), filepath.Ext(mp3)) + ".mp4"
			dir := outDir
			if dir == "" {
				dir = filepath.Dir(mp3)
			}
			out := filepath.Join(dir, name)
			idx := i
			a.mw.Synchronize(func() {
				a.status.SetText(fmt.Sprintf("[%d/%d] Converting %s ...", idx+1, len(files), filepath.Base(mp3)))
			})

			cmd := exec.Command(ff, ffmpegArgs(mp3, img, out, fast)...)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			stderr, err := cmd.CombinedOutput()
			if err != nil {
				msg := strings.TrimSpace(string(stderr))
				if len(msg) > 300 {
					msg = msg[len(msg)-300:]
				}
				failed = append(failed, filepath.Base(mp3)+": "+msg)
			} else {
				ok++
			}
			done := i + 1
			a.mw.Synchronize(func() { a.progress.SetValue(done) })
		}
		a.mw.Synchronize(func() {
			a.running = false
			a.convert.SetEnabled(true)
			a.cancelBtn.SetEnabled(false)
			summary := fmt.Sprintf("Done. %d video(s) created.", ok)
			if len(failed) > 0 {
				summary += fmt.Sprintf(" %d failed.", len(failed))
				show := failed
				if len(show) > 10 {
					show = show[:10]
				}
				walk.MsgBox(a.mw, appTitle, summary+"\n\n"+strings.Join(show, "\n"), walk.MsgBoxIconWarning)
			} else {
				walk.MsgBox(a.mw, appTitle, summary, walk.MsgBoxIconInformation)
			}
			a.status.SetText(summary)
		})
	}()
}
