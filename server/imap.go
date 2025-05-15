package server

import (
	"fmt"
	"log"
	"time"

	"github.com/carloslauriano/simpleEmail/config"
	"github.com/carloslauriano/simpleEmail/storage"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
)

// IMAPBackend implementa a interface backend.Backend
type IMAPBackend struct {
	store storage.Storage
}

// NewIMAPBackend cria um novo backend IMAP
func NewIMAPBackend(store storage.Storage) *IMAPBackend {
	return &IMAPBackend{
		store: store,
	}
}

// Login implementa a autenticação IMAP
func (b *IMAPBackend) Login(connInfo *imap.ConnInfo, username, password string) (backend.User, error) {
	user, err := b.store.AuthenticateUser(username, password)
	if err != nil {
		return nil, fmt.Errorf("autenticação falhou: %w", err)
	}

	return &IMAPUser{
		backend: b,
		user:    user,
	}, nil
}

// IMAPUser implementa a interface backend.User
type IMAPUser struct {
	backend *IMAPBackend
	user    *storage.User
}

// Username retorna o nome do usuário
func (u *IMAPUser) Username() string {
	return u.user.Username
}

// ListMailboxes lista as caixas de entrada do usuário
func (u *IMAPUser) ListMailboxes(subscribed bool) ([]backend.Mailbox, error) {
	mailboxes, err := u.backend.store.ListMailboxes(u.user.ID)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar caixas de entrada: %w", err)
	}

	result := make([]backend.Mailbox, len(mailboxes))
	for i, m := range mailboxes {
		result[i] = &IMAPMailbox{
			backend: u.backend,
			user:    u.user,
			mailbox: m,
		}
	}

	return result, nil
}

// GetMailbox obtém uma caixa de entrada específica
func (u *IMAPUser) GetMailbox(name string) (backend.Mailbox, error) {
	mailbox, err := u.backend.store.GetMailbox(u.user.ID, name)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter caixa de entrada: %w", err)
	}

	return &IMAPMailbox{
		backend: u.backend,
		user:    u.user,
		mailbox: mailbox,
	}, nil
}

// CreateMailbox cria uma nova caixa de entrada
func (u *IMAPUser) CreateMailbox(name string) error {
	mailbox := &storage.Mailbox{
		UserID: u.user.ID,
		Name:   name,
	}

	return u.backend.store.CreateMailbox(mailbox)
}

// DeleteMailbox remove uma caixa de entrada
func (u *IMAPUser) DeleteMailbox(name string) error {
	mailbox, err := u.backend.store.GetMailbox(u.user.ID, name)
	if err != nil {
		return fmt.Errorf("falha ao obter caixa de entrada: %w", err)
	}

	return u.backend.store.DeleteMailbox(mailbox.ID)
}

// RenameMailbox renomeia uma caixa de entrada
func (u *IMAPUser) RenameMailbox(existingName, newName string) error {
	mailbox, err := u.backend.store.GetMailbox(u.user.ID, existingName)
	if err != nil {
		return fmt.Errorf("falha ao obter caixa de entrada: %w", err)
	}

	mailbox.Name = newName
	return u.backend.store.UpdateMailbox(mailbox)
}

// Logout finaliza a sessão
func (u *IMAPUser) Logout() error {
	return nil
}

// IMAPMailbox implementa a interface backend.Mailbox
type IMAPMailbox struct {
	backend *IMAPBackend
	user    *storage.User
	mailbox *storage.Mailbox
}

// Name retorna o nome da caixa de entrada
func (m *IMAPMailbox) Name() string {
	return m.mailbox.Name
}

// Info retorna informações sobre a caixa de entrada
func (m *IMAPMailbox) Info() (*imap.MailboxInfo, error) {
	return &imap.MailboxInfo{
		Attributes: []string{},
		Delimiter:  "/",
		Name:       m.mailbox.Name,
	}, nil
}

// Status retorna o status da caixa de entrada
func (m *IMAPMailbox) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
	status := &imap.MailboxStatus{
		Name: m.mailbox.Name,
	}

	messages, err := m.backend.store.ListMessages(m.mailbox.ID)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar mensagens: %w", err)
	}

	for _, item := range items {
		switch item {
		case imap.StatusMessages:
			status.Messages = uint32(len(messages))
		case imap.StatusRecent:
			status.Recent = 0
		case imap.StatusUnseen:
			unseen := 0
			for _, msg := range messages {
				if !msg.Seen {
					unseen++
				}
			}
			status.Unseen = uint32(unseen)
		case imap.StatusUidNext:
			status.UidNext = uint32(len(messages) + 1)
		case imap.StatusUidValidity:
			status.UidValidity = 1
		}
	}

	return status, nil
}

// SetSubscribed marca a caixa de entrada como inscrita
func (m *IMAPMailbox) SetSubscribed(subscribed bool) error {
	return nil
}

// Check verifica a integridade da caixa de entrada
func (m *IMAPMailbox) Check() error {
	return nil
}

// ListMessages lista as mensagens da caixa de entrada
func (m *IMAPMailbox) ListMessages(uid bool, seqSet *imap.SeqSet, items []imap.FetchItem, ch chan<- *imap.Message) error {
	defer close(ch)

	messages, err := m.backend.store.ListMessages(m.mailbox.ID)
	if err != nil {
		return fmt.Errorf("falha ao listar mensagens: %w", err)
	}

	for i, msg := range messages {
		seqNum := uint32(i + 1)
		if seqSet.Contains(seqNum) {
			imapMsg := &imap.Message{
				SeqNum: seqNum,
				Uid:    uint32(msg.ID),
				Items:  make(map[imap.FetchItem]interface{}),
			}

			for _, item := range items {
				switch item {
				case imap.FetchEnvelope:
					imapMsg.Items[item] = &imap.Envelope{
						Date:      msg.Date,
						Subject:   msg.Subject,
						From:      []*imap.Address{{PersonalName: msg.From}},
						To:        []*imap.Address{{PersonalName: msg.To}},
						MessageId: fmt.Sprintf("%d", msg.ID),
					}
				case imap.FetchBody, imap.FetchBodyStructure:
					imapMsg.Items[item] = &imap.BodyStructure{
						MIMEType:    "text",
						MIMESubType: "plain",
						Size:        uint32(len(msg.Body)),
					}
				case imap.FetchFlags:
					flags := []string{}
					if msg.Seen {
						flags = append(flags, imap.SeenFlag)
					}
					if msg.Deleted {
						flags = append(flags, imap.DeletedFlag)
					}
					if msg.Draft {
						flags = append(flags, imap.DraftFlag)
					}
					imapMsg.Items[item] = flags
				case imap.FetchInternalDate:
					imapMsg.Items[item] = msg.Date
				case imap.FetchRFC822Size:
					imapMsg.Items[item] = msg.Size
				case imap.FetchUid:
					imapMsg.Items[item] = msg.ID
				}
			}

			ch <- imapMsg
		}
	}

	return nil
}

// SearchMessages pesquisa mensagens na caixa de entrada
func (m *IMAPMailbox) SearchMessages(uid bool, criteria *imap.SearchCriteria) ([]uint32, error) {
	messages, err := m.backend.store.ListMessages(m.mailbox.ID)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar mensagens: %w", err)
	}

	var results []uint32
	for i, msg := range messages {
		seqNum := uint32(i + 1)
		if criteria.SeqNum != nil && !criteria.SeqNum.Contains(seqNum) {
			continue
		}

		if criteria.Uid != nil && !criteria.Uid.Contains(uint32(msg.ID)) {
			continue
		}

		if criteria.Since != (time.Time{}) && msg.Date.Before(criteria.Since) {
			continue
		}

		if criteria.Before != (time.Time{}) && msg.Date.After(criteria.Before) {
			continue
		}

		matches := true

		if criteria.Seen && !msg.Seen {
			matches = false
		}

		if criteria.Unseen && msg.Seen {
			matches = false
		}

		if criteria.Deleted && !msg.Deleted {
			matches = false
		}

		if criteria.Undeleted && msg.Deleted {
			matches = false
		}

		if criteria.Draft && !msg.Draft {
			matches = false
		}

		if criteria.Undraft && msg.Draft {
			matches = false
		}

		if !matches {
			continue
		}

		if criteria.Header != nil {
			// Implementar pesquisa por cabeçalho
		}

		if criteria.Body != nil {
			// Implementar pesquisa por corpo
		}

		if criteria.Text != nil {
			// Implementar pesquisa por texto
		}

		if uid {
			results = append(results, uint32(msg.ID))
		} else {
			results = append(results, seqNum)
		}
	}

	return results, nil
}

// CreateMessage cria uma nova mensagem na caixa de entrada
func (m *IMAPMailbox) CreateMessage(flags []string, date time.Time, body imap.Literal) error {
	return nil
}

// UpdateMessagesFlags atualiza as flags das mensagens
func (m *IMAPMailbox) UpdateMessagesFlags(uid bool, seqSet *imap.SeqSet, operation imap.FlagsOp, flags []string) error {
	messages, err := m.backend.store.ListMessages(m.mailbox.ID)
	if err != nil {
		return fmt.Errorf("falha ao listar mensagens: %w", err)
	}

	for i, msg := range messages {
		seqNum := uint32(i + 1)
		if seqSet.Contains(seqNum) {
			seen := msg.Seen
			deleted := msg.Deleted
			draft := msg.Draft

			for _, flag := range flags {
				switch flag {
				case imap.SeenFlag:
					seen = true
				case imap.DeletedFlag:
					deleted = true
				case imap.DraftFlag:
					draft = true
				}
			}

			if err := m.backend.store.UpdateMessageFlags(msg.ID, "", seen, deleted, draft); err != nil {
				return fmt.Errorf("falha ao atualizar flags: %w", err)
			}
		}
	}

	return nil
}

// CopyMessages copia mensagens para outra caixa de entrada
func (m *IMAPMailbox) CopyMessages(uid bool, seqSet *imap.SeqSet, destName string) error {
	return nil
}

// Expunge remove mensagens marcadas como excluídas
func (m *IMAPMailbox) Expunge() error {
	messages, err := m.backend.store.ListMessages(m.mailbox.ID)
	if err != nil {
		return fmt.Errorf("falha ao listar mensagens: %w", err)
	}

	for _, msg := range messages {
		if msg.Deleted {
			if err := m.backend.store.DeleteMessage(msg.ID); err != nil {
				return fmt.Errorf("falha ao excluir mensagem: %w", err)
			}
		}
	}

	return nil
}

// StartIMAPServer inicia o servidor IMAP
func StartIMAPServer(cfg *config.Config, store storage.Storage) error {
	be := NewIMAPBackend(store)
	s := imap.NewServer(be)

	s.Addr = fmt.Sprintf("%s:%d", cfg.IMAP.Address, cfg.IMAP.Port)
	s.AllowInsecureAuth = true

	log.Printf("Iniciando servidor IMAP em %s", s.Addr)
	return s.ListenAndServe()
} 