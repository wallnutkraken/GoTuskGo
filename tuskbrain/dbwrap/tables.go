package dbwrap

// General contains the general values for operation, such as the current Telegram message offset
type General struct {
	Name  string `gorm:"primary_key"`
	Value int    `gorm:"not null"`
}

// Message contains a stored message from Telegram, anonymized
type Message struct {
	ID      int    `gorm:"primary_key"`
	Content string `gorm:"not null"`
}

// Subscription contains a subscibed chat ID
type Subscription struct {
	ID     int   `gorm:"primary_key"`
	ChatID int64 `gorm:"not null"`
}

// SubscribeError is an error relating to subscriptions
type SubscribeError struct {
	ID     int    `gorm:"primary_key"`
	ChatID int64  `gorm:"not null"`
	Error  string `gorm:"not null"`
	Unix   int64  `gorm:"not null"`
}
