package storage

import (
	"time"
)

// User representa um usuário do sistema de email
type User struct {
	ID       int64
	Username string
	Password string // Deve ser armazenado com hash
	Name     string
	Email    string
	Created  time.Time
	Updated  time.Time
}

// Mailbox representa uma caixa de email
type Mailbox struct {
	ID     int64
	UserID int64
	Name   string
	Path   string
}

// Message representa uma mensagem de email
type Message struct {
	ID        int64
	MailboxID int64
	UID       uint32
	From      string
	To        string
	Cc        string
	Subject   string
	Date      time.Time
	Body      string
	RawData   []byte
	Flags     string // Armazena flags como \\Seen, \\Answered, etc.
	Size      int
	Seen      bool
	Deleted   bool
	Draft     bool
	Created   time.Time
}

// Attachment representa um anexo de email
type Attachment struct {
	ID        int64
	MessageID int64
	Filename  string
	MimeType  string
	Data      []byte
	Size      int
	Created   time.Time
} 