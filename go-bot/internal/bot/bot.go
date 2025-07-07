package bot

import (
	"bufio"
	"fmt"
	"log"
	"os/exec" // <-- NOVO import
	"strconv" // <-- NOVO import
	"strings"
	"sync"
	"time" // <-- NOVO import

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
	// --- NOVO: Inicia o processo de encaminhamento de logs em uma goroutine ---
	go b.startLogForwarder()

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
				go b.handleCommand(update.Message)
			}
		}
	}
}

func (b *Bot) startLogForwarder() {
	// Espera um pouco para o bot se estabilizar e a conexão com a API ser feita
	time.Sleep(15 * time.Second)

	log.Println("   ⟫  VERIFICANDO CONFIGURAÇÃO DO TERMINAL DE LOGS...")

	terminalIDStr, err := b.Storage.GetSetting("terminal_channel_id")
	if err != nil {
		log.Printf("[ERRO FATAL] Não foi possível buscar o ID do terminal no DB: %v", err)
		return
	}
	if terminalIDStr == "" {
		log.Println("   ⟫  TERMINAL DE LOGS NÃO CONFIGURADO. Encaminhamento desativado.")
		return
	}
	terminalID, _ := strconv.ParseInt(terminalIDStr, 10, 64)

	// Mensagem de Diagnóstico para o Admin
	adminMessage := fmt.Sprintf("✅ TENTANDO INICIAR O TERMINAL DE LOGS PARA O CANAL <code>%d</code>...", terminalID)
	b.API.Send(tgbotapi.NewMessage(b.Config.AdminUserID, adminMessage))

	// Comando para seguir os logs do serviço systemd
	cmd := exec.Command("journalctl", "-u", "GloryLogs.service", "-f", "--no-pager")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errorMsg := fmt.Sprintf("❌ ERRO ao criar pipe para journalctl: %v", err)
		log.Println(errorMsg)
		b.API.Send(tgbotapi.NewMessage(terminalID, errorMsg))
		return
	}
	if err := cmd.Start(); err != nil {
		errorMsg := fmt.Sprintf("❌ ERRO ao iniciar journalctl (verifique as permissões do usuário do serviço): %v", err)
		log.Println(errorMsg)
		b.API.Send(tgbotapi.NewMessage(terminalID, errorMsg))
		return
	}

	log.Printf("   ⟫  SUCESSO! INICIANDO ENCAMINHAMENTO DE LOGS PARA O CANAL: %d", terminalID)
	b.API.Send(tgbotapi.NewMessage(terminalID, "✅ **Terminal de Logs Conectado!** Aguardando novas saídas do sistema..."))

	// Lê a saída do comando linha por linha
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// Evita enviar linhas vazias
		if strings.TrimSpace(line) != "" {
			msg := tgbotapi.NewMessage(terminalID, "<code>"+line+"</code>")
			msg.ParseMode = tgbotapi.ModeHTML
			b.API.Send(msg)
		}
	}

	// Se o loop terminar, significa que o comando falhou
	if err := scanner.Err(); err != nil {
		errorMsg := fmt.Sprintf("❌ ERRO ao ler a saída do journalctl: %v", err)
		log.Println(errorMsg)
		b.API.Send(tgbotapi.NewMessage(terminalID, errorMsg))
	}
	log.Println("   ⟫  ENCAMINHADOR DE LOGS PAROU.")
	b.API.Send(tgbotapi.NewMessage(terminalID, "❌ **Terminal de Logs Desconectado!** O processo foi interrompido."))
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
	defer func() {
		if r := recover(); r != nil {
			log.Printf("!!!!!! PÂNICO RECUPERADO em handleCommand: %v", r)
			errorMsg := ("Ocorreu um erro interno crítico ao processar seu comando. O administrador foi notificado.")
			b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, errorMsg))
		}
	}()

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
	case "cloud":
		b.handleCloudCommand(msg)
	case "release":
		b.handleReleaseCommand(msg)
	case "profile":
		b.handleProfileCommand(msg)
	case "plans":
		b.handlePlansCommand(msg)
	// --- NOVOS COMANDOS ---
	case "users":
		b.handleUsersCommand(msg)
	case "invites":
		b.handleInvitesCommand(msg)
	case "terminal":
		b.handleTerminalCommand(msg)
	default:
		// Comando não reconhecido
	}
}
