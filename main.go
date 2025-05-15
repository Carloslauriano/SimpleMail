package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/carloslauriano/simpleEmail/config"
	"github.com/carloslauriano/simpleEmail/server"
	"github.com/carloslauriano/simpleEmail/storage"
)

func main() {
	// Carregar configuração
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

	// Inicializar armazenamento
	store, err := storage.NewStorage(cfg)
	if err != nil {
		log.Fatalf("Erro ao inicializar armazenamento: %v", err)
	}
	defer store.Close()

	// Iniciar servidores em goroutines separadas
	errors := make(chan error, 3)

	go func() {
		if err := server.StartSMTPServer(cfg, store); err != nil {
			errors <- err
		}
	}()

	go func() {
		if err := server.StartIMAPServer(cfg, store); err != nil {
			errors <- err
		}
	}()

	go func() {
		if err := server.StartPOP3Server(cfg, store); err != nil {
			errors <- err
		}
	}()

	// Aguardar sinais de interrupção
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errors:
		log.Fatalf("Erro no servidor: %v", err)
	case sig := <-sigChan:
		log.Printf("Recebido sinal %v, encerrando...", sig)
	}
} 