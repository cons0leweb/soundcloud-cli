package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cons0leweb/soundcloud-cli/internal/player"
	"github.com/cons0leweb/soundcloud-cli/internal/soundcloud"
	"github.com/cons0leweb/soundcloud-cli/internal/tui"
)

var version = "dev"

func main() {
	var (
		limit        = flag.Int("limit", 20, "maximum number of search results")
		ytdlpBinary  = flag.String("yt-dlp", "yt-dlp", "path to yt-dlp")
		ffplayBinary = flag.String("ffplay", "ffplay", "path to ffplay")
		cookies      = flag.String("cookies", "auto", "path to a Netscape-format cookie file, auto, or empty for public mode")
		harFile      = flag.String("har", "", "path to a SoundCloud HAR file for personal mixes, likes, and history")
		showVersion  = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println("soundcloud", version)
		return
	}
	if *limit < 1 || *limit > 100 {
		fatal("--limit must be between 1 and 100")
	}
	if *cookies == "auto" {
		if info, err := os.Stat("netscape.txt"); err == nil && !info.IsDir() {
			*cookies = "netscape.txt"
		} else {
			*cookies = ""
		}
	}
	if _, err := exec.LookPath(*ytdlpBinary); err != nil {
		fatal("yt-dlp was not found; install it or pass --yt-dlp /path/to/yt-dlp")
	}
	if _, err := exec.LookPath(*ffplayBinary); err != nil {
		fatal("ffplay was not found; install FFmpeg or pass --ffplay /path/to/ffplay")
	}
	if *cookies != "" {
		if info, err := os.Stat(*cookies); err != nil || info.IsDir() {
			fatal("cookie file was not found; pass --cookies=/path/to/file or --cookies= for public mode")
		}
	}

	client := soundcloud.New(*ytdlpBinary, *cookies, *harFile, *limit)
	audio := player.New(*ffplayBinary)
	defer audio.Close()

	program := tea.NewProgram(tui.New(client, audio), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fatal(err.Error())
	}
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "soundcloud:", message)
	os.Exit(1)
}
