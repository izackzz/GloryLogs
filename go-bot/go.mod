module glorylogs-bot

go 1.18

require (
	github.com/fatih/color v1.18.0
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/joho/godotenv v1.5.1
)

require (
	// indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.25.0 // indirect
)

// GloryLogs/
// ├── go-bot/
// │   ├── cmd/
// │   │   └── glorylogs-bot/
// │   │       └── main.go          # Ponto de entrada da aplicação
// │   ├── internal/
// │   │   ├── bot/
// │   │   │   ├── handlers.go      # Lógica para os comandos (/start) e callbacks
// │   │   │   └── bot.go           # Configuração e inicialização do bot
// │   │   ├── config/
// │   │   │   └── config.go        # Carregamento das configurações (.env)
// │   │   ├── storage/
// │   │   │   └── storage.go       # Funções para gerenciar os arquivos (users.csv, etc.)
// │   │   └── search/
// │   │       └── engine.go        # Lógica de busca nos arquivos de log
// │   ├── db/                      # (será criado automaticamente pelo bot)
// │   ├── logs/                    # (coloque seus arquivos .txt aqui)
// │   ├── bg/
// │   │   ├── bg.png
// │   │   └── mkt.jpg
// │   ├── go.mod                   # Arquivo de dependências do Go (substituto do requirements.txt)
// │   └── .env                     # Suas variáveis de ambiente
