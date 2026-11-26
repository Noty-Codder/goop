package utils

import (
	"math/rand"
	"strings"
	"unicode"

	"github.com/AshokShau/gotdbot"
)

var smallCaps = map[rune]rune{
	'a': 'ᴀ', 'b': 'ʙ', 'c': 'ᴄ', 'd': 'ᴅ', 'e': 'ᴇ', 'f': 'ꜰ', 'g': 'ɢ', 'h': 'ʜ', 'i': 'ɪ',
	'j': 'ᴊ', 'k': 'ᴋ', 'l': 'ʟ', 'm': 'ᴍ', 'n': 'ɴ', 'o': 'ᴏ', 'p': 'ᴘ', 'q': 'ǫ', 'r': 'ʀ',
	's': 'ꜱ', 't': 'ᴛ', 'u': 'ᴜ', 'v': 'ᴠ', 'w': 'ᴡ', 'x': 'x', 'y': 'ʏ', 'z': 'ᴢ',
}

// SC converts plain text to the "Sᴍᴀʟʟ Cᴀᴘs" font: the first letter of every word
// stays a capital, the remaining letters become small capitals.
func SC(s string) string {
	var b strings.Builder
	inWord := false
	for _, r := range s {
		if r > unicode.MaxASCII || !unicode.IsLetter(r) {
			inWord = unicode.IsLetter(r) || unicode.IsDigit(r)
			b.WriteRune(r)
			continue
		}
		if inWord {
			b.WriteRune(smallCaps[unicode.ToLower(r)])
		} else {
			b.WriteRune(unicode.ToUpper(r))
		}
		inWord = true
	}
	return b.String()
}

// SCHTML applies SC to the text nodes of an HTML string. Tags, entities and the
// content of <code>, <pre> and <a> elements are left untouched so commands,
// ids and links keep working.
func SCHTML(s string) string {
	var out, text strings.Builder
	flush := func() {
		if text.Len() > 0 {
			out.WriteString(SC(text.String()))
			text.Reset()
		}
	}
	skip := 0
	for i := 0; i < len(s); {
		switch s[i] {
		case '<':
			end := strings.IndexByte(s[i:], '>')
			if end < 0 {
				text.WriteString(s[i:])
				i = len(s)
				continue
			}
			flush()
			tag := s[i : i+end+1]
			low := strings.ToLower(tag)
			switch {
			case strings.HasPrefix(low, "<code"), strings.HasPrefix(low, "<pre"), strings.HasPrefix(low, "<a "):
				skip++
			case strings.HasPrefix(low, "</code"), strings.HasPrefix(low, "</pre"), strings.HasPrefix(low, "</a"):
				if skip > 0 {
					skip--
				}
			}
			out.WriteString(tag)
			i += end + 1
		case '&':
			end := strings.IndexByte(s[i:], ';')
			if end > 0 && end < 10 {
				flush()
				out.WriteString(s[i : i+end+1])
				i += end + 1
				continue
			}
			text.WriteByte(s[i])
			i++
		default:
			if skip > 0 {
				out.WriteByte(s[i])
				i++
				continue
			}
			j := i
			for j < len(s) && s[j] != '<' && s[j] != '&' {
				j++
			}
			text.WriteString(s[i:j])
			i = j
		}
	}
	flush()
	return out.String()
}

// RandStyle returns a random coloured button style.
func RandStyle() gotdbot.ButtonStyle {
	switch rand.Intn(3) {
	case 0:
		return gotdbot.ButtonStylePrimary{}
	case 1:
		return gotdbot.ButtonStyleSuccess{}
	default:
		return gotdbot.ButtonStyleDanger{}
	}
}

// RandBtn builds a callback button with the small-caps font and a random colour.
func RandBtn(text, data string) gotdbot.InlineKeyboardButton {
	return cb(SC(text), data, RandStyle())
}

// RandURL builds a URL button with the small-caps font and a random colour.
func RandURL(text, link string) gotdbot.InlineKeyboardButton {
	return url(SC(text), link, RandStyle())
}
