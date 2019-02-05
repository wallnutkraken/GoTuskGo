// Package dbwrap contains all the GORM database methods for GoTuskGo, these functions should only
// be accessed through an interface, defined by the consumer of this package
package dbwrap

import (
	"time"

	"github.com/jinzhu/gorm"
)

const (
	// GeneralOffset is the name of the General option for the telegram offset
	GeneralOffset = "telegram_offset"
)

// Wrapper is the GORM wrapper containing all GoTuskGo database methods
type Wrapper struct {
	db *gorm.DB
}

// New created a new instance of the database Wrapper
func New(db *gorm.DB) Wrapper {
	return Wrapper{
		db: db,
	}
}

// AutoMigrate runs the AutoMigrate GORM tool
func (w Wrapper) AutoMigrate() error {
	return w.db.Set("gorm:table_options", "CHARSET=utf8mb4").AutoMigrate(&General{}, &Message{}, &Subscription{}, &SubscribeError{}).Error
}

// GetOffset gets the current offset
func (w Wrapper) GetOffset() int {
	offset := General{}
	if err := w.db.Where(&General{Name: GeneralOffset}).First(&offset).Error; err != nil {
		// Default to 0, row might not exist yet
		return 0
	}
	return offset.Value
}

// SetOffset sets the given offset as current
func (w Wrapper) SetOffset(value int) error {
	offset := General{
		Name:  GeneralOffset,
		Value: value,
	}
	return w.db.Save(&offset).Error
}

// AddMessage adds a given message to the messages list in the database
func (w Wrapper) AddMessage(msg string) error {
	message := Message{
		Content: msg,
	}
	return w.db.Save(&message).Error
}

// GetAllMessages returns all messages
func (w Wrapper) GetAllMessages() ([]Message, error) {
	msg := []Message{}
	return msg, w.db.Find(&msg).Error
}

// GetSubscription returns a subscription, if found
func (w Wrapper) GetSubscription(chatID int64) (Subscription, error) {
	sub := Subscription{}
	return sub, w.db.Where(&Subscription{ChatID: chatID}).First(&sub).Error
}

// GetSubscriptions returns all subscriptions
func (w Wrapper) GetSubscriptions() ([]Subscription, error) {
	sub := []Subscription{}
	return sub, w.db.Find(&sub).Error
}

// AddSubscription creates a new subscription
func (w Wrapper) AddSubscription(chatID int64) error {
	sub := Subscription{
		ChatID: chatID,
	}
	return w.db.Create(&sub).Error
}

// Unsubscribe removes a subscription
func (w Wrapper) Unsubscribe(sub Subscription) error {
	return w.db.Delete(&sub).Error
}

// AddSubscribeError creates a new subscription error row
func (w Wrapper) AddSubscribeError(chatID int64, message string) error {
	subErr := SubscribeError{
		ChatID: chatID,
		Error:  message,
		Unix:   time.Now().Unix(),
	}
	return w.db.Save(&subErr).Error
}

// GetSubscribeErrors returns all subscribe errors
func (w Wrapper) GetSubscribeErrors() ([]SubscribeError, error) {
	sub := []SubscribeError{}
	return sub, w.db.Find(&sub).Error
}

// PurgeSubscribeErrors deletes all subscribe errors
func (w Wrapper) PurgeSubscribeErrors() error {
	return w.db.Delete(&SubscribeError{}).Error
}
