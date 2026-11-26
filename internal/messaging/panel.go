package messaging

import (
	"IstkharMusic/internal/cache"
	"IstkharMusic/internal/config"
	"IstkharMusic/internal/utils"
	"fmt"

	td "github.com/AshokShau/gotdbot"
)

func btn(text, data string) td.InlineKeyboardButton {
	return utils.RandBtn(text, data)
}

// ControlButtonsFor builds the premium player panel for a chat. mode is one of
// play, pause, resume, mute, unmute; an empty mode returns only a close button.
func ControlButtonsFor(chatID int64, mode string) *td.ReplyMarkupInlineKeyboard {
	closeBtn := btn("Close", "vcplay_close")
	if mode == "" {
		return &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{{closeBtn}}}
	}

	first := btn("Pause", "play_pause")
	if mode == "pause" {
		first = btn("Resume", "play_resume")
	}

	second := btn("Mute", "play_mute")
	if mode == "mute" {
		second = btn("Unmute", "play_unmute")
	}

	ap := "Autoplay: Off"
	if cache.ChatCache.GetAutoplay(chatID) {
		ap = "Autoplay: On"
	}

	return &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{first, btn("Skip", "play_skip"), btn("End", "play_stop")},
			{second, btn("Add to Playlist", "play_add_to_list")},
			{btn(ap, "play_ap_"+mode), closeBtn},
		},
	}
}

// PanelCaption renders the now playing text in the bot font.
func PanelCaption(title, url, name, duration, requester string) string {
	return fmt.Sprintf(
		"<blockquote><b>➜ %s</b></blockquote>\n<blockquote>▶️ <b>%s</b> : <a href='%s'>%s</a>\n⏱ <b>%s</b> : %s\n👤 <b>%s</b> : %s</blockquote>",
		utils.SC(title), utils.SC("Track"), url, name, utils.SC("Duration"), duration, utils.SC("Requester"), requester,
	)
}

// FinishPanel replaces the progress message with the premium photo panel. If the
// photo cannot be sent it falls back to editing the progress message.
func FinishPanel(c *td.Client, progress *td.Message, caption string, markup *td.ReplyMarkupInlineKeyboard, explicit bool) (*td.Message, error) {
	if progress == nil || progress.Id == 0 {
		return nil, nil
	}
	if Silent(progress.ChatId, explicit) {
		_ = progress.Delete(c, &td.DeleteMessagesOpts{Revoke: true})
		return nil, nil
	}

	if photo := utils.ImageFile(config.PanelImg); photo != nil {
		msg, err := c.SendPhoto(progress.ChatId, photo, &td.SendPhotoOpts{
			Caption:     caption,
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		})
		if err == nil {
			_ = progress.Delete(c, &td.DeleteMessagesOpts{Revoke: true})
			return msg, nil
		}
		c.Logger.Warn("failed to send player panel photo", "error", err)
	}

	return progress.EditText(c, caption, &td.EditMessageTextOpts{
		ReplyMarkup:           markup,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})
}

// EditAny edits a message that may either be a photo (caption) or plain text.
func EditAny(c *td.Client, chatID, messageID int64, text string, markup *td.ReplyMarkupInlineKeyboard) error {
	_, err := c.EditMessageCaption(chatID, messageID, text, &td.EditMessageCaptionOpts{ParseMode: "HTML", ReplyMarkup: markup})
	if err == nil {
		return nil
	}
	_, err = c.EditMessageText(chatID, messageID, text, &td.EditMessageTextOpts{ParseMode: "HTML", DisableWebPagePreview: true, ReplyMarkup: markup})
	return err
}
