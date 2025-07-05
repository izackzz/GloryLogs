package bot

import (
	// "log"
	"strings"
	"sync"

	"github.com/fatih/color"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"glorylogs-bot/internal/config"
	"glorylogs-bot/internal/storage"
)

// SearchState armazena o estado de uma pesquisa ativa para um usuário.
type SearchState struct {
	Term                 string
	Results              []string
	Offset               int
	MessageID            int
	ChatID               int64
	ThreadID             int
	UserCommandMessageID int
}

// Bot é a struct principal que gerencia todas as dependências e o estado do bot.
type Bot struct {
	API              *tgbotapi.BotAPI
	Config           config.Config
	Storage          *storage.Storage
	UserSearchStates map[int64]*SearchState
	stateMutex       sync.RWMutex
}

// NewBot cria uma nova instância do Bot.
func NewBot(cfg config.Config, store *storage.Storage) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}
	// api.Debug = true // Descomente para ver logs detalhados da API

	color.Green("   ⟫  STARTED ONLINE @%s", api.Self.UserName)

	return &Bot{
		API:              api,
		Config:           cfg,
		Storage:          store,
		UserSearchStates: make(map[int64]*SearchState),
	}, nil
}

// Start inicia o bot e o loop de escuta de atualizações.
func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.API.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			go b.handleCallbackQuery(update.CallbackQuery)
			continue
		}

		if update.Message != nil {
			if update.Message.IsCommand() {
				// Executa o handler de comando em uma nova goroutine
				go b.handleCommand(update.Message)
			}
		}
	}
}

func (b *Bot) sendMediaBanner(chatID int64, replyToID int, bannerPath, caption string, keyboard *tgbotapi.InlineKeyboardMarkup) {
	lowerPath := strings.ToLower(bannerPath)
	var chattable tgbotapi.Chattable

	// Verifica se o caminho do arquivo termina com uma extensão de vídeo
	if strings.HasSuffix(lowerPath, ".mp4") || strings.HasSuffix(lowerPath, ".mov") {
		videoConfig := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(bannerPath))
		videoConfig.Caption = caption
		videoConfig.ParseMode = tgbotapi.ModeHTML
		if keyboard != nil {
			videoConfig.ReplyMarkup = keyboard
		}
		if replyToID != 0 {
			videoConfig.ReplyToMessageID = replyToID
		}
		chattable = videoConfig
	} else {
		// Caso contrário, assume que é uma foto
		photoConfig := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(bannerPath))
		photoConfig.Caption = caption
		photoConfig.ParseMode = tgbotapi.ModeHTML
		if keyboard != nil {
			photoConfig.ReplyMarkup = keyboard
		}
		if replyToID != 0 {
			photoConfig.ReplyToMessageID = replyToID
		}
		chattable = photoConfig
	}

	// Envia a mídia (foto ou vídeo) para o Telegram
	if _, err := b.API.Send(chattable); err != nil {
		// log.Printf("Erro ao enviar banner de mídia (%s): %v", bannerPath, err)
	}
}

// handleCommand é o roteador principal para todos os comandos recebidos.
func (b *Bot) handleCommand(msg *tgbotapi.Message) {

	// --- CÓDIGO NOVO: ADICIONE O RECUPERADOR DE PÂNICO AQUI ---
	defer func() {
		if r := recover(); r != nil {
			// Este bloco só será executado se ocorrer um pânico
			// log.Printf("!!!!!! PÂNICO RECUPERADO em handleCommand: %v", r)

			// Opcional: avisa o usuário que algo deu muito errado
			errorMsg := ("Ocorreu um erro interno crítico ao processar seu comando. O administrador foi notificado.")
			b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
		}
	}()
	// --- FIM DO CÓDIGO NOVO ---

	// Registra o chat em qualquer comando para manter a lista de chats atualizada
	b.registerChat(msg.Chat.ID)

	switch msg.Command() {
	case "start":
		b.handleStartCommand(msg)
	case "help":
		b.handleHelpCommand(msg)
	case "info":
		b.handleInfoCommand(msg)
	case "invite":
		b.handleInviteCommand(msg)
	case "add":
		b.handleAddUserCommand(msg)
	case "remove":
		b.handleRemoveUserCommand(msg)
	case "admin":
		b.handleAdminCommand(msg)
	case "search":
		b.handleSearchCommand(msg)
	case "all":
		b.handleBroadcastCommand(msg)
	case "profile":
		b.handleProfileCommand(msg)
	case "plans":
		b.handlePlansCommand(msg)
	default:
		// Comando não reconhecido
	}
}
