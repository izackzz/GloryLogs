package main

import (
	"fmt"
	"log"

	"glorylogs-bot/internal/bot"
	"glorylogs-bot/internal/config"
	"glorylogs-bot/internal/storage"
)

func main() {
	// Carrega as configurações do arquivo .env
	cfg := config.LoadConfig()

	// Exibe o banner de inicialização, mantendo a formatação original
	printStartupBanner(cfg.BotUsername)

	// Inicializa a camada de armazenamento de dados (carrega users.csv, etc.)
	store := storage.NewStorage()

	// Cria uma nova instância do bot, injetando as dependências
	botInstance, err := bot.NewBot(cfg, store)
	if err != nil {
		log.Fatalf("Erro ao criar a instância do bot: %v", err)
	}

	// Inicia o bot. Esta função contém o loop principal que escuta as atualizações.
	botInstance.Start()
}

// printStartupBanner exibe o logo e a mensagem de status no console.
func printStartupBanner(botUsername string) {
	// Códigos de cor ANSI para o verde
	colorGreen := "\033[92m"
	colorReset := "\033[0m"

	banner := colorGreen + `
==============================================

           ┬ ┬┌─┐┬  ┌─┐┌─┐┌┬┐┌─┐
           │││├┤ │  │  │ ││││├┤   🐦‍🔥
           └┴┘└─┘┴─┘└─┘└─┘┴ ┴└─┘
           @%s IS ONLINE...

==============================================

` + colorReset

	fmt.Printf(banner, botUsername)

}
