/*
 * IstkharMusic - Telegram Music Bot
 *  https://github.com/TEAM-ISTKHAR/IstkharMusic
 *
 *  Based on TgMusicBot, Copyright (c) 2025-2026 Ashok Shau
 *  Licensed under GNU GPL v3
 */

package utils

import (
	"IstkharMusic/internal/config"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/AshokShau/gotdbot"
)

func cb(text, data string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeCallback{
			Data: []byte(data),
		},
		Style: style,
	}
}

func url(text, link string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeUrl{
			Url: link,
		},
		Style: style,
	}
}

var CloseBtn = cb(SC("Close"), "vcplay_close", gotdbot.ButtonStyleDanger{})
var HomeBtn = cb(SC("Home"), "help_back", gotdbot.ButtonStyleSuccess{})
var HelpBtn = cb(SC("Help"), "help_all", gotdbot.ButtonStylePrimary{})

var channelBtn = url(SC("Updates"), config.SupportChannel, gotdbot.ButtonStylePrimary{})
var groupBtn = url(SC("Support"), config.SupportGroup, gotdbot.ButtonStyleSuccess{})

func SupportKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
			{CloseBtn},
		},
	}
}

func SupportBtn() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
		},
	}
}

func SettingsKeyboard(playMode, adminMode string, cmdDelete bool, language string, autoplay bool) *gotdbot.ReplyMarkupInlineKeyboard {
	playText := "Everyone"
	if playMode == Admins {
		playText = "Admins"
	}

	deleteText := "False"
	if cmdDelete {
		deleteText = "True"
	}

	adminText := "Everyone"
	if adminMode == Admins {
		adminText = "Admins"
	}

	langText := "English"
	if language != "en" && language != "" {
		langText = language
	}

	autoplayText := "Disabled"
	if autoplay {
		autoplayText = "Enabled"
	}

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{
				cb(SC("Play Mode")+" ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(playText, "settings_play", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb(SC("Command Delete")+" ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(deleteText, "settings_delete", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb(SC("Admin Mode")+" ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(adminText, "settings_admin", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb(SC("Autoplay")+" ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(autoplayText, "settings_autoplay", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb(SC("Language")+" ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(langText, "settings_lang", gotdbot.ButtonStyleDefault{}),
			},
			{CloseBtn},
		},
	}
}

func HelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	h := func(label, key string) gotdbot.InlineKeyboardButton { return RandBtn(label, "help_"+key) }
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{h("Admins", "admins"), h("Auth", "auth"), h("Approve", "approve")},
			{h("Language", "language"), h("Ping", "ping"), h("Play", "play")},
			{h("Restrictor", "restrictor"), h("Stats", "stats"), h("Sudoers", "sudoers")},
			{h("Autoplay", "autoplay"), h("Vc Logger", "vclogger"), h("Loop", "loop")},
			{h("Playlist", "playlist"), h("Data", "data")},
			{HomeBtn, CloseBtn},
		},
	}
}

func BackHelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{HelpBtn, HomeBtn},
			{CloseBtn},
		},
	}
}

// ControlButtons is the plain fallback keyboard; the player panel lives in
// messaging.ControlButtonsFor.
func ControlButtons(_ string) *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{{CloseBtn}},
	}
}

func AddMeMarkup(username string) *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{RandURL("➕ Add me to your group", fmt.Sprintf("https://t.me/%s?startgroup=true", username))},
			{HelpBtn},
			{channelBtn, groupBtn},
		},
	}
}

func PlayNowButton(trackID string) gotdbot.InlineKeyboardButton {
	return cb(SC("Play Now"), PlayNowCallbackData(trackID), gotdbot.ButtonStyleDanger{})
}

func QueueMarkup(trackID string) *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{PlayNowButton(trackID), CloseBtn},
		},
	}
}

// PlayNowCallbackData keeps Telegram remote file IDs within the 64-byte callback
// limit without changing the original IDs needed for downloading queued files.
func PlayNowCallbackData(trackID string) string {
	data := "play_now_" + trackID
	if len(data) <= 64 {
		return data
	}
	hash := sha256.Sum256([]byte(trackID))
	return "play_now_hash_" + base64.RawURLEncoding.EncodeToString(hash[:])
}
