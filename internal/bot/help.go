package bot

import (
	"IstkharMusic/internal/messaging"
	"IstkharMusic/internal/utils"
	"fmt"
	"html"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

type helpPage struct {
	Title string
	Body  string
}

func cmdLine(cmd, desc string) string {
	return fmt.Sprintf("• <code>%s</code> — %s\n", cmd, utils.SC(desc))
}

func helpPages() map[string]helpPage {
	return map[string]helpPage{
		"help_play": {"Play", cmdLine("/play [song]", "play from youtube, spotify and more") +
			cmdLine("/vplay [song]", "play a video") +
			cmdLine("/fplay /fvplay", "play right now, skipping queue") +
			cmdLine("/mix [song]", "mix playlist of related songs") +
			cmdLine("/queue", "show the queue") +
			cmdLine("/skip /pause /resume /end", "playback controls") +
			cmdLine("/seek [sec]", "jump in the track") +
			cmdLine("/mute /unmute", "mute the stream") +
			cmdLine("/remove [n]", "remove a queued track") +
			cmdLine("/splay /sskip …", "silent mode, add s before a command")},
		"help_admins": {"Admins", cmdLine("/skip /pause /resume /end", "control playback") +
			cmdLine("/seek /mute /unmute", "extra controls") +
			cmdLine("/remove /loop", "queue tools") +
			cmdLine("/join [link]", "bring the assistant to the group") +
			cmdLine("/reload", "refresh admin cache") +
			cmdLine("/settings", "chat settings, play mode, admin mode, language")},
		"help_auth": {"Auth", cmdLine("/auth", "authorize a user for admin commands") +
			cmdLine("/unauth", "remove authorization") +
			cmdLine("/authlist", "list authorized users")},
		"help_approve": {"Approve", cmdLine("/approve", "reply or id, exempt a user from link and bio remover") +
			cmdLine("/unapprove", "remove the exemption") +
			cmdLine("/approved", "list approved users")},
		"help_restrictor": {"Restrictor", cmdLine("/linkremove on|off", "delete messages that contain links") +
			cmdLine("/bioremove on|off", "delete messages from users who have links in their bio") +
			cmdLine("/approve /unapprove /approved", "manage exempt users") +
			"\n" + utils.SC("Admins, authorized and approved users are never touched. The bot needs delete messages right.")},
		"help_language": {"Language", cmdLine("/settings", "open settings and tap Language")},
		"help_ping":     {"Ping", cmdLine("/ping", "latency, uptime and runtime info")},
		"help_stats":    {"Stats", cmdLine("/stats", "system, bot and database statistics (sudo)")},
		"help_data": {"Data", cmdLine("/data", "today's report: groups added and removed, songs, users (sudo)") +
			cmdLine("/bdata", "last 24 hours report (sudo)") +
			cmdLine("/selfpromo run", "run the promo broadcast now (sudo)") +
			"\n" + utils.SC("A daily report is also sent automatically every morning at 7 am IST.")},
		"help_sudoers": {"Sudoers", cmdLine("/stats /data /bdata", "statistics") +
			cmdLine("/av", "active voice chats") +
			cmdLine("/broadcast", "broadcast a replied message") +
			cmdLine("/selfpromo run", "promo broadcast") +
			cmdLine("/logger on|off", "song play logger") +
			cmdLine("/leaveall /clearass", "assistant maintenance")},
		"help_autoplay": {"Autoplay", cmdLine("/autoplay", "toggle autoplay for the running chat") +
			utils.SC("Use the Autoplay button on the player panel as well. When the queue ends the bot leaves the voice chat.")},
		"help_vclogger": {"Vc Logger", cmdLine("/vclogger on|off", "show join and leave notices of the voice chat") +
			utils.SC("Notices disappear automatically after a few seconds and include a Join Live Vc button.")},
		"help_loop": {"Loop", cmdLine("/loop [0-10]", "repeat the current track")},
		"help_playlist": {"Playlist", cmdLine("/createplaylist", "create a playlist") +
			cmdLine("/deleteplaylist", "delete a playlist") +
			cmdLine("/addtoplaylist", "add a track") +
			cmdLine("/removefromplaylist", "remove a track") +
			cmdLine("/playlistinfo /myplaylists", "view playlists")},
	}
}

func helpMenuText() string {
	return "<blockquote><b>" + utils.SC("Click the buttons below to get information about my commands.") + "</b></blockquote>\n" +
		"<blockquote><b>" + utils.SC("Note") + ":</b> <i>" + utils.SC("All commands can be used with") + " /</i></blockquote>"
}

func helpHandler(c *td.Client, m *td.Message) error {
	deleteCmd(c, m)
	_, err := m.ReplyText(c, helpMenuText(), &td.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: utils.HelpMenuKeyboard()})
	return err
}

func helpCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()

	user, err := c.GetUser(cb.SenderUserId)
	if err != nil || user == nil {
		user = &td.User{FirstName: "User", Id: cb.SenderUserId}
	}

	switch {
	case strings.Contains(data, "help_all"):
		_ = cb.Answer(c, "Opening help menu...", nil)
		_ = messaging.EditAny(c, cb.ChatId, cb.MessageId, helpMenuText(), utils.HelpMenuKeyboard())
		return nil

	case strings.Contains(data, "help_back"):
		_ = cb.Answer(c, "Returning to main menu...", nil)
		_ = messaging.EditAny(c, cb.ChatId, cb.MessageId, startCaption(c, user.FirstName), utils.AddMeMarkup(c.Me().Usernames.EditableUsername))
		return nil
	}

	if page, ok := helpPages()[data]; ok {
		_ = cb.Answer(c, page.Title, nil)
		text := fmt.Sprintf("<blockquote><b>%s</b></blockquote>\n<blockquote>%s</blockquote>", utils.SC(page.Title), page.Body)
		_ = messaging.EditAny(c, cb.ChatId, cb.MessageId, text, utils.BackHelpMenuKeyboard())
		return nil
	}

	_ = cb.Answer(c, "Unknown help category.", &td.CallbackAnswerOpts{ShowAlert: true})
	return nil
}

func startCaption(c *td.Client, name string) string {
	return fmt.Sprintf(
		"<blockquote><b>✨ %s %s</b></blockquote>\n"+
			"<blockquote><b>%s</b> %s\n\n"+
			"🎧 %s\n⚡ %s\n🛡 %s\n🔗 %s</blockquote>\n"+
			"<b>%s</b>",
		utils.SC("Hey"), html.EscapeString(name),
		html.EscapeString(c.Me().FirstName), utils.SC("is your premium music companion for telegram voice chats"),
		utils.SC("Crystal clear hi-res streaming"),
		utils.SC("Fast search with zero delay"),
		utils.SC("Link and bio remover included"),
		utils.SC("Autoplay, playlists and vc logger"),
		utils.SC("Add me to your group and send /play song name"),
	)
}
