package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config representa a configuração global do sistema
type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
	IMAP     IMAPConfig     `mapstructure:"imap"`
	POP3     POP3Config     `mapstructure:"pop3"`
}

// DatabaseConfig representa a configuração do banco de dados
type DatabaseConfig struct {
	Type     string `mapstructure:"type"` // "sqlite" ou "postgres"
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	Path     string `mapstructure:"path"` // Para SQLite
}

// SMTPConfig representa a configuração do servidor SMTP
type SMTPConfig struct {
	Address      string `mapstructure:"address"`
	Port         int    `mapstructure:"port"`
	Domain       string `mapstructure:"domain"`
	AllowInsecure bool  `mapstructure:"allow_insecure"`
	MaxMessageBytes int `mapstructure:"max_message_bytes"`
}

// IMAPConfig representa a configuração do servidor IMAP
type IMAPConfig struct {
	Address string `mapstructure:"address"`
	Port    int    `mapstructure:"port"`
}

// POP3Config representa a configuração do servidor POP3
type POP3Config struct {
	Address string `mapstructure:"address"`
	Port    int    `mapstructure:"port"`
}

var cfg *Config

// LoadConfig carrega configurações do arquivo config.yaml
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		// Usar diretório atual se nenhum caminho for fornecido
		dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(dir, "config.yaml")
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo de configuração: %w", err)
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("erro ao processar configuração: %w", err)
	}

	return cfg, nil
}

// GetConfig retorna a configuração atual
func GetConfig() *Config {
	return cfg
} 