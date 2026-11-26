/*
 * IstkharMusic - Telegram Music Bot
 *  https://github.com/TEAM-ISTKHAR/IstkharMusic
 *
 *  Based on TgMusicBot, Copyright (c) 2025-2026 Ashok Shau
 *  Licensed under GNU GPL v3
 */

package bot

import (
	"IstkharMusic/internal/cache"
	"IstkharMusic/internal/calls"
	"IstkharMusic/internal/db"
	"IstkharMusic/internal/messaging"
	"IstkharMusic/internal/utils"
	"fmt"
	"html"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func playCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()
	if !adminModeCB(c, cb) {
		return td.EndGroups
	}

	chatID := cb.ChatId
	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "Unknown", Id: cb.SenderUserId}
	}

	if !cache.ChatCache.IsActive(chatID) {
		text := "There is no active playback."
		_ = cb.Answer(c, text, nil)
		_ = editCB(c, cb, text, messaging.ControlButtonsFor(chatID, ""))
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatID)
	if currentTrack == nil {
		_ = cb.Answer(c, "There is no active playback.", nil)
		_ = editCB(c, cb, "There is no active playback.", messaging.ControlButtonsFor(chatID, ""))
		return nil
	}

	buildTrackMessage := func(status, emoji string) string {
		return fmt.Sprintf("<blockquote><b>%s %s</b></blockquote>\n<blockquote>🎵 <b>%s</b> : <a href='%s'>%s</a>\n⏱ <b>%s</b> : %s\n👤 <b>%s</b> : %s</blockquote>",
			emoji, utils.SC(status), utils.SC("Track"),
			html.EscapeString(currentTrack.URL), html.EscapeString(currentTrack.Name),
			utils.SC("Duration"), utils.SecToMin(currentTrack.Duration),
			utils.SC("Requester"), html.EscapeString(currentTrack.User),
		)
	}

	if strings.HasPrefix(data, "play_ap_") {
		mode := strings.TrimPrefix(data, "play_ap_")
		state := !cache.ChatCache.GetAutoplay(chatID)
		cache.ChatCache.SetAutoplay(chatID, state)
		if state {
			_ = cb.Answer(c, "Autoplay enabled.", nil)
		} else {
			_ = cb.Answer(c, "Autoplay disabled.", nil)
		}
		_, _ = c.EditMessageReplyMarkup(chatID, cb.MessageId, &td.EditMessageReplyMarkupOpts{ReplyMarkup: messaging.ControlButtonsFor(chatID, mode)})
		return nil
	}

	switch {
	case strings.Contains(data, "play_skip"):
		if err := calls.Calls.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, "Unable to skip the current track.", nil)
			_ = editCB(c, cb, "Unable to skip the current track.", messaging.ControlButtonsFor(chatID, ""))
			return nil
		}
		_ = cb.Answer(c, "Track skipped.", nil)
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil

	case strings.Contains(data, "play_stop"):
		if err := calls.Calls.Stop(chatID, false); err != nil {
			_ = cb.Answer(c, "Unable to stop playback.", nil)
			_ = editCB(c, cb, "Unable to stop playback.", messaging.ControlButtonsFor(chatID, ""))
			return nil
		}

		msg := fmt.Sprintf("<b>Playback stopped.</b>\nRequested by: %s", html.EscapeString(user.FirstName))
		_ = cb.Answer(c, "Playback stopped.", nil)
		err := editCB(c, cb, msg, messaging.ControlButtonsFor(chatID, ""))
		return err

	case strings.Contains(data, "play_pause"):
		if _, err = calls.Calls.Pause(chatID); err != nil {
			_ = cb.Answer(c, "Unable to pause playback.", nil)
			_ = editCB(c, cb, "Unable to pause playback.", messaging.ControlButtonsFor(chatID, ""))
			return nil
		}
		_ = cb.Answer(c, "Playback paused.", nil)
		text := buildTrackMessage("Paused", "⏸") + fmt.Sprintf("\n\nPaused by %s", html.EscapeString(user.FirstName))
		_ = editCB(c, cb, text, messaging.ControlButtonsFor(chatID, "pause"))
		return nil

	case strings.Contains(data, "play_resume"):
		if _, err := calls.Calls.Resume(chatID); err != nil {
			_ = cb.Answer(c, "Unable to resume playback.", nil)
			_ = editCB(c, cb, "Unable to resume playback.", messaging.ControlButtonsFor(chatID, "pause"))
			return nil
		}
		_ = cb.Answer(c, "Playback resumed.", nil)
		text := buildTrackMessage("Now Playing", "▶") + fmt.Sprintf("\n\nResumed by %s", html.EscapeString(user.FirstName))
		_ = editCB(c, cb, text, messaging.ControlButtonsFor(chatID, "resume"))
		return nil

	case strings.Contains(data, "play_mute"):
		if _, err := calls.Calls.Mute(chatID); err != nil {
			_ = cb.Answer(c, "Unable to mute playback.", nil)
			_ = editCB(c, cb, "Unable to mute playback.", messaging.ControlButtonsFor(chatID, "mute"))
			return nil
		}
		_ = cb.Answer(c, "Playback muted.", nil)
		text := buildTrackMessage("Muted", "🔇") + fmt.Sprintf("\n\nMuted by %s", html.EscapeString(user.FirstName))
		_ = editCB(c, cb, text, messaging.ControlButtonsFor(chatID, "mute"))
		return nil

	case strings.Contains(data, "play_unmute"):
		if _, err := calls.Calls.Unmute(chatID); err != nil {
			_ = cb.Answer(c, "Unable to unmute playback.", nil)
			_ = editCB(c, cb, "Unable to unmute playback.", messaging.ControlButtonsFor(chatID, "unmute"))
			return nil
		}
		_ = cb.Answer(c, "Playback unmuted.", nil)
		text := buildTrackMessage("Now Playing", "▶") + fmt.Sprintf("\n\nUnmuted by %s", html.EscapeString(user.FirstName))
		_ = editCB(c, cb, text, messaging.ControlButtonsFor(chatID, "unmute"))
		return nil

	case strings.Contains(data, "play_add_to_list"):
		playlists, err := db.Instance.GetUserPlaylists(cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, "Unable to fetch playlists.", nil)
			return nil
		}

		var playlistID string
		if len(playlists) == 0 {
			playlistID, err = db.Instance.CreatePlaylist("My Playlist (IstkharMusic)", cb.SenderUserId)
			if err != nil {
				_ = cb.Answer(c, "Unable to create playlist.", nil)
				return nil
			}
		} else {
			playlistID = playlists[0].ID
		}

		song := db.Song{
			URL:      currentTrack.URL,
			Name:     currentTrack.Name,
			TrackID:  currentTrack.TrackID,
			Duration: currentTrack.Duration,
			Platform: currentTrack.Platform,
		}

		err = db.Instance.AddSongToPlaylist(playlistID, song)
		if err != nil {
			_ = cb.Answer(c, "Unable to add track to playlist.", nil)
			return nil
		}

		playlist, err := db.Instance.GetPlaylist(playlistID)
		if err != nil {
			_ = cb.Answer(c, "Playlist not found.", nil)
			return nil
		}

		_ = cb.Answer(c, fmt.Sprintf("Track \"%s\" added to playlist \"%s\".", song.Name, playlist.Name), nil)
		return nil

	case strings.HasPrefix(data, "play_now_"):
		trackID := strings.TrimPrefix(data, "play_now_")
		if strings.HasPrefix(data, "play_now_hash_") {
			trackID = ""
			for _, track := range cache.ChatCache.GetQueue(chatID) {
				if track != nil && utils.PlayNowCallbackData(track.TrackID) == data {
					trackID = track.TrackID
					break
				}
			}
		}
		if ok := cache.ChatCache.MoveTrackToFront(chatID, trackID); !ok {
			_ = cb.Answer(c, "Track not found in queue.", nil)
			return nil
		}

		_ = cb.Answer(c, "Playing now.", nil)
		if err = calls.Calls.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, "Unable to play the track.", nil)
			return nil
		}

		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	text := buildTrackMessage("Now Playing", "▶")
	_ = editCB(c, cb, text, messaging.ControlButtonsFor(chatID, "resume"))
	return nil
}

func vcPlayHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()

	if strings.Contains(data, "vcplay_close") {
		_ = cb.Answer(c, "Closing panel.", nil)
		_ = c.DeleteMessages(cb.ChatId, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	c.Logger.Info("Received vcplay callback", "data", data)
	return nil
}

// editCB edits the callback message whether it is a photo panel or a text message.
func editCB(c *td.Client, cb *td.UpdateNewCallbackQuery, text string, markup *td.ReplyMarkupInlineKeyboard) error {
	return messaging.EditAny(c, cb.ChatId, cb.MessageId, text, markup)
}
