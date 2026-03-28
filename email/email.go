package email

import (
	"errors"

	"avenue/backend/shared"
)

// Message represents an outbound email.
type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// Sender is the interface for sending emails. Implementations can be swapped
// by reassigning the Default variable before use.
type Sender interface {
	Send(msg Message) error
}

// Default is the package-level sender, set during application initialization.
var Default Sender

// globalEmailTo overrides the To address on all outbound messages when set.
var globalEmailTo = shared.GetEnv("GLOBAL_EMAIL_TO", "")

var NotConfigured = errors.New("sender is not configured")

// Send delivers msg via Default, overriding the To address with GLOBAL_EMAIL_TO
// if that variable is set.
func Send(msg Message) error {
	if Default == nil {
		return NotConfigured
	}

	if globalEmailTo != "" {
		msg.To = globalEmailTo
	}

	return Default.Send(msg)
}
