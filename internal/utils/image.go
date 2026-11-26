package utils

import (
	"os"
	"strings"

	"github.com/AshokShau/gotdbot"
)

// ImageFile resolves an image setting (http(s) URL, local path or file id) into an
// input file. It returns nil when nothing usable is configured.
func ImageFile(src string) gotdbot.InputFile {
	src = strings.TrimSpace(src)
	if src == "" {
		return nil
	}
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return gotdbot.InputFileRemote{Id: src}
	}
	if _, err := os.Stat(src); err == nil {
		return gotdbot.InputFileLocal{Path: src}
	}
	return nil
}
