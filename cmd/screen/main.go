package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	defaultName := fmt.Sprintf("record_%s.mp4", time.Now().Format("2006-01-02_15-04-05"))
	output := flag.String("o", defaultName, "output filename")
	fps := flag.Int("fps", 30, "frames per second")
	display := flag.String("display", os.Getenv("DISPLAY"), "X11 display (example :0.0)")
	mic := flag.Bool("mic", false, "record microphone input")
	micDev := flag.String("mic-dev", "default", "audio source PulseAudio/PipeWire (list: pactl list short sources)")
	flag.Parse()

	var cmd *exec.Cmd
	wayland := os.Getenv("XDG_SESSION_TYPE") == "wayland"

	if wayland {
		if _, err := exec.LookPath("wf-recorder"); err != nil {
			fmt.Println("Wayland: install wf-recorder (sudo apt install wf-recorder)")
			os.Exit(1)
		}
		args := []string{"-f", *output}
		if *mic {
			if *micDev == "default" {
				args = append(args, "--audio")
			} else {
				args = append(args, "--audio="+*micDev)
			}
		}
		cmd = exec.Command("wf-recorder", args...)
	} else {
		if _, err := exec.LookPath("ffmpeg"); err != nil {
			fmt.Println("install ffmpeg: sudo apt install ffmpeg")
			os.Exit(1)
		}
		if *display == "" {
			*display = ":0.0"
		}
		args := []string{
			"-y",
			"-f", "x11grab",
			"-framerate", fmt.Sprint(*fps),
			"-i", *display,
		}
		if *mic {
			args = append(args, "-f", "pulse", "-i", *micDev, "-c:a", "aac", "-b:a", "128k")
		}
		args = append(args,
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-pix_fmt", "yuv420p",
			*output,
		)
		cmd = exec.Command("ffmpeg", args...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Println("failed to start recording:", err)
		os.Exit(1)
	}
	fmt.Printf("recording → %s\npress Ctrl+C to stop.\n", *output)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	if wayland {
		_ = cmd.Process.Signal(os.Interrupt)
	} else {
		_, _ = stdin.Write([]byte("q"))
	}
	_ = cmd.Wait()
	fmt.Println("\ndone:", *output)
}
