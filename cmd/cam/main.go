package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type session struct {
	ff, play *exec.Cmd
	ffIn     io.WriteCloser
	done     chan struct{}
}

func inputArgs(device, size string, fps int) ([]string, error) {
	f := fmt.Sprint(fps)
	switch runtime.GOOS {
	case "linux":
		if device == "" {
			device = "/dev/video0"
		}
		return []string{"-f", "v4l2", "-framerate", f, "-video_size", size, "-i", device}, nil
	case "darwin":
		if device == "" {
			device = "0"
		}
		return []string{"-f", "avfoundation", "-framerate", f, "-video_size", size, "-i", device}, nil
	case "windows":
		if device == "" {
			return nil, fmt.Errorf("type camera name: -d \"Integrated Camera\"\n" +
				"camera list: ffmpeg -list_devices true -f dshow -i dummy")
		}
		return []string{"-f", "dshow", "-framerate", f, "-video_size", size, "-i", "video=" + device}, nil
	}
	return nil, fmt.Errorf("unsupported ОС: %s", runtime.GOOS)
}

func start(in []string, recFile string, noBorder bool) (*session, error) {
	args := []string{"-hide_banner", "-loglevel", "error"}
	args = append(args, in...)

	if recFile != "" {
		args = append(args, "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p", recFile)
	}

	args = append(args, "-c:v", "mjpeg", "-q:v", "4", "-f", "nut", "pipe:1")

	pargs := []string{"-hide_banner", "-loglevel", "error", "-window_title", "Camera",
		"-fflags", "nobuffer", "-flags", "low_delay", "-framedrop"}

	if noBorder {
		pargs = append(pargs, "-noborder")
	}
	pargs = append(pargs, "-i", "pipe:0")

	ff := exec.Command("ffmpeg", args...)
	ffIn, err := ff.StdinPipe()

	if err != nil {
		return nil, err
	}

	out, err := ff.StdoutPipe()

	if err != nil {
		return nil, err
	}

	ff.Stderr = os.Stderr

	play := exec.Command("ffplay", pargs...)
	play.Stdin = out
	play.Stderr = os.Stderr

	if err := ff.Start(); err != nil {
		return nil, err
	}

	if err := play.Start(); err != nil {
		ff.Process.Kill()
		return nil, err
	}

	s := &session{ff: ff, play: play, ffIn: ffIn, done: make(chan struct{})}

	go func() {
		play.Wait()
		close(s.done)
	}()

	return s, nil
}

func (s *session) stop() {
	io.WriteString(s.ffIn, "q")
	s.ffIn.Close()

	finished := make(chan struct{})

	go func() {
		s.ff.Wait()
		close(finished)
	}()

	select {
	case <-finished:

	case <-time.After(5 * time.Second):
		s.ff.Process.Kill()
		<-finished
	}

	s.play.Process.Kill()
	<-s.done
}

func main() {
	device := flag.String("d", "", "camera (Linux: /dev/video0, macOS: index 0, Windows: name from dshow)")
	size := flag.String("s", "1920x1080", "resolution")
	fps := flag.Int("fps", 30, "frames per second")
	noBorder := flag.Bool("noborder", false, "borderless window")
	flag.Parse()

	for _, bin := range []string{"ffmpeg", "ffplay"} {
		if _, err := exec.LookPath(bin); err != nil {
			fmt.Println("not found", bin, "— install ffmpeg and add it to PATH")
			return
		}
	}

	in, err := inputArgs(*device, *size, *fps)

	if err != nil {
		fmt.Println(err)
		return
	}

	sess, err := start(in, "", *noBorder)

	if err != nil {
		fmt.Println("fail to start:", err)
		return
	}

	lines := make(chan string)

	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			lines <- strings.ToLower(strings.TrimSpace(sc.Text()))
		}
		close(lines)
	}()

	fmt.Println("r + Enter — start/stop recording | q + Enter — quit")
	recording := false

	for {
		select {
		case <-sess.done:
			sess.stop()
			return

		case cmd, ok := <-lines:
			if !ok {
				lines = nil
				continue
			}
			switch cmd {
			case "q":
				sess.stop()
				return
			case "r":
				sess.stop()
				name := ""
				if !recording {
					name = time.Now().Format("2006-01-02_15-04-05") + ".mp4"
					fmt.Println("write to", name)
				} else {
					fmt.Println("record stopped")
				}
				recording = !recording
				sess, err = start(in, name, *noBorder)
				if err != nil {
					fmt.Println("failed to restart:", err)
					return
				}
			}
		}
	}
}
