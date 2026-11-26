/*
 * IstkharMusic - Telegram Music Bot
 *  https://github.com/TEAM-ISTKHAR/IstkharMusic
 *
 *  Based on TgMusicBot, Copyright (c) 2025-2026 Ashok Shau
 *  Licensed under GNU GPL v3
 */

package bot

import (
	"IstkharMusic/internal/config"
	"IstkharMusic/internal/utils"
	"fmt"
	"html"
	"runtime"
	"time"

	td "github.com/AshokShau/gotdbot"
)

func pingHandler(c *td.Client, m *td.Message) error {
	deleteCmd(c, m)

	start := time.Now()
	text := func(latency int64) string {
		return fmt.Sprintf(
			"<blockquote><b>📊 %s</b></blockquote>\n<blockquote>⚡ <b>%s</b> : <code>%d ms</code>\n⏱ <b>%s</b> : <code>%s</code>\n🧵 <b>%s</b> : <code>%d</code></blockquote>",
			utils.SC("System performance"), utils.SC("Latency"), latency, utils.SC("Uptime"), getFormattedDuration(time.Since(startTime)),
			utils.SC("Go routines"), runtime.NumGoroutine(),
		)
	}

	if photo := utils.ImageFile(config.PingImg); photo != nil {
		msg, err := m.ReplyPhoto(c, photo, &td.SendPhotoOpts{Caption: utils.SC("Pinging… please wait…"), ParseMode: "HTML", ReplyMarkup: utils.SupportBtn()})
		if err == nil && msg != nil {
			_, err = c.EditMessageCaption(msg.ChatId, msg.Id, text(time.Since(start).Milliseconds()), &td.EditMessageCaptionOpts{ParseMode: "HTML", ReplyMarkup: utils.SupportBtn()})
			return err
		}
	}

	msg, err := m.ReplyText(c, utils.SC("Pinging… please wait…"), nil)
	if err != nil {
		return err
	}
	_, err = msg.EditText(c, text(time.Since(start).Milliseconds()), &td.EditMessageTextOpts{ParseMode: "HTML", ReplyMarkup: utils.SupportBtn()})
	return err
}

func supportHandler(c *td.Client, m *td.Message) error {
	deleteCmd(c, m)
	caption := "<blockquote><b>💬 " + utils.SC("Need help or want updates?") + "</b></blockquote>\n<blockquote>" + utils.SC("Join our support group and updates channel.") + "</blockquote>"
	if photo := utils.ImageFile(config.SupportImg); photo != nil {
		if _, err := m.ReplyPhoto(c, photo, &td.SendPhotoOpts{Caption: caption, ParseMode: "HTML", ReplyMarkup: utils.SupportBtn()}); err == nil {
			return nil
		}
	}
	_, err := m.ReplyText(c, caption, &td.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: utils.SupportBtn()})
	return err
}

func startHandler(c *td.Client, m *td.Message) error {
	chatID := m.ChatId
	go storeChatToDB(chatID)

	deleteCmd(c, m)

	caption := startCaption(c, firstName(c, m))
	markup := utils.AddMeMarkup(c.Me().Usernames.EditableUsername)

	if !m.IsPrivate() {
		caption = fmt.Sprintf("<blockquote><b>%s %s</b></blockquote>\n<blockquote>⏱ <b>%s</b> : <code>%s</code>\n🎵 %s /play</blockquote>",
			html.EscapeString(c.Me().FirstName), utils.SC("is ready!"), utils.SC("Uptime"), getFormattedDuration(time.Since(startTime)), utils.SC("Start music with"))
	}

	if photo := utils.ImageFile(config.StartImg); photo != nil {
		if _, err := m.ReplyPhoto(c, photo, &td.SendPhotoOpts{Caption: caption, ParseMode: "HTML", ReplyMarkup: markup}); err == nil {
			return nil
		} else {
			c.Logger.Warn("start photo failed", "error", err)
		}
	}

	_, err := m.ReplyText(c, caption, &td.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: markup})
	return err
}

// sendThanksForAdding greets a group when the bot is added to it.
func sendThanksForAdding(c *td.Client, chatID, actorID int64) {
	defer func() { _ = recover() }()

	adder := "Unknown"
	if actorID > 0 {
		if u, err := c.GetUser(actorID); err == nil && u != nil {
			adder = u.FirstName
		}
	}
	title := ""
	if chat, err := c.GetChat(chatID); err == nil && chat != nil {
		title = chat.Title
	}

	caption := fmt.Sprintf(
		"<blockquote><b>🎵 %s @%s</b></blockquote>\n"+
			"<blockquote>👤 <b>%s</b> : %s\n📌 <b>%s</b> : %s\n🆔 <b>%s</b> : <code>%d</code></blockquote>\n"+
			"<b>%s</b>\n<b>%s</b> /settings.\n\n/play <code>%s</code> — %s",
		utils.SC("Thanks for adding"), c.Me().Usernames.EditableUsername,
		utils.SC("Added by"), html.EscapeString(adder),
		utils.SC("Group"), html.EscapeString(title),
		utils.SC("Chat id"), chatID,
		utils.SC("I can play music in this group voice chat."),
		utils.SC("To change language, send"),
		utils.SC("song name"), utils.SC("play a song"),
	)
	markup := &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{
		{utils.RandBtn("🌐 Language", "settings_lang")},
		{utils.RandURL("📢 Updates", config.SupportChannel), utils.RandURL("💬 Support", config.SupportGroup)},
	}}

	if photo := utils.ImageFile(config.StartImg); photo != nil {
		if _, err := c.SendPhoto(chatID, photo, &td.SendPhotoOpts{Caption: caption, ParseMode: "HTML", ReplyMarkup: markup}); err == nil {
			return
		}
	}
	_, _ = c.SendMessage(chatID, caption, &td.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: markup})
}
