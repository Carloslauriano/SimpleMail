package server

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/carloslauriano/simpleEmail/config"
	"github.com/carloslauriano/simpleEmail/storage"
)

// POP3Server implementa o servidor POP3
type POP3Server struct {
	store storage.Storage
	cfg   *config.Config
}

// NewPOP3Server cria um novo servidor POP3
func NewPOP3Server(store storage.Storage, cfg *config.Config) *POP3Server {
	return &POP3Server{
		store: store,
		cfg:   cfg,
	}
}

// handleConnection gerencia uma conexão POP3
func (s *POP3Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Enviar saudação
	conn.Write([]byte("+OK SimpleEmail POP3 server ready\r\n"))

	// Ler comandos do cliente
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		cmd := string(buf[:n])
		// Processar comando...
		// Implementar comandos POP3 básicos: USER, PASS, LIST, RETR, DELE, QUIT
	}
}

// StartPOP3Server inicia o servidor POP3
func StartPOP3Server(cfg *config.Config, store storage.Storage) error {
	server := NewPOP3Server(store, cfg)
	addr := fmt.Sprintf("%s:%d", cfg.POP3.Address, cfg.POP3.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("falha ao iniciar servidor POP3: %w", err)
	}

	log.Printf("Iniciando servidor POP3 em %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Erro ao aceitar conexão: %v", err)
			continue
		}

		go server.handleConnection(conn)
	}
} 