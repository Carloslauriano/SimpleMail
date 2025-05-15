package server

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/carloslauriano/simpleEmail/config"
	"github.com/carloslauriano/simpleEmail/storage"
	"github.com/emersion/go-smtp"
)

// SMTPBackend implementa a interface smtp.Backend
type SMTPBackend struct {
	store storage.Storage
}

// NewSMTPBackend cria um novo backend SMTP
func NewSMTPBackend(store storage.Storage) *SMTPBackend {
	return &SMTPBackend{
		store: store,
	}
}

// Login implementa a autenticação SMTP
func (b *SMTPBackend) Login(state smtp.Session, username, password string) (smtp.Session, error) {
	user, err := b.store.AuthenticateUser(username, password)
	if err != nil {
		return nil, fmt.Errorf("autenticação falhou: %w", err)
	}

	return &SMTPSession{
		backend: b,
		user:    user,
	}, nil
}

// AnonymousLogin não é permitido
func (b *SMTPBackend) AnonymousLogin(state smtp.Session) (smtp.Session, error) {
	return nil, fmt.Errorf("login anônimo não permitido")
}

// SMTPSession implementa a interface smtp.Session
type SMTPSession struct {
	backend *SMTPBackend
	user    *storage.User
	from    string
	to      []string
}

// Mail inicia uma nova transação de email
func (s *SMTPSession) Mail(from string, opts smtp.MailOptions) error {
	s.from = from
	return nil
}

// Rcpt adiciona um destinatário
func (s *SMTPSession) Rcpt(to string) error {
	s.to = append(s.to, to)
	return nil
}

// Data processa o conteúdo do email
func (s *SMTPSession) Data(r io.Reader) error {
	// Ler o conteúdo do email
	body, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("falha ao ler email: %w", err)
	}

	// Obter a caixa de entrada do usuário
	mailbox, err := s.backend.store.GetMailbox(s.user.ID, "INBOX")
	if err != nil {
		return fmt.Errorf("falha ao obter caixa de entrada: %w", err)
	}

	// Criar a mensagem
	msg := &storage.Message{
		MailboxID: mailbox.ID,
		From:      s.from,
		To:        strings.Join(s.to, ","),
		Date:      time.Now(),
		Body:      string(body),
		RawData:   body,
		Size:      len(body),
	}

	// Salvar a mensagem
	if err := s.backend.store.CreateMessage(msg); err != nil {
		return fmt.Errorf("falha ao salvar mensagem: %w", err)
	}

	return nil
}

// Reset limpa o estado da sessão
func (s *SMTPSession) Reset() {
	s.from = ""
	s.to = nil
}

// Logout finaliza a sessão
func (s *SMTPSession) Logout() error {
	return nil
}

// StartSMTPServer inicia o servidor SMTP
func StartSMTPServer(cfg *config.Config, store storage.Storage) error {
	be := NewSMTPBackend(store)
	s := smtp.NewServer(be)

	s.Addr = fmt.Sprintf("%s:%d", cfg.SMTP.Address, cfg.SMTP.Port)
	s.Domain = cfg.SMTP.Domain
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024 // 1MB
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	log.Printf("Iniciando servidor SMTP em %s", s.Addr)
	return s.ListenAndServe()
} 