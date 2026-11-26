package db

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Event types stored for the daily report.
const (
	EventChatAdd     = "chat_add"
	EventChatRemove  = "chat_remove"
	EventPlay        = "play"
	EventNewUser     = "new_user"
	EventLinkRemoved = "link_removed"
	EventBioRemoved  = "bio_removed"
	EventVCJoin      = "vc_join"
	EventVCLeave     = "vc_leave"
)

type eventDoc struct {
	Type string    `bson:"type"`
	Chat int64     `bson:"chat"`
	At   time.Time `bson:"at"`
}

func (db *Database) events() *mongo.Collection { return db.DB.Collection("events") }

// EnsureEventIndex keeps events for 10 days only.
func (db *Database) EnsureEventIndex() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = db.events().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(10 * 24 * 3600),
	})
}

// AddEvent records one event; errors are ignored on purpose.
func (db *Database) AddEvent(kind string, chatID int64) {
	if db == nil {
		return
	}
	ctx, cancel := db.ctx()
	defer cancel()
	_, _ = db.events().InsertOne(ctx, eventDoc{Type: kind, Chat: chatID, At: time.Now().UTC()})
}

// CountEvents returns how many events of a kind happened since the given time.
func (db *Database) CountEvents(kind string, since time.Time) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	n, err := db.events().CountDocuments(ctx, bson.M{"type": kind, "at": bson.M{"$gte": since.UTC()}})
	if err != nil {
		return 0
	}
	return n
}

// RestrictorSettings holds link/bio remover options for a chat.
type RestrictorSettings struct {
	ID         int64   `bson:"_id"`
	LinkRemove bool    `bson:"link_remove"`
	BioRemove  bool    `bson:"bio_remove"`
	Approved   []int64 `bson:"approved"`
	VCLogger   *bool   `bson:"vc_logger,omitempty"`
}

func (db *Database) restrictor() *mongo.Collection { return db.DB.Collection("restrictor") }

// GetRestrictor returns the settings of a chat (defaults when missing).
func (db *Database) GetRestrictor(chatID int64) *RestrictorSettings {
	if v, ok := restrictorCache.Load(chatID); ok {
		return v.(*RestrictorSettings)
	}
	ctx, cancel := db.ctx()
	defer cancel()
	s := &RestrictorSettings{ID: chatID}
	err := db.restrictor().FindOne(ctx, bson.M{"_id": chatID}).Decode(s)
	if err != nil {
		s = &RestrictorSettings{ID: chatID}
	}
	restrictorCache.Store(chatID, s)
	return s
}

func (db *Database) saveRestrictor(s *RestrictorSettings) error {
	ctx, cancel := db.ctx()
	defer cancel()
	_, err := db.restrictor().ReplaceOne(ctx, bson.M{"_id": s.ID}, s, options.Replace().SetUpsert(true))
	if err == nil {
		restrictorCache.Store(s.ID, s)
	}
	return err
}

// SetLinkRemove toggles the link remover.
func (db *Database) SetLinkRemove(chatID int64, on bool) error {
	s := *db.GetRestrictor(chatID)
	s.LinkRemove = on
	return db.saveRestrictor(&s)
}

// SetBioRemove toggles the bio remover.
func (db *Database) SetBioRemove(chatID int64, on bool) error {
	s := *db.GetRestrictor(chatID)
	s.BioRemove = on
	return db.saveRestrictor(&s)
}

// SetVCLogger toggles the voice chat logger for a chat.
func (db *Database) SetVCLogger(chatID int64, on bool) error {
	s := *db.GetRestrictor(chatID)
	s.VCLogger = &on
	return db.saveRestrictor(&s)
}

// VCLoggerEnabled reports whether VC join/leave notices are enabled (default on).
func (db *Database) VCLoggerEnabled(chatID int64) bool {
	s := db.GetRestrictor(chatID)
	return s.VCLogger == nil || *s.VCLogger
}

// IsApproved reports whether the user is exempt from the restrictor.
func (db *Database) IsApproved(chatID, userID int64) bool {
	for _, id := range db.GetRestrictor(chatID).Approved {
		if id == userID {
			return true
		}
	}
	return false
}

// SetApproved approves or unapproves a user.
func (db *Database) SetApproved(chatID, userID int64, approve bool) error {
	s := *db.GetRestrictor(chatID)
	list := make([]int64, 0, len(s.Approved)+1)
	for _, id := range s.Approved {
		if id != userID {
			list = append(list, id)
		}
	}
	if approve {
		list = append(list, userID)
	}
	s.Approved = list
	return db.saveRestrictor(&s)
}

var restrictorCache sync.Map
