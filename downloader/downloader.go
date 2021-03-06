package downloader

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"

	"github.com/pkg/errors"
)

// Downloader handles downloading of video urls, wrapping youtube-dl
type Downloader struct {
	downloadsDir string
}

// New creates a new downloader that outputs to the given path
func New(path string) Downloader {
	dl := Downloader{downloadsDir: path}
	log.Println("creating downloader for path ", path)
	return dl
}

// Download fetches a single URL with youtube-dl and returns
// the full path to the output file. We also require that youtube-dl is
// in $PATH.
func (d Downloader) Download(url, format string) (string, error) {
	opts := defaultOptions()
	opts = append(opts, stringOption{"--output", path.Join(d.downloadsDir, "%(title)s.%(ext)s")})
	if format == "" {
		opts = append(opts, defaultFormat)
	} else {
		log.Println("Using user-specified format: ", format)
		opts = append(opts, stringOption{"--format", format})
	}

	cmd := exec.Command("youtube-dl", url)
	cmd.Args = append(cmd.Args, opts.ToCmdArgs()...)

	log.Println("Running cmd", cmd.String())

	cmd.Stderr = os.Stderr
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return "", errors.Wrap(err, "failed to create pipe")
	}

	scanner := bufio.NewScanner(cmdReader)
	pathChan := make(chan string)
	go capturingLogger(*scanner, pathChan)

	err = cmd.Start()
	if err != nil {
		return "", errors.Wrap(err, "error starting cmd")
	}

	var outFile string
	foundFile := false
	for p := range pathChan {
		foundFile = true
		outFile = p
	}

	err = cmd.Wait()
	if err != nil {
		return "", errors.Wrap(err, "error running cmd, check logs")
	}

	if !foundFile {
		return "", errors.New("unable to locate output file")
	}

	return outFile, nil
}

// pathPatterns contains patterns used to extract filenames from youtube-dl's output
var pathPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\[download\][\s](.*?)[\s]has already.+$`),
	regexp.MustCompile(`^\[ffmpeg\] Merging formats into "(.*?)"$`),
	regexp.MustCompile(`^\[download\] Destination:\W(.*?)$`),
}

// capturingLogger prints and scans for the output file. The most recent
// path found is assumed to be the final output (particularly in cases where youtube-dl
// merges video+audio files).
func capturingLogger(s bufio.Scanner, out chan<- string) {
	for s.Scan() {
		log.Println("youtube-dl ->", s.Text())
		if path, ok := matchLogPath(s.Text()); ok {
			out <- path
		}
	}

	close(out)
}

// matchLogPath looks for file paths in log lines by matching against regex patterns.
// It returns the filename and whether any match was found.
func matchLogPath(logLine string) (string, bool) {
	// log.Println("matching line", logLine)
	for _, r := range pathPatterns {
		if matches := r.FindStringSubmatch(logLine); matches != nil {
			log.Println("Matched path, ", matches[1])
			return matches[1], true
		}
	}

	return "", false
}
