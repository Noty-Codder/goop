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
	"IstkharMusic/internal/messaging"
	"fmt"

	td "github.com/AshokShau/gotdbot"
)

func stopHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId
	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "The bot isn't streaming in the video chat.", nil)
		return nil
	}

	_ = calls.Calls.Stop(chatID, false)
	_, _ = messaging.Reply(c, m, fmt.Sprintf("<b>Stream ended by</b> %s", firstName(c, m)), replyOpts)
	return nil
}
