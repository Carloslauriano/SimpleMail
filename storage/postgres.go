package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/carloslauriano/simpleEmail/config"
	_ "github.com/lib/pq"
)

// PostgresStorage implementa a interface Storage para PostgreSQL
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage cria uma nova instância de armazenamento PostgreSQL
func NewPostgresStorage(cfg *config.DatabaseConfig) (Storage, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir banco de dados PostgreSQL: %w", err)
	}

	return &PostgresStorage{
		db: db,
	}, nil
}

// Open abre a conexão com o banco de dados
func (s *PostgresStorage) Open() error {
	if err := s.createSchema(); err != nil {
		return fmt.Errorf("falha ao criar esquema PostgreSQL: %w", err)
	}
	return nil
}

// Close fecha a conexão com o banco de dados
func (s *PostgresStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// createSchema cria o esquema do banco de dados
func (s *PostgresStorage) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) NOT NULL UNIQUE,
		password VARCHAR(255) NOT NULL,
		name VARCHAR(255),
		email VARCHAR(255) NOT NULL UNIQUE,
		created TIMESTAMP NOT NULL,
		updated TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS mailboxes (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		path VARCHAR(255) NOT NULL,
		UNIQUE(user_id, name)
	);

	CREATE TABLE IF NOT EXISTS messages (
		id SERIAL PRIMARY KEY,
		mailbox_id INTEGER NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
		uid INTEGER NOT NULL,
		from_addr VARCHAR(255) NOT NULL,
		to_addr VARCHAR(255) NOT NULL,
		cc TEXT,
		subject TEXT,
		date TIMESTAMP NOT NULL,
		body TEXT,
		raw_data BYTEA,
		flags TEXT,
		size INTEGER NOT NULL,
		seen BOOLEAN NOT NULL DEFAULT FALSE,
		deleted BOOLEAN NOT NULL DEFAULT FALSE,
		draft BOOLEAN NOT NULL DEFAULT FALSE,
		created TIMESTAMP NOT NULL,
		UNIQUE(mailbox_id, uid)
	);

	CREATE TABLE IF NOT EXISTS attachments (
		id SERIAL PRIMARY KEY,
		message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
		filename VARCHAR(255) NOT NULL,
		mime_type VARCHAR(255) NOT NULL,
		data BYTEA NOT NULL,
		size INTEGER NOT NULL,
		created TIMESTAMP NOT NULL
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateUser cria um novo usuário
func (s *PostgresStorage) CreateUser(user *User) error {
	now := time.Now()
	user.Created = now
	user.Updated = now

	var id int64
	err := s.db.QueryRow(
		"INSERT INTO users (username, password, name, email, created, updated) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		user.Username, user.Password, user.Name, user.Email, user.Created, user.Updated,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("falha ao criar usuário: %w", err)
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
func (s *PostgresStorage) GetUser(username string) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, password, name, email, created, updated FROM users WHERE username = $1",
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
func (s *PostgresStorage) UpdateUser(user *User) error {
	user.Updated = time.Now()
	_, err := s.db.Exec(
		"UPDATE users SET password = $1, name = $2, email = $3, updated = $4 WHERE id = $5",
		user.Password, user.Name, user.Email, user.Updated, user.ID,
	)
	if err != nil {
		return fmt.Errorf("falha ao atualizar usuário: %w", err)
	}
	return nil
}

// DeleteUser exclui um usuário
func (s *PostgresStorage) DeleteUser(userID int64) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		return fmt.Errorf("falha ao excluir usuário: %w", err)
	}
	return nil
}

// AuthenticateUser autentica um usuário
func (s *PostgresStorage) AuthenticateUser(username, password string) (*User, error) {
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

func (s *PostgresStorage) CreateMailbox(mailbox *Mailbox) error {
	var id int64
	err := s.db.QueryRow(
		"INSERT INTO mailboxes (user_id, name, path) VALUES ($1, $2, $3) RETURNING id",
		mailbox.UserID, mailbox.Name, mailbox.Path,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("falha ao criar caixa de correio: %w", err)
	}
	mailbox.ID = id
	return nil
}

func (s *PostgresStorage) GetMailbox(userID int64, name string) (*Mailbox, error) {
	mailbox := &Mailbox{}
	err := s.db.QueryRow(
		"SELECT id, user_id, name, path FROM mailboxes WHERE user_id = $1 AND name = $2",
		userID, name,
	).Scan(&mailbox.ID, &mailbox.UserID, &mailbox.Name, &mailbox.Path)

	if err == sql.ErrNoRows {
		return nil, ErrMailboxNotFound
	} else if err != nil {
		return nil, fmt.Errorf("falha ao obter caixa de correio: %w", err)
	}

	return mailbox, nil
}

func (s *PostgresStorage) ListMailboxes(userID int64) ([]*Mailbox, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, name, path FROM mailboxes WHERE user_id = $1",
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

func (s *PostgresStorage) DeleteMailbox(mailboxID int64) error {
	_, err := s.db.Exec("DELETE FROM mailboxes WHERE id = $1", mailboxID)
	if err != nil {
		return fmt.Errorf("falha ao excluir caixa de correio: %w", err)
	}
	return nil
}

// UpdateMailbox atualiza uma caixa de correio existente
func (s *PostgresStorage) UpdateMailbox(mailbox *Mailbox) error {
	_, err := s.db.Exec(
		"UPDATE mailboxes SET name = $1, path = $2 WHERE id = $3",
		mailbox.Name, mailbox.Path, mailbox.ID,
	)
	if err != nil {
		return fmt.Errorf("falha ao atualizar caixa de correio: %w", err)
	}
	return nil
}

// Implementações de Message

func (s *PostgresStorage) CreateMessage(message *Message) error {
	message.Created = time.Now()
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO messages 
		(mailbox_id, uid, from_addr, to_addr, cc, subject, date, body, raw_data, flags, size, seen, deleted, draft, created) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING id`,
		message.MailboxID, message.UID, message.From, message.To, message.Cc, message.Subject, 
		message.Date, message.Body, message.RawData, message.Flags, message.Size, 
		message.Seen, message.Deleted, message.Draft, message.Created,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("falha ao criar mensagem: %w", err)
	}
	message.ID = id
	return nil
}

func (s *PostgresStorage) GetMessage(mailboxID int64, uid uint32) (*Message, error) {
	message := &Message{}
	err := s.db.QueryRow(
		`SELECT id, mailbox_id, uid, from_addr, to_addr, cc, subject, date, body, raw_data, 
		flags, size, seen, deleted, draft, created FROM messages 
		WHERE mailbox_id = $1 AND uid = $2`,
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

func (s *PostgresStorage) ListMessages(mailboxID int64) ([]*Message, error) {
	rows, err := s.db.Query(
		`SELECT id, mailbox_id, uid, from_addr, to_addr, cc, subject, date, body, raw_data, 
		flags, size, seen, deleted, draft, created FROM messages 
		WHERE mailbox_id = $1 ORDER BY uid`,
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

func (s *PostgresStorage) UpdateMessageFlags(messageID int64, flags string, seen, deleted, draft bool) error {
	_, err := s.db.Exec(
		"UPDATE messages SET flags = $1, seen = $2, deleted = $3, draft = $4 WHERE id = $5",
		flags, seen, deleted, draft, messageID,
	)
	if err != nil {
		return fmt.Errorf("falha ao atualizar flags da mensagem: %w", err)
	}
	return nil
}

func (s *PostgresStorage) DeleteMessage(messageID int64) error {
	_, err := s.db.Exec("DELETE FROM messages WHERE id = $1", messageID)
	if err != nil {
		return fmt.Errorf("falha ao excluir mensagem: %w", err)
	}
	return nil
}

// Implementações de Attachment

func (s *PostgresStorage) CreateAttachment(attachment *Attachment) error {
	attachment.Created = time.Now()
	var id int64
	err := s.db.QueryRow(
		"INSERT INTO attachments (message_id, filename, mime_type, data, size, created) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		attachment.MessageID, attachment.Filename, attachment.MimeType, attachment.Data, attachment.Size, attachment.Created,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("falha ao criar anexo: %w", err)
	}
	attachment.ID = id
	return nil
}

func (s *PostgresStorage) GetAttachments(messageID int64) ([]*Attachment, error) {
	rows, err := s.db.Query(
		"SELECT id, message_id, filename, mime_type, data, size, created FROM attachments WHERE message_id = $1",
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

func (s *PostgresStorage) DeleteAttachment(attachmentID int64) error {
	_, err := s.db.Exec("DELETE FROM attachments WHERE id = $1", attachmentID)
	if err != nil {
		return fmt.Errorf("falha ao excluir anexo: %w", err)
	}
	return nil
} 