/*
 * IstkharMusic - Telegram Music Bot
 *  https://github.com/TEAM-ISTKHAR/IstkharMusic
 *
 *  Based on TgMusicBot, Copyright (c) 2025-2026 Ashok Shau
 *  Licensed under GNU GPL v3
 */

package downloader

import (
	"IstkharMusic/internal/utils"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	downloadTimeout        = 40 * time.Second
	defaultDownloadDirPerm = 0755
)

var (
	errMissingCDNURL = errors.New("missing cdn url")
)

func processDownload(track *utils.TrackInfo) (string, error) {
	if track == nil {
		return "", errors.New("track info is nil")
	}
	if track.CdnURL == "" {
		return "", errMissingCDNURL
	}

	if track.Key != "" && strings.EqualFold(track.Platform, "spotify") {
		return processSpotify(track)
	}

	return track.CdnURL, nil
}

var (
	sanitizeRegex = regexp.MustCompile(`[<>:"/\\|?*]`)
	filenameRegex = regexp.MustCompile(`filename\*?=(?:UTF-8'')?([^;]+)`)
)

func sanitizeFilename(fileName string) string {
	fileName = strings.ReplaceAll(fileName, "/", "")
	fileName = strings.ReplaceAll(fileName, "\\", "")
	fileName = sanitizeRegex.ReplaceAllString(fileName, "")
	fileName = strings.TrimSpace(fileName)
	return fileName
}

func extractFilename(contentDisp string) string {
	if contentDisp == "" {
		return ""
	}
	matches := filenameRegex.FindStringSubmatch(contentDisp)
	if len(matches) > 1 {
		decoded, err := url.QueryUnescape(matches[1])
		if err == nil {
			return decoded
		}
	}
	return ""
}
