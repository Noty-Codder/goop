package bot

import (
	"IstkharMusic/internal/cache"
	"IstkharMusic/internal/config"
	"IstkharMusic/internal/db"
	"IstkharMusic/internal/utils"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
)

var bioLinkRe = regexp.MustCompile(`(?i)(https?://|www\.|t\.me/|telegram\.me/|@[a-z0-9_]{4,}|[a-z0-9-]+\.(com|net|org|in|me|io|xyz|info|link|live|co)\b)`)

type bioEntry struct {
	hasLink bool
	at      time.Time
}

var bioCache sync.Map

// canManage reports whether the sender may change restrictor settings.
func canManage(c *td.Client, m *td.Message) bool {
	if m.IsPrivate() {
		return false
	}
	if isDev(c, m) || db.Instance.IsAuthUser(m.ChatId, m.SenderID()) {
		return true
	}
	if mem, err := cache.GetUserAdmin(c, m.ChatId, m.SenderID(), false); err == nil && mem != nil {
		return true
	}
	_, _ = m.ReplyText(c, "You must be an administrator to use this command.", nil)
	return false
}

func exemptFromRestrictor(c *td.Client, chatID, userID int64) bool {
	for _, dev := range config.DEVS {
		if dev == userID {
			return true
		}
	}
	if db.Instance.IsApproved(chatID, userID) || db.Instance.IsAuthUser(chatID, userID) {
		return true
	}
	if mem, err := cache.GetUserAdmin(c, chatID, userID, false); err == nil && mem != nil {
		return true
	}
	if u, err := c.GetUser(userID); err == nil && u != nil {
		if _, isBot := u.Type.(*td.UserTypeBot); isBot {
			return true
		}
	}
	return false
}

func messageHasLink(m *td.Message) bool {
	for _, e := range m.GetEntities() {
		switch e.Type.(type) {
		case *td.TextEntityTypeUrl, *td.TextEntityTypeTextUrl:
			return true
		}
	}
	return false
}

func userBioHasLink(c *td.Client, userID int64) bool {
	if v, ok := bioCache.Load(userID); ok {
		e := v.(bioEntry)
		if time.Since(e.at) < 10*time.Minute {
			return e.hasLink
		}
	}
	info, err := c.GetUserFullInfo(userID)
	if err != nil || info == nil || info.Bio == nil {
		return false
	}
	has := bioLinkRe.MatchString(info.Bio.Text)
	bioCache.Store(userID, bioEntry{hasLink: has, at: time.Now()})
	return has
}

func tempNotice(c *td.Client, chatID int64, text string) {
	msg, err := c.SendMessage(chatID, text, &td.SendMessageOpts{ParseMode: "HTML", DisableWebPagePreview: true})
	if err != nil || msg == nil {
		return
	}
	id := msg.Id
	time.AfterFunc(7*time.Second, func() {
		_ = c.DeleteMessages(chatID, []int64{id}, &td.DeleteMessagesOpts{Revoke: true})
	})
}

func logRemoval(c *td.Client, kind string, chatID, userID int64, name string) {
	if config.LoggerId == 0 || chatID == config.LoggerId || !db.Instance.GetLoggerStatus() {
		return
	}
	text := fmt.Sprintf("<blockquote><b>🚫 %s</b></blockquote>\n<blockquote>%s ➜ <code>%d</code>\n%s ➜ <a href=\"tg://user?id=%d\">%s</a></blockquote>",
		utils.SC(kind), utils.SC("Chat"), chatID, utils.SC("User"), userID, html.EscapeString(name))
	_, _ = c.SendMessage(config.LoggerId, text, &td.SendMessageOpts{ParseMode: "HTML"})
}

func restrictorWatcher(c *td.Client, u *td.UpdateNewMessage) error {
	m := u.Message
	if m == nil || m.ChatId >= 0 {
		return nil
	}
	s := db.Instance.GetRestrictor(m.ChatId)
	if !s.LinkRemove && !s.BioRemove {
		return nil
	}
	uid := m.SenderID()
	if uid <= 0 || uid == c.Me().Id {
		return nil
	}

	linkHit := s.LinkRemove && messageHasLink(m)
	bioHit := !linkHit && s.BioRemove && userBioHasLink(c, uid)
	if !linkHit && !bioHit {
		return nil
	}
	if exemptFromRestrictor(c, m.ChatId, uid) {
		return nil
	}

	if err := m.Delete(c, &td.DeleteMessagesOpts{Revoke: true}); err != nil {
		return nil
	}

	name := firstName(c, m)
	mention := fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", uid, html.EscapeString(name))
	if linkHit {
		go db.Instance.AddEvent(db.EventLinkRemoved, m.ChatId)
		tempNotice(c, m.ChatId, fmt.Sprintf("<blockquote>🔗 %s, <b>%s</b></blockquote>", mention, utils.SC("links are not allowed here")))
		logRemoval(c, "Link removed", m.ChatId, uid, name)
	} else {
		go db.Instance.AddEvent(db.EventBioRemoved, m.ChatId)
		tempNotice(c, m.ChatId, fmt.Sprintf("<blockquote>🧬 %s, <b>%s</b></blockquote>", mention, utils.SC("remove the link from your bio to chat here")))
		logRemoval(c, "Bio link removed", m.ChatId, uid, name)
	}
	return nil
}

func toggleArg(arg string) (on, valid bool) {
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "on", "enable", "yes", "true":
		return true, true
	case "off", "disable", "no", "false":
		return false, true
	}
	return false, false
}

func onOff(v bool) string {
	if v {
		return "On ✅"
	}
	return "Off ❌"
}

func linkRemoveHandler(c *td.Client, m *td.Message) error {
	if !canManage(c, m) {
		return td.EndGroups
	}
	on, ok := toggleArg(Args(m))
	if !ok {
		_, _ = m.ReplyText(c, fmt.Sprintf("<blockquote><b>🔗 %s</b> : %s\n%s <code>/linkremove on|off</code></blockquote>",
			utils.SC("Link remover"), onOff(db.Instance.GetRestrictor(m.ChatId).LinkRemove), utils.SC("Usage")), &td.SendMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}
	_ = db.Instance.SetLinkRemove(m.ChatId, on)
	_, _ = m.ReplyText(c, fmt.Sprintf("<blockquote><b>🔗 %s</b> : %s</blockquote>", utils.SC("Link remover"), onOff(on)), &td.SendMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}

func bioRemoveHandler(c *td.Client, m *td.Message) error {
	if !canManage(c, m) {
		return td.EndGroups
	}
	on, ok := toggleArg(Args(m))
	if !ok {
		_, _ = m.ReplyText(c, fmt.Sprintf("<blockquote><b>🧬 %s</b> : %s\n%s <code>/bioremove on|off</code></blockquote>",
			utils.SC("Bio remover"), onOff(db.Instance.GetRestrictor(m.ChatId).BioRemove), utils.SC("Usage")), &td.SendMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}
	_ = db.Instance.SetBioRemove(m.ChatId, on)
	_, _ = m.ReplyText(c, fmt.Sprintf("<blockquote><b>🧬 %s</b> : %s</blockquote>", utils.SC("Bio remover"), onOff(on)), &td.SendMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}

func approveHandler(c *td.Client, m *td.Message) error {
	return setApproval(c, m, true)
}

func unapproveHandler(c *td.Client, m *td.Message) error {
	return setApproval(c, m, false)
}

func setApproval(c *td.Client, m *td.Message, approve bool) error {
	if !canManage(c, m) {
		return td.EndGroups
	}
	uid, err := getTargetUserID(c, m)
	if err != nil {
		_, _ = m.ReplyText(c, err.Error(), nil)
		return td.EndGroups
	}
	if err := db.Instance.SetApproved(m.ChatId, uid, approve); err != nil {
		_, _ = m.ReplyText(c, "Failed to update the approved list.", nil)
		return td.EndGroups
	}
	word := "approved ✅"
	if !approve {
		word = "unapproved ❌"
	}
	_, _ = m.ReplyText(c, fmt.Sprintf("<blockquote><a href=\"tg://user?id=%d\">%d</a> %s</blockquote>", uid, uid, utils.SC("is now "+word)), &td.SendMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}

func approvedListHandler(c *td.Client, m *td.Message) error {
	if !canManage(c, m) {
		return td.EndGroups
	}
	list := db.Instance.GetRestrictor(m.ChatId).Approved
	if len(list) == 0 {
		_, _ = m.ReplyText(c, "<blockquote>"+utils.SC("No approved users in this chat")+"</blockquote>", &td.SendMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}
	var sb strings.Builder
	sb.WriteString("<blockquote><b>✅ " + utils.SC("Approved users") + "</b></blockquote>\n<blockquote>")
	for _, id := range list {
		sb.WriteString(fmt.Sprintf("• <a href=\"tg://user?id=%d\">%d</a>\n", id, id))
	}
	sb.WriteString("</blockquote>")
	_, _ = m.ReplyText(c, sb.String(), &td.SendMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}
