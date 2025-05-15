package storage

import (
	"errors"
	"fmt"

	"github.com/carloslauriano/simpleEmail/config"
)

// ErrUserNotFound é retornado quando um usuário não é encontrado
var ErrUserNotFound = errors.New("usuário não encontrado")

// ErrMailboxNotFound é retornado quando uma caixa de correio não é encontrada
var ErrMailboxNotFound = errors.New("caixa de correio não encontrada")

// ErrMessageNotFound é retornado quando uma mensagem não é encontrada
var ErrMessageNotFound = errors.New("mensagem não encontrada")

// Storage é a interface para operações de armazenamento
type Storage interface {
	// Métodos de inicialização
	Open() error
	Close() error

	// Métodos de usuário
	CreateUser(user *User) error
	GetUser(username string) (*User, error)
	UpdateUser(user *User) error
	DeleteUser(userID int64) error
	AuthenticateUser(username, password string) (*User, error)

	// Métodos de caixa de correio
	CreateMailbox(mailbox *Mailbox) error
	GetMailbox(userID int64, name string) (*Mailbox, error)
	ListMailboxes(userID int64) ([]*Mailbox, error)
	DeleteMailbox(mailboxID int64) error
	UpdateMailbox(mailbox *Mailbox) error

	// Métodos de mensagem
	CreateMessage(message *Message) error
	GetMessage(mailboxID int64, uid uint32) (*Message, error)
	ListMessages(mailboxID int64) ([]*Message, error)
	UpdateMessageFlags(messageID int64, flags string, seen, deleted, draft bool) error
	DeleteMessage(messageID int64) error
	
	// Métodos de anexo
	CreateAttachment(attachment *Attachment) error
	GetAttachments(messageID int64) ([]*Attachment, error)
	DeleteAttachment(attachmentID int64) error
}

// NewStorage cria uma nova instância de armazenamento com base na configuração
func NewStorage(cfg *config.Config) (Storage, error) {
	switch cfg.Database.Type {
	case "sqlite":
		return NewSQLiteStorage(&cfg.Database)
	case "postgres":
		return NewPostgresStorage(&cfg.Database)
	default:
		return nil, fmt.Errorf("tipo de banco de dados não suportado: %s", cfg.Database.Type)
	}
} 