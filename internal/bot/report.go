package bot

import (
	"IstkharMusic/internal/cache"
	"IstkharMusic/internal/config"
	"IstkharMusic/internal/db"
	"IstkharMusic/internal/utils"
	"fmt"
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
)

var istZone = time.FixedZone("IST", 5*3600+1800)

// buildReport renders the activity report since the given moment.
func buildReport(title string, since time.Time) string {
	added := db.Instance.CountEvents(db.EventChatAdd, since)
	removed := db.Instance.CountEvents(db.EventChatRemove, since)
	plays := db.Instance.CountEvents(db.EventPlay, since)
	newUsers := db.Instance.CountEvents(db.EventNewUser, since)
	links := db.Instance.CountEvents(db.EventLinkRemoved, since)
	bios := db.Instance.CountEvents(db.EventBioRemoved, since)
	vcJoin := db.Instance.CountEvents(db.EventVCJoin, since)
	vcLeave := db.Instance.CountEvents(db.EventVCLeave, since)

	chats, _ := db.Instance.GetAllChats()
	users, _ := db.Instance.GetAllUsers()
	active := len(cache.ChatCache.GetActiveChats())

	line := func(emoji, label string, v any) string {
		return fmt.Sprintf("%s <b>%s</b> ➜ <code>%v</code>\n", emoji, utils.SC(label), v)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<blockquote><b>📊 %s</b></blockquote>\n", utils.SC(title)))
	sb.WriteString(fmt.Sprintf("<i>%s ➜ %s IST</i>\n\n", since.In(istZone).Format("02 Jan 15:04"), time.Now().In(istZone).Format("02 Jan 15:04")))
	sb.WriteString("<blockquote>")
	sb.WriteString(line("➕", "Added in groups", added))
	sb.WriteString(line("➖", "Removed from groups", removed))
	sb.WriteString(line("📈", "Net groups", fmt.Sprintf("%+d", added-removed)))
	sb.WriteString(line("💬", "Total groups", len(chats)))
	sb.WriteString(line("👤", "Total users", len(users)))
	sb.WriteString(line("🆕", "New users", newUsers))
	sb.WriteString("</blockquote>\n<blockquote>")
	sb.WriteString(line("🎵", "Songs played", plays))
	sb.WriteString(line("🎧", "Active vcs now", active))
	sb.WriteString(line("📶", "Vc joins", vcJoin))
	sb.WriteString(line("📴", "Vc leaves", vcLeave))
	sb.WriteString(line("🔗", "Links removed", links))
	sb.WriteString(line("🧬", "Bio links removed", bios))
	sb.WriteString("</blockquote>")
	return sb.String()
}

func startOfTodayIST() time.Time {
	n := time.Now().In(istZone)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, istZone)
}

func dataHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}
	deleteCmd(c, m)

	cmd := strings.ToLower(strings.TrimPrefix(strings.Fields(m.Text())[0], "/"))
	cmd = strings.Split(cmd, "@")[0]
	arg := strings.ToLower(Args(m))

	if cmd == "bdata" && arg == "" || arg == "24h" {
		_, err := m.ReplyText(c, buildReport("Last 24 hours report", time.Now().Add(-24*time.Hour)), &td.SendMessageOpts{ParseMode: "HTML"})
		return err
	}
	_, err := m.ReplyText(c, buildReport("Today's report", startOfTodayIST()), &td.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// StartDailyJobs runs the 24h report (and the self promo broadcast) every day at
// AUTO_BROADCAST_HOUR (default 07:00 IST).
func StartDailyJobs(c *td.Client) {
	if !config.AutoBroadcast {
		return
	}
	go func() {
		for {
			now := time.Now().In(istZone)
			next := time.Date(now.Year(), now.Month(), now.Day(), config.AutoBroadcastHour, 0, 0, 0, istZone)
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next))
			runDailyJob(c)
		}
	}()
}

func runDailyJob(c *td.Client) {
	defer func() { _ = recover() }()

	report := buildReport("Daily 24h report", time.Now().Add(-24*time.Hour))
	for _, id := range []int64{config.OwnerId, config.LoggerId} {
		if id == 0 {
			continue
		}
		if _, err := c.SendMessage(id, report, &td.SendMessageOpts{ParseMode: "HTML"}); err != nil {
			c.Logger.Warn("daily report failed", "chat_id", id, "error", err)
		}
	}

	if config.AutoPromo {
		runPromo(c, func(text string) {
			if config.OwnerId != 0 {
				_, _ = c.SendMessage(config.OwnerId, text, &td.SendMessageOpts{ParseMode: "HTML"})
			}
		})
	}
}

func promoCaption(c *td.Client) string {
	return fmt.Sprintf(
		"<blockquote><b>🎶 %s</b></blockquote>\n"+
			"<blockquote>🎧 %s\n⚡ %s\n🛡 %s\n🔗 %s</blockquote>\n"+
			"<b>%s</b> /play <code>%s</code>",
		utils.SC(config.BotBrand+" premium music experience"),
		utils.SC("Crystal clear high quality streaming"),
		utils.SC("Fast, stable and always active"),
		utils.SC("Link and bio remover for clean chats"),
		utils.SC("Autoplay, playlists and daily stats"),
		utils.SC("Send"), utils.SC("song name"),
	)
}

func promoMarkup(c *td.Client) *td.ReplyMarkupInlineKeyboard {
	rows := [][]td.InlineKeyboardButton{
		{utils.RandURL("Add me to your group", "https://t.me/"+c.Me().Usernames.EditableUsername+"?startgroup=true")},
		{utils.RandURL("Updates", config.SupportChannel), utils.RandURL("Support", config.SupportGroup)},
	}
	return &td.ReplyMarkupInlineKeyboard{Rows: rows}
}

// runPromo sends the self promotion post to every served chat.
func runPromo(c *td.Client, notify func(string)) {
	if !broadcastInProgress.CompareAndSwap(false, true) {
		notify("A broadcast is already in progress.")
		return
	}
	defer broadcastInProgress.Store(false)
	broadcastCancelFlag.Store(false)

	chats, _ := db.Instance.GetAllChats()
	caption := promoCaption(c)
	markup := promoMarkup(c)
	photo := utils.ImageFile(config.StartImg)

	ok, failed := 0, 0
	for _, id := range chats {
		if broadcastCancelFlag.Load() {
			break
		}
		var err error
		if photo != nil {
			_, err = c.SendPhoto(id, photo, &td.SendPhotoOpts{Caption: caption, ParseMode: "HTML", ReplyMarkup: markup})
		} else {
			_, err = c.SendMessage(id, caption, &td.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: markup})
		}
		if err == nil {
			ok++
			time.Sleep(250 * time.Millisecond)
			continue
		}
		if wait := getFloodWait(err); wait > 0 {
			time.Sleep(time.Duration(wait+5) * time.Second)
		}
		failed++
	}

	notify(fmt.Sprintf("<blockquote><b>📣 %s</b>\n%s : <code>%d</code>\n%s : <code>%d</code></blockquote>",
		utils.SC("Self promo finished"), utils.SC("Sent"), ok, utils.SC("Failed"), failed))
}

func selfPromoHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}
	if strings.ToLower(Args(m)) != "run" {
		_, _ = m.ReplyText(c, "Usage: <code>/selfpromo run</code>", &td.SendMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}
	if broadcastInProgress.Load() {
		_, _ = m.ReplyText(c, "A broadcast is already in progress.", nil)
		return td.EndGroups
	}
	_, _ = m.ReplyText(c, "<b>"+utils.SC("Self promo started")+"</b>", &td.SendMessageOpts{ParseMode: "HTML"})
	go runPromo(c, func(text string) {
		_, _ = m.ReplyText(c, text, &td.SendMessageOpts{ParseMode: "HTML"})
	})
	return td.EndGroups
}
