package main

import (
	"fmt"
	"log"

	"glorylogs-bot/internal/bot"
	"glorylogs-bot/internal/config"
	"glorylogs-bot/internal/storage"
)

func main() {
	cfg := config.LoadConfig()
	printStartupBanner(cfg.BotUsername)

	// Inicializa a camada de armazenamento de dados (agora com SQL)
	store, err := storage.NewStorage()
	if err != nil {
		log.Fatalf("Erro fatal ao inicializar o banco de dados: %v", err)
	}
	defer store.Close() // Garante que a conexão com o DB seja fechada ao sair

	// Cria uma nova instância do bot, injetando as dependências
	botInstance, err := bot.NewBot(cfg, store)
	if err != nil {
		log.Fatalf("Erro ao criar a instância do bot: %v", err)
	}

	botInstance.Start()
}

// printStartupBanner (função sem alterações)
func printStartupBanner(botUsername string) {
	colorGreen := "\033[92m"
	colorReset := "\033[0m"
	banner := colorGreen + `
==============================================
           ┬ ┬┌─┐┬  ┌─┐┌─┐┌┬┐┌─┐
           │││├┤ │  │  │ ││││├┤   🐦‍🔥
           └┴┘└─┘┴─┘└─┘┴ ┴└─┘┴ ┴└─┘
           @%s IS ONLINE...
==============================================
` + colorReset
	fmt.Printf(banner, botUsername)
}
