package downloader

import "fmt"

// defaultQuality = "bestvideo[height<=?1080]+bestaudio/best"
// Goals: <1080p, reasonable size, avoid merging if possible
// Below works decently well, but merges a lot?
// https://www.reddit.com/r/youtubedl/comments/fe08jx/can_youtubedl_download_only_mp4_files_at_1080_or/
const defaultQuality = "bestvideo[ext=mp4][height<=?1080]+bestaudio[ext=m4a]/best"

type ytdlopts []option

type option interface {
	toArg() string
}

type stringOption struct {
	Option string
	Value  string
}
type boolOption struct {
	Option string
}

func (o boolOption) toArg() string {
	return o.Option
}
func (o stringOption) toArg() string {
	return fmt.Sprintf("%s=%s", o.Option, o.Value)
}

func (o ytdlopts) ToCmdArgs() []string {
	var args []string
	for _, opt := range o {
		args = append(args, opt.toArg())
	}
	return args
}

func defaultOptions() ytdlopts {
	return ytdlopts{
		stringOption{"--retries", "3"},
		stringOption{"--format", defaultQuality},
		boolOption{"--no-progress"},
	}
}