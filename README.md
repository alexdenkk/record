# record
Some tools to record screen and web-camera

## requirements

- go 1.26.1 or later
- ffmpeg (with `ffplay`, it is used for the camera preview)
- [wf-recorder](https://github.com/ammen99/wf-recorder) (screen recording on Wayland only)
- PulseAudio or PipeWire (only for microphone recording on Linux)

## clone
```sh
git clone https://github.com/alexdenkk/record.git
cd record
```

## camera

Shows a live camera preview window.
Recording can be started and stopped while the preview is running.

```sh
go build -o cam ./cmd/cam/main.go
./cam
```

flags
```sh
-d /dev/video0   # camera device (Linux: /dev/video0, macOS: index like 0, Windows: name from dshow)
-s 1920x1080     # resolution (default 1920x1080)
-fps 30          # frames per second (default 30)
-noborder        # borderless preview window
```

controls

Type a command in the terminal and press Enter:
```sh
r  # start/stop recording to <date_time>.mp4 in the current directory
q  # quit
```

Recording is saved without sound. Starting and stopping a recording briefly restarts the preview.

On Windows the camera name is required, list the available cameras with:
```sh
ffmpeg -list_devices true -f dshow -i dummy
./cam -d "Integrated Camera"
```

## screen

Records the screen to an mp4 file. Uses `ffmpeg` (x11grab) on X11 and `wf-recorder` on Wayland. Linux only.

```sh
go build -o screen ./cmd/screen/main.go
./screen
```

flags
```sh
-o record.mp4    # output filename (default record_<date_time>.mp4)
-fps 30          # frames per second (default 30, X11 only)
-display :0.0    # X11 display (default $DISPLAY, X11 only)
-mic             # record microphone input
-mic-dev default # audio source PulseAudio/PipeWire
```

List available audio sources:
```sh
pactl list short sources
```

Press `Ctrl+C` to stop the recording.

example
```sh
./screen -o demo.mp4 -fps 60 -mic
```
