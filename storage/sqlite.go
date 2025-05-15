package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/carloslauriano/simpleEmail/config"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage implementa a interface Storage para SQLite
type SQLiteStorage struct {
	db   *sql.DB
	path string
}

// NewSQLiteStorage cria uma nova instância de armazenamento SQLite
func NewSQLiteStorage(cfg *config.DatabaseConfig) (Storage, error) {
	// Garantir que o diretório existe
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("falha ao criar diretório para SQLite: %w", err)
	}

	return &SQLiteStorage{
		path: cfg.Path,
	}, nil
}

// Open abre a conexão com o banco de dados
func (s *SQLiteStorage) Open() error {
	db, err := sql.Open("sqlite3", s.path)
	if err != nil {
		return fmt.Errorf("falha ao abrir banco de dados SQLite: %w", err)
	}
	s.db = db

	if err := s.createSchema(); err != nil {
		s.db.Close()
		return fmt.Errorf("falha ao criar esquema SQLite: %w", err)
	}

	return nil
}

// Close fecha a conexão com o banco de dados
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// createSchema cria o esquema do banco de dados
func (s *SQLiteStorage) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		name TEXT,
		email TEXT NOT NULL UNIQUE,
		created DATETIME NOT NULL,
		updated DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS mailboxes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		path TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE(user_id, name)
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mailbox_id INTEGER NOT NULL,
		uid INTEGER NOT NULL,
		from_addr TEXT NOT NULL,
		to_addr TEXT NOT NULL,
		cc TEXT,
		subject TEXT,
		date DATETIME NOT NULL,
		body TEXT,
		raw_data BLOB,
		flags TEXT,
		size INTEGER NOT NULL,
		seen BOOLEAN NOT NULL DEFAULT 0,
		deleted BOOLEAN NOT NULL DEFAULT 0,
		draft BOOLEAN NOT NULL DEFAULT 0,
		created DATETIME NOT NULL,
		FOREIGN KEY (mailbox_id) REFERENCES mailboxes(id) ON DELETE CASCADE,
		UNIQUE(mailbox_id, uid)
	);

	CREATE TABLE IF NOT EXISTS attachments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id INTEGER NOT NULL,
		filename TEXT NOT NULL,
		mime_type TEXT NOT NULL,
		data BLOB NOT NULL,
		size INTEGER NOT NULL,
		created DATETIME NOT NULL,
		FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateUser cria um novo usuário
func (s *SQLiteStorage) CreateUser(user *User) error {
	now := time.Now()
	user.Created = now
	user.Updated = now

	result, err := s.db.Exec(
		"INSERT INTO users (username, password, name, email, created, updated) VALUES (?, ?, ?, ?, ?, ?)",
		user.Username, user.Password, user.Name, user.Email, user.Created, user.Updated,
	)
	if err != nil {
		return fmt.Errorf("falha ao criar usuário: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("falha ao obter ID do usuário: %w", err)
	}
	user.ID = id

	// Criar caixas padrão
	for _, name := range []string{"INBOX", "Sent", "Drafts", "Trash"} {
		mailbox := &Mailbox{
			UserID: user.ID,
			Name:   name,
			Path:   name,
		}
		if err := s.CreateMailbox(mailbox); err != nil {
			return fmt.Errorf("falha ao criar caixa de correio padrão: %w", err)
		}
	}

	return nil
}

// GetUser obtém um usuário pelo nome de usuário
func (s *SQLiteStorage) GetUser(username string) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, password, name, email, created, updated FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Name, &user.Email, &user.Created, &user.Updated)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, fmt.Errorf("falha ao obter usuário: %w", err)
	}

	return user, nil
}

// UpdateUser atualiza um usuário existente
func (s *SQLiteStorage) UpdateUser(user *User) error {
	user.Updated = time.Now()
	_, err := s.db.Exec(
		"UPDATE users SET password = ?, name = ?, email = ?, updated = ? WHERE id = ?",
		user.Password, user.Name, user.Email, user.Updated, user.ID,
	)
	if err != nil {
		return fmt.Errorf("falha ao atualizar usuário: %w", err)
	}
	return nil
}

// DeleteUser exclui um usuário
func (s *SQLiteStorage) DeleteUser(userID int64) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return fmt.Errorf("falha ao excluir usuário: %w", err)
	}
	return nil
}

// AuthenticateUser autentica um usuário
func (s *SQLiteStorage) AuthenticateUser(username, password string) (*User, error) {
	user, err := s.GetUser(username)
	if err != nil {
		return nil, err
	}

	// Aqui deveria verificar o hash da senha, mas para simplificar, comparamos diretamente
	if user.Password != password {
		return nil, fmt.Errorf("senha inválida")
	}

	return user, nil
}

// Métodos de implementação para Mailbox

func (s *SQLiteStorage) CreateMailbox(mailbox *Mailbox) error {
	result, err := s.db.Exec(
		"INSERT INTO mailboxes (user_id, name, path) VALUES (?, ?, ?)",
		mailbox.UserID, mailbox.Name, mailbox.Path,
	)
	if err != nil {
		return fmt.Errorf("falha ao criar caixa de correio: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("falha ao obter ID da caixa de correio: %w", err)
	}
	mailbox.ID = id

	return nil
}

func (s *SQLiteStorage) GetMailbox(userID int64, name string) (*Mailbox, error) {
	mailbox := &Mailbox{}
	err := s.db.QueryRow(
		"SELECT id, user_id, name, path FROM mailboxes WHERE user_id = ? AND name = ?",
		userID, name,
	).Scan(&mailbox.ID, &mailbox.UserID, &mailbox.Name, &mailbox.Path)

	if err == sql.ErrNoRows {
		return nil, ErrMailboxNotFound
	} else if err != nil {
		return nil, fmt.Errorf("falha ao obter caixa de correio: %w", err)
	}

	return mailbox, nil
}

func (s *SQLiteStorage) ListMailboxes(userID int64) ([]*Mailbox, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, name, path FROM mailboxes WHERE user_id = ?",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar caixas de correio: %w", err)
	}
	defer rows.Close()

	var mailboxes []*Mailbox
	for rows.Next() {
		mb := &Mailbox{}
		if err := rows.Scan(&mb.ID, &mb.UserID, &mb.Name, &mb.Path); err != nil {
			return nil, fmt.Errorf("falha ao ler dados da caixa de correio: %w", err)
		}
		mailboxes = append(mailboxes, mb)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar sobre caixas de correio: %w", err)
	}

	return mailboxes, nil
}

func (s *SQLiteStorage) DeleteMailbox(mailboxID int64) error {
	_, err := s.db.Exec("DELETE FROM mailboxes WHERE id = ?", mailboxID)
	if err != nil {
		return fmt.Errorf("falha ao excluir caixa de correio: %w", err)
	}
	return nil
}

// UpdateMailbox atualiza uma caixa de correio existente
func (s *SQLiteStorage) UpdateMailbox(mailbox *Mailbox) error {
	_, err := s.db.Exec(
		"UPDATE mailboxes SET name = ?, path = ? WHERE id = ?",
		mailbox.Name, mailbox.Path, mailbox.ID,
	)
	if err != nil {
		return fmt.Errorf("falha ao atualizar caixa de correio: %w", err)
	}
	return nil
}

// Implementações de Message

func (s *SQLiteStorage) CreateMessage(message *Message) error {
	message.Created = time.Now()
	result, err := s.db.Exec(
		`INSERT INTO messages 
		(mailbox_id, uid, from_addr, to_addr, cc, subject, date, body, raw_data, flags, size, seen, deleted, draft, created) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		message.MailboxID, message.UID, message.From, message.To, message.Cc, message.Subject, 
		message.Date, message.Body, message.RawData, message.Flags, message.Size, 
		message.Seen, message.Deleted, message.Draft, message.Created,
	)
	if err != nil {
		return fmt.Errorf("falha ao criar mensagem: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("falha ao obter ID da mensagem: %w", err)
	}
	message.ID = id

	return nil
}

func (s *SQLiteStorage) GetMessage(mailboxID int64, uid uint32) (*Message, error) {
	message := &Message{}
	err := s.db.QueryRow(
		`SELECT id, mailbox_id, uid, from_addr, to_addr, cc, subject, date, body, raw_data, 
		flags, size, seen, deleted, draft, created FROM messages 
		WHERE mailbox_id = ? AND uid = ?`,
		mailboxID, uid,
	).Scan(
		&message.ID, &message.MailboxID, &message.UID, &message.From, &message.To, 
		&message.Cc, &message.Subject, &message.Date, &message.Body, &message.RawData,
		&message.Flags, &message.Size, &message.Seen, &message.Deleted, &message.Draft, &message.Created,
	)

	if err == sql.ErrNoRows {
		return nil, ErrMessageNotFound
	} else if err != nil {
		return nil, fmt.Errorf("falha ao obter mensagem: %w", err)
	}

	return message, nil
}

func (s *SQLiteStorage) ListMessages(mailboxID int64) ([]*Message, error) {
	rows, err := s.db.Query(
		`SELECT id, mailbox_id, uid, from_addr, to_addr, cc, subject, date, body, raw_data, 
		flags, size, seen, deleted, draft, created FROM messages 
		WHERE mailbox_id = ? ORDER BY uid`,
		mailboxID,
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar mensagens: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		if err := rows.Scan(
			&msg.ID, &msg.MailboxID, &msg.UID, &msg.From, &msg.To, 
			&msg.Cc, &msg.Subject, &msg.Date, &msg.Body, &msg.RawData,
			&msg.Flags, &msg.Size, &msg.Seen, &msg.Deleted, &msg.Draft, &msg.Created,
		); err != nil {
			return nil, fmt.Errorf("falha ao ler dados da mensagem: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar sobre mensagens: %w", err)
	}

	return messages, nil
}

func (s *SQLiteStorage) UpdateMessageFlags(messageID int64, flags string, seen, deleted, draft bool) error {
	_, err := s.db.Exec(
		"UPDATE messages SET flags = ?, seen = ?, deleted = ?, draft = ? WHERE id = ?",
		flags, seen, deleted, draft, messageID,
	)
	if err != nil {
		return fmt.Errorf("falha ao atualizar flags da mensagem: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) DeleteMessage(messageID int64) error {
	_, err := s.db.Exec("DELETE FROM messages WHERE id = ?", messageID)
	if err != nil {
		return fmt.Errorf("falha ao excluir mensagem: %w", err)
	}
	return nil
}

// Implementações de Attachment

func (s *SQLiteStorage) CreateAttachment(attachment *Attachment) error {
	attachment.Created = time.Now()
	result, err := s.db.Exec(
		"INSERT INTO attachments (message_id, filename, mime_type, data, size, created) VALUES (?, ?, ?, ?, ?, ?)",
		attachment.MessageID, attachment.Filename, attachment.MimeType, attachment.Data, attachment.Size, attachment.Created,
	)
	if err != nil {
		return fmt.Errorf("falha ao criar anexo: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("falha ao obter ID do anexo: %w", err)
	}
	attachment.ID = id

	return nil
}

func (s *SQLiteStorage) GetAttachments(messageID int64) ([]*Attachment, error) {
	rows, err := s.db.Query(
		"SELECT id, message_id, filename, mime_type, data, size, created FROM attachments WHERE message_id = ?",
		messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter anexos: %w", err)
	}
	defer rows.Close()

	var attachments []*Attachment
	for rows.Next() {
		att := &Attachment{}
		if err := rows.Scan(&att.ID, &att.MessageID, &att.Filename, &att.MimeType, &att.Data, &att.Size, &att.Created); err != nil {
			return nil, fmt.Errorf("falha ao ler dados do anexo: %w", err)
		}
		attachments = append(attachments, att)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar sobre anexos: %w", err)
	}

	return attachments, nil
}

func (s *SQLiteStorage) DeleteAttachment(attachmentID int64) error {
	_, err := s.db.Exec("DELETE FROM attachments WHERE id = ?", attachmentID)
	if err != nil {
		return fmt.Errorf("falha ao excluir anexo: %w", err)
	}
	return nil
} 