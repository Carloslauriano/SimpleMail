# Simple Email

Um servidor de email simples em Go que suporta SMTP, IMAP e POP3, com opções de armazenamento em SQLite ou PostgreSQL.

## Características

- Servidor SMTP para envio de emails
- Servidor IMAP para acesso a emails
- Servidor POP3 para acesso a emails
- Suporte a armazenamento em SQLite ou PostgreSQL
- Configuração flexível via arquivo YAML
- Suporte a múltiplos usuários e caixas de correio
- Suporte a anexos
- Suporte a flags de mensagem (lida, excluída, rascunho)

## Requisitos

- Go 1.21 ou superior
- SQLite3 (para armazenamento SQLite)
- PostgreSQL (para armazenamento PostgreSQL)

## Instalação

1. Clone o repositório:
```bash
git clone https://github.com/carloslauriano/simpleEmail.git
cd simpleEmail
```

2. Instale as dependências:
```bash
go mod download
```

3. Compile o projeto:
```bash
go build
```

## Configuração

O servidor é configurado através do arquivo `config.yaml`. Um exemplo de configuração é fornecido:

```yaml
database:
  type: "sqlite"  # ou "postgres"
  host: "localhost"
  port: 5432
  user: "simplemail"
  password: "simplemail"
  dbname: "simplemail"
  path: "./data/simplemail.db"

smtp:
  address: "0.0.0.0"
  port: 25
  domain: "localhost"
  allow_insecure: false
  max_message_bytes: 10485760

imap:
  address: "0.0.0.0"
  port: 143

pop3:
  address: "0.0.0.0"
  port: 110
```

## Uso

1. Configure o arquivo `config.yaml` de acordo com suas necessidades.

2. Execute o servidor:
```bash
./simpleEmail
```

3. Configure seu cliente de email para usar os servidores:
   - SMTP: `localhost:25`
   - IMAP: `localhost:143`
   - POP3: `localhost:110`

## Estrutura do Projeto

```
.
├── config/
│   ├── config.go
│   └── config.yaml
├── server/
│   ├── smtp.go
│   ├── imap.go
│   └── pop3.go
├── storage/
│   ├── models.go
│   ├── storage.go
│   ├── sqlite.go
│   └── postgres.go
├── main.go
├── go.mod
└── README.md
```

## Contribuindo

Contribuições são bem-vindas! Por favor, sinta-se à vontade para enviar um Pull Request.

## Licença

Este projeto está licenciado sob a licença MIT - veja o arquivo LICENSE para mais detalhes. 




<!-- beleza quero que meu sistema "Simple Email" seja um servidor de smtp, imap e pop3 lavando os email ou sqlite ou postegra dependendo da configuração feito em go -->