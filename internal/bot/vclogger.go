package bot

import (
	"IstkharMusic/internal/calls"
	"IstkharMusic/internal/db"
	"IstkharMusic/internal/utils"
	"fmt"
	"html"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
)

const vcNoticeDelay = 7 * time.Second

type vcKey struct{ chat, user int64 }

type vcLogger struct {
	mu     sync.Mutex
	joins  map[vcKey]int
	recent map[int64][]time.Time
}

var vcLog = &vcLogger{joins: map[vcKey]int{}, recent: map[int64][]time.Time{}}

// ClearChat forgets join counters when a voice chat ends.
func (l *vcLogger) ClearChat(chatID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k := range l.joins {
		if k.chat == chatID {
			delete(l.joins, k)
		}
	}
	delete(l.recent, chatID)
}

// allow limits notices to 20 per minute per chat so big voice chats cannot flood.
func (l *vcLogger) allow(chatID int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := time.Now().Add(-time.Minute)
	kept := l.recent[chatID][:0]
	for _, t := range l.recent[chatID] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= 20 {
		l.recent[chatID] = kept
		return false
	}
	l.recent[chatID] = append(kept, time.Now())
	return true
}

func (l *vcLogger) notify(c *td.Client, chatID, userID int64, joined bool) {
	defer func() { _ = recover() }()

	if userID <= 0 || userID == c.Me().Id || !db.Instance.VCLoggerEnabled(chatID) || !l.allow(chatID) {
		return
	}

	if joined {
		go db.Instance.AddEvent(db.EventVCJoin, chatID)
	} else {
		go db.Instance.AddEvent(db.EventVCLeave, chatID)
	}

	name, username, role := "User", "Ignored", "Member"
	if u, err := c.GetUser(userID); err == nil && u != nil {
		name = u.FirstName
		if u.LastName != "" {
			name += " " + u.LastName
		}
		if u.Usernames != nil && u.Usernames.EditableUsername != "" {
			username = "@" + u.Usernames.EditableUsername
		}
	}
	if mem, err := c.GetChatMember(chatID, &td.MessageSenderUser{UserId: userID}); err == nil && mem != nil {
		switch mem.Status.(type) {
		case *td.ChatMemberStatusCreator:
			role = "Owner"
		case *td.ChatMemberStatusAdministrator:
			role = "Admin"
		}
	}

	tag, action := "#LeaveVideoChat", "Left"
	if joined {
		tag, action = "#JoinVideoChat", "Joined"
	}

	text := fmt.Sprintf(
		"<b>%s</b>\n\n<blockquote>%s ➜ <a href=\"tg://user?id=%d\">%s</a>\n%s ➜ <code>%d</code>\n%s ➜ %s\n%s ➜ %s [%s]</blockquote>",
		tag,
		utils.SC("Name"), userID, html.EscapeString(name),
		utils.SC("Id"), userID,
		utils.SC("Username"), html.EscapeString(username),
		utils.SC("Action"), action, role,
	)

	if joined {
		l.mu.Lock()
		l.joins[vcKey{chatID, userID}]++
		n := l.joins[vcKey{chatID, userID}]
		l.mu.Unlock()
		text += fmt.Sprintf("\n🔄 <b>%s</b> ➜ <code>%d</code>", utils.SC("Join count"), n)
	}

	opts := &td.SendMessageOpts{ParseMode: "HTML", DisableWebPagePreview: true}
	if link := vcLink(c, chatID); link != "" {
		opts.ReplyMarkup = &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{{utils.RandURL("Join live vc 📶", link)}}}
	}

	msg, err := c.SendMessage(chatID, text, opts)
	if err != nil || msg == nil {
		return
	}
	id := msg.Id
	time.AfterFunc(vcNoticeDelay, func() {
		_ = c.DeleteMessages(chatID, []int64{id}, &td.DeleteMessagesOpts{Revoke: true})
	})
}

func vcLink(c *td.Client, chatID int64) string {
	sg, err := getSupergroup(c, chatID)
	if err != nil || sg == nil || sg.Usernames == nil || sg.Usernames.EditableUsername == "" {
		return ""
	}
	return "https://t.me/" + sg.Usernames.EditableUsername + "?videochat"
}

// registerVCLogger connects the assistant participant updates to the notifier.
func registerVCLogger(c *td.Client) {
	calls.Calls.SetParticipantHook(func(chatID, userID int64, joined bool) {
		go vcLog.notify(c, chatID, userID, joined)
	})
}

func vcLoggerHandler(c *td.Client, m *td.Message) error {
	if !canManage(c, m) {
		return td.EndGroups
	}
	on, ok := toggleArg(Args(m))
	if !ok {
		_, _ = m.ReplyText(c, fmt.Sprintf("<blockquote><b>📶 %s</b> : %s\n%s <code>/vclogger on|off</code></blockquote>",
			utils.SC("Vc logger"), onOff(db.Instance.VCLoggerEnabled(m.ChatId)), utils.SC("Usage")), &td.SendMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}
	_ = db.Instance.SetVCLogger(m.ChatId, on)
	_, _ = m.ReplyText(c, fmt.Sprintf("<blockquote><b>📶 %s</b> : %s</blockquote>", utils.SC("Vc logger"), onOff(on)), &td.SendMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}
