package bot

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"glorylogs-bot/internal/config"
	"glorylogs-bot/internal/search"
	"glorylogs-bot/internal/storage"
)

// handleCallbackQuery processa todos os cliques em botões inline.
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := b.API.Request(callback); err != nil {
		color.Red("   ⟫  ERRO AO RESPONDER AO CALLBACK: %v", err)
	}

	userID := query.From.ID
	data := query.Data

	// Handlers que não dependem de estado ou autorização
	switch data {
	case "show_plans":
		b.handleShowPlans(query)
		return
	case "delete_broadcast":
		b.handleDeleteBroadcast(query)
		return
	}

	// Handlers que precisam de autorização e estado de busca
	if !b.isUserAuthorized(userID, query.Message.Chat.ID) {
		return
	}

	b.stateMutex.Lock()
	defer b.stateMutex.Unlock()

	state, ok := b.UserSearchStates[userID]
	if !ok {
		// Se o estado não existe, não podemos prosseguir com ações de busca
		return
	}

	switch data {
	case "delete_search":
		b.handleDeleteSearch(query, state)
	case "download":
		b.handleDownload(query, state)
	case "next", "prev":
		b.handlePagination(state, data)
	default:
		color.Red("   ⟫  CALLBACK DE BUSCA NÃO RECONHECIDO: %s", data)
	}
}

// handleShowPlans lida com a exibição da mensagem de planos.
func (b *Bot) handleShowPlans(query *tgbotapi.CallbackQuery) {
	// Obtém o texto e o teclado dos planos, como antes.
	messageText, replyMarkup := b.getPlansMessageAndKeyboard()

	// A tentativa de apagar a mensagem anterior foi removida para garantir
	// que a função funcione em qualquer chat, mesmo sem permissão de admin.

	// Envia uma nova mensagem com o banner e as informações dos planos.
	b.sendMediaBanner(query.Message.Chat.ID, 0, b.Config.PlansBanner, messageText, &replyMarkup)

	// Responde ao callback para que o botão no cliente do usuário pare de "carregar".
	callback := tgbotapi.NewCallback(query.ID, "Confira nossos planos!")
	b.API.Request(callback)
}

// handlePagination atualiza o offset e reenvia a página de resultados.
// Esta função agora recebe o estado diretamente para evitar deadlocks.
func (b *Bot) handlePagination(state *SearchState, data string) {
	switch data {
	case "next":
		state.Offset += 30
	case "prev":
		state.Offset -= 30
	}
	// Chama a nova versão da função de envio que não causa deadlock
	b.sendPremiumResultsPage(state)
}

// isUserAuthorized verifica se um usuário tem permissão (premium ou admin).
func (b *Bot) isUserAuthorized(userID int64, chatID int64) bool {
	if userID == b.Config.AdminUserID {
		return true
	}

	// Tenta verificar o usuário primeiro
	user, err := b.Storage.GetUser(userID)
	if err == nil && user != nil {
		if user.Premium == "y" {
			endDate, err := time.Parse("2006-01-02", user.EndDate)
			if err == nil && !endDate.Before(time.Now().Truncate(24*time.Hour)) {
				return true
			}
		}
	}

	// Se for um grupo, tenta verificar o chat
	chat, err := b.Storage.GetUser(chatID)
	if err == nil && chat != nil {
		if chat.Premium == "y" {
			endDate, err := time.Parse("2006-01-02", chat.EndDate)
			if err == nil && !endDate.Before(time.Now().Truncate(24*time.Hour)) {
				return true
			}
		}
	}

	return false
}

// getPlansMessageAndKeyboard retorna o texto e os botões da mensagem de planos.
func (b *Bot) getPlansMessageAndKeyboard() (string, tgbotapi.InlineKeyboardMarkup) {
	// Pega a mensagem da configuração
	messageText := b.Config.PlansMessage

	// Se a variável de ambiente não estiver definida, usa um texto padrão como fallback
	if messageText == "" {
		log.Println("Aviso: PLANS_MESSAGE_ENV não definida no .env, usando texto padrão.")
		messageText = `
		<blockquote>✨ CONHEÇA O PREMIUM ✨</blockquote>
		<blockquote>⁝⁝⁝ Para adquirir ou tirar dúvidas, fale com um administrador.</blockquote>
		`
	} else {
		// Substitui os caracteres '\n' por quebras de linha reais
		messageText = strings.ReplaceAll(messageText, "\\n", "\n")
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🛍️ ꜰᴀʟᴀʀ ᴄᴏᴍ ꜱᴜᴘᴏʀᴛᴇ", "https://t.me/"+b.Config.AdminMention),
		),
	)
	return messageText, keyboard
}

// handleInviteCommand lida com o comando /invite.
func (b *Bot) handleInviteCommand(msg *tgbotapi.Message) {
	if msg.From.ID != b.Config.AdminUserID {
		return
	}
	args := strings.Fields(msg.CommandArguments())
	if len(args) != 2 || !strings.HasPrefix(args[1], "max:") {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Uso incorreto. Formato: <code>/invite &lt;dias&gt; max:&lt;limite&gt;</code>\nExemplo: <code>/invite 30 max:10</code>")
		reply.ParseMode = tgbotapi.ModeHTML
		b.API.Send(reply)
		return
	}
	days, err1 := strconv.Atoi(args[0])
	limitStr := strings.TrimPrefix(args[1], "max:")
	limit, err2 := strconv.Atoi(limitStr)
	if err1 != nil || err2 != nil {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Os valores de dias e limite devem ser números inteiros."))
		return
	}
	code := generateInviteCode(8)
	newInvite := &storage.Invite{
		Code:  code,
		Days:  days,
		Limit: limit,
		Used:  0,
	}

	// Salva o novo convite no banco de dados
	if err := b.Storage.AddInvite(newInvite); err != nil {
		log.Printf("Erro ao salvar convite no DB: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ocorreu um erro interno ao criar o convite."))
		return
	}

	link := fmt.Sprintf("https://t.me/%s?start=%s", b.Config.BotUsername, code)
	mensagem := fmt.Sprintf("✅ Link de convite gerado com sucesso!\n\n<b>Link:</b> <code>%s</code>\n<b>Duração:</b> %d dias\n<b>Limite de usos:</b> %d", link, days, limit)
	reply := tgbotapi.NewMessage(msg.Chat.ID, mensagem)
	reply.ParseMode = tgbotapi.ModeHTML
	b.API.Send(reply)
}

// handleInvitation lida com um usuário que entra com um código de convite.
func (b *Bot) handleInvitation(msg *tgbotapi.Message, code string) {
	userID := msg.From.ID
	userName := msg.From.FirstName

	if b.isUserPremium(userID) {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Você já possui uma assinatura premium ativa!"))
		return
	}

	// Busca o convite no banco de dados
	invite, err := b.Storage.GetInvite(code)
	if err != nil {
		log.Printf("Erro ao buscar convite no DB: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ocorreu um erro interno ao verificar o convite."))
		return
	}
	if invite == nil {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Código de convite inválido ou expirado."))
		return
	}

	if invite.Used >= invite.Limit {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Este link de convite já atingiu o limite máximo de usos."))
		return
	}

	// Ativa o premium para o usuário
	now := time.Now().UTC()
	endDate := now.AddDate(0, 0, invite.Days)
	newUser := &storage.User{
		ID:               userID,
		RegistrationDate: now.Format("2006-01-02"),
		EndDate:          endDate.Format("2006-01-02"),
		Premium:          "y",
		DailyLimit:       10,
		SearchesToday:    0,
		LastSearchDate:   "",
	}

	if err := b.Storage.AddOrUpdateUser(newUser); err != nil {
		log.Printf("Erro ao adicionar usuário via convite no DB: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Erro ao ativar sua assinatura."))
		return
	}

	// Incrementa o uso do convite
	if err := b.Storage.IncrementInviteUsage(code); err != nil {
		log.Printf("Erro ao incrementar uso do convite no DB: %v", err)
		// Continua mesmo com erro, pois o usuário já foi adicionado
	}

	color.Green("   ⟫  USER %d JOIN WITH INVITE CODE %s", userID, code)

	mensagem := fmt.Sprintf("<blockquote>🎉 Parabéns, %s!!!</blockquote>\nSua assinatura premium foi ativada com sucesso por <b>%d dias</b>.\n\nPara começar, use o comando <b>/search</b>. Para ver todos os comandos, digite <b>/help</b>.", userName, invite.Days)
	reply := tgbotapi.NewMessage(msg.Chat.ID, mensagem)
	reply.ParseMode = tgbotapi.ModeHTML
	b.API.Send(reply)
}

// isUserPremium é uma verificação específica para saber se o premium está ativo.
func (b *Bot) isUserPremium(userID int64) bool {
	user, err := b.Storage.GetUser(userID)
	if err != nil || user == nil {
		return false
	}

	if user.Premium != "y" {
		return false
	}

	endDate, err := time.Parse("2006-01-02", user.EndDate)
	if err != nil {
		return false
	}
	return !time.Now().UTC().After(endDate)
}

// generateInviteCode gera uma string alfanumérica aleatória.
func generateInviteCode(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	result := make([]byte, length)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "fallback"
		}
		result[i] = letters[num.Int64()]
	}
	return string(result)
}

// handleStartCommand lida com o comando /start.
func (b *Bot) handleStartCommand(msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	if args != "" {
		b.handleInvitation(msg, args)
		return
	}

	color.Green("   ⟫  USER %d STARTED A BOT", msg.From.ID)
	userName := msg.From.FirstName

	// A mensagem de boas-vindas está correta.
	mensagem := fmt.Sprintf(
		"Olá %s, seja bem-vindo!\n\n"+
			"<blockquote>Sou o Bot de consultas 𝐆𝐋𝐎𝐑𝐘 𝐋𝐎𝐆𝐒 👁‍🗨!</blockquote>\n"+
			"<i><b>by</b> @%s</i>\n\n"+
			"🔍 Para realizar uma consulta, utilize o comando:\n"+
			"<b>/search &lt;sua_busca&gt;</b>\n\n"+
			"ℹ️ Utilize os operadores de busca avançada para refinar seus resultados:\n\n"+
			"<code>inurl:</code> Busca na URL\n"+
			"<code>intext:</code> Busca no usuário e senha\n"+
			"<code>site:</code> Busca pelo domínio\n"+
			"<code>filetype:</code> Busca por extensão de arquivo\n\n"+
			"📌 Exemplo: <code>/search intext:facebook inurl:login site:example.com</code>\n\n"+
			"<blockquote>➡️ Use as setas de navegação para ver mais resultados.</blockquote>\n\n"+
			"❓ Para ver todos os comandos disponíveis, digite /help\n\n",
		userName, b.Config.AdminMention,
	)

	// --- A CORREÇÃO ESTÁ AQUI ---
	// Corrigimos o tipo de botão para "ᴘʟᴀɴᴏꜱ"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("ꜱᴜᴘᴏʀᴛᴇ", "https://t.me/"+b.Config.AdminMention), // Este está correto (é uma URL)
			tgbotapi.NewInlineKeyboardButtonData("ᴘʟᴀɴᴏꜱ", "show_plans"),                          // CORRIGIDO: Agora usa NewInlineKeyboardButtonData
		),
	)
	// --- FIM DA CORREÇÃO ---

	// A chamada para enviar o banner continua a mesma.
	b.sendMediaBanner(msg.Chat.ID, msg.MessageID, b.Config.StartBanner, mensagem, &keyboard)
}

// handleHelpCommand lida com o comando /help.
func (b *Bot) handleHelpCommand(msg *tgbotapi.Message) {
	mensagem := `<blockquote>🎯 <b>Comandos Disponíveis:</b></blockquote>

🔎 <b>/search</b> &lt;sua_busca&gt; - Realiza uma pesquisa nos logs.
ℹ️ <b>/info</b> - Exibe informações sobre a base de dados.
🗣️ <b>/profile</b> - Informações sobre seu plano e assinatura.

📄 <b>Operadores de Busca Avançada:</b>
⁝⁝⁝ <code>inurl:</code> Busca na URL
⁝⁝⁝ <code>intext:</code> Busca no usuário e senha
⁝⁝⁝ <code>site:</code> Busca pelo domínio
⁝⁝⁝ <code>filetype:</code> Busca por extensão de arquivo

📝 <b>Exemplo:</b> <code>/search intext:"admin" site:example.com</code>

<blockquote>➡️ Use as setas de navegação para ver mais resultados durante a pesquisa.</blockquote>

<blockquote>⏬🗂️ Faça download do resultado completo da busca.</blockquote>`
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("ꜱᴜᴘᴏʀᴛᴇ", "https://t.me/"+b.Config.AdminMention),
			tgbotapi.NewInlineKeyboardButtonData("ᴘʟᴀɴᴏꜱ", "show_plans"),
		),
	)
	b.sendMediaBanner(msg.Chat.ID, msg.MessageID, b.Config.StartBanner, mensagem, &keyboard)
}

// handleInfoCommand lida com o comando /info.
func (b *Bot) handleInfoCommand(msg *tgbotapi.Message) {
	if !b.isUserAuthorized(msg.From.ID, msg.Chat.ID) {
		reply := fmt.Sprintf("<blockquote>❌ VOCÊ NÃO TEM PERMISSÃO SUFICIENTE PARA USAR ESSE BOT, ENTRE EM CONTATO COM @%s</blockquote>", b.Config.AdminMention)
		msgConfig := tgbotapi.NewMessage(msg.Chat.ID, reply)
		msgConfig.ReplyToMessageID = msg.MessageID
		msgConfig.ParseMode = tgbotapi.ModeHTML
		b.API.Send(msgConfig)
		return
	}

	// Coleta as estatísticas do banco de dados do bot
	dbStats, err := b.calculateDBStats()
	if err != nil {
		color.Red("Erro ao calcular estatísticas do DB: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ocorreu um erro ao buscar as estatísticas do bot."))
		return
	}

	// Coleta as estatísticas do sistema do servidor
	sysStats, err := getSystemStats()
	if err != nil {
		color.Red("Erro ao calcular estatísticas do sistema: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ocorreu um erro ao buscar as estatísticas do servidor."))
		return
	}

	// Monta a mensagem final combinando TODAS as informações
	mensagem := fmt.Sprintf(
		`
<blockquote>[ =============== ]    𝐁𝐎𝐓 𝐒𝐓𝐀𝐓𝐔𝐒    [ =============== ]</blockquote>
🗂️ ⁝⁝⁝ TOTAL DE ARQUIVOS: <b>%d</b>
📄 ⁝⁝⁝ TOTAL DE LINHAS: <b>%d</b>
✅ ⁝⁝⁝ ENTRADAS VÁLIDAS: <b>%d</b>
💾 ⁝⁝⁝ TAMANHO DA BASE: <b>%s</b>
📥 ⁝⁝⁝ ÚLTIMO ARQUIVO: <b>%s</b>
🕒 ⁝⁝⁝ DATA DE ENTRADA: <b>%s</b>
👥 ⁝⁝⁝ USUÁRIOS ATIVOS: <b>%d</b>

<blockquote>[ =============== ]      𝐒𝐄𝐑𝐕𝐄𝐑      [ =============== ]</blockquote>
🖥️ ⁝⁝⁝ SISTEMA: <b>%s</b>
🕒 ⁝⁝⁝ UPTIME: <b>~%s</b>
🔥 ⁝⁝⁝ CPU: <b>%s (%s cores)</b>
⚡ ⁝⁝⁝ USO DE CPU: <b>%s</b>
🧠 ⁝⁝⁝ RAM TOTAL: <b>%s</b>
💾 ⁝⁝⁝ USO DE RAM: <b>%s</b>
💽 ⁝⁝⁝ DISCO TOTAL: <b>%s</b>
📀 ⁝⁝⁝ USO DE DISCO: <b>%s</b>`,
		// Estatísticas detalhadas do Bot (re-adicionadas)
		dbStats["totalFiles"],
		dbStats["totalLines"],
		dbStats["validEntries"],
		b.formatSize(dbStats["totalSize"].(int64)), // Usa a função para formatar o tamanho
		dbStats["lastFileName"],
		dbStats["lastFileDate"],
		dbStats["activeUsers"],
		// Estatísticas do Servidor
		sysStats["os"],
		sysStats["uptime"],
		sysStats["cpuModel"],
		sysStats["cpuCores"],
		sysStats["cpuUsage"],
		sysStats["memTotal"],
		sysStats["memUsage"],
		sysStats["diskTotal"],
		sysStats["diskUsage"],
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, mensagem)
	reply.ParseMode = tgbotapi.ModeHTML
	reply.ReplyToMessageID = msg.MessageID
	b.API.Send(reply)
}

func (b *Bot) registerChat(chatID int64) {
	err := b.Storage.AddChat(chatID)

	if err != nil {
		log.Printf("Erro ao registrar chat %d no DB: %v", chatID, err)
	}
}

func (b *Bot) calculateDBStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Contar usuários ativos do banco de dados
	query := `SELECT COUNT(*) FROM users WHERE premium = 'y' AND end_date >= ?`
	today := time.Now().UTC().Format("2006-01-02")

	var activeUsers int
	err := b.Storage.DB.QueryRow(query, today).Scan(&activeUsers)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar usuários ativos no DB: %w", err)
	}
	stats["activeUsers"] = activeUsers

	// Lógica de contagem de arquivos de log (permanece a mesma)
	var totalFiles, totalLines, validEntries int
	var totalSize int64
	var lastFileModTime time.Time
	var lastFileName string

	err = filepath.Walk(config.LogsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".txt") {
			totalFiles++
			totalSize += info.Size()
			if info.ModTime().After(lastFileModTime) {
				lastFileModTime = info.ModTime()
				lastFileName = info.Name()
			}
			file, _ := os.Open(path)
			if file != nil {
				defer file.Close()
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					totalLines++
					if strings.Count(scanner.Text(), ":") >= 2 {
						validEntries++
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("erro ao percorrer pasta de logs: %w", err)
	}

	stats["totalFiles"] = totalFiles
	stats["totalLines"] = totalLines
	stats["validEntries"] = validEntries
	stats["totalSize"] = totalSize
	stats["lastFileName"] = "N/A"
	if lastFileName != "" {
		stats["lastFileName"] = lastFileName
	}
	stats["lastFileDate"] = "N/A"
	if !lastFileModTime.IsZero() {
		stats["lastFileDate"] = lastFileModTime.Format("02/01/2006 15:04:05")
	}
	return stats, nil
}

func (b *Bot) formatSize(sizeBytes int64) string {
	if sizeBytes == 0 {
		return "0B"
	}
	sizeNames := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	fSize := float64(sizeBytes)
	for fSize >= 1024 && i < len(sizeNames)-1 {
		fSize /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", fSize, sizeNames[i])
}

// Handlers de Administração

func (b *Bot) handleAddUserCommand(msg *tgbotapi.Message) {
	if msg.From.ID != b.Config.AdminUserID {
		return
	}
	args := strings.Fields(msg.CommandArguments())
	dailyLimit := 99999
	var days int
	var targetID int64
	var err error
	var tempArgs []string
	for _, arg := range args {
		if strings.HasPrefix(strings.ToLower(arg), "limit:") {
			limitStr := strings.TrimPrefix(strings.ToLower(arg), "limit:")
			dailyLimit, err = strconv.Atoi(limitStr)
			if err != nil {
				b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Formato de limite inválido. Use `limit:100`."))
				return
			}
		} else {
			tempArgs = append(tempArgs, arg)
		}
	}
	args = tempArgs
	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		if len(args) != 1 {
			b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Uso no grupo: /add <dias> [limit:<limite>]"))
			return
		}
		days, err = strconv.Atoi(args[0])
		if err != nil {
			b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ O número de dias deve ser um valor numérico."))
			return
		}
		targetID = msg.Chat.ID
	} else {
		if len(args) != 2 {
			b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Uso no privado: /add <@user|id> <dias> [limit:<limite>]"))
			return
		}
		days, err = strconv.Atoi(args[1])
		if err != nil {
			b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ O número de dias deve ser um valor numérico."))
			return
		}
		targetID, err = b.resolveUserID(args[0])
		if err != nil {
			b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Usuário `%s` não encontrado.", args[0])))
			return
		}
	}
	now := time.Now().UTC()
	endDate := now.AddDate(0, 0, days)
	newUser := &storage.User{
		ID:               targetID,
		RegistrationDate: now.Format("2006-01-02"),
		EndDate:          endDate.Format("2006-01-02"),
		Premium:          "y",
		DailyLimit:       dailyLimit,
		SearchesToday:    0,
		LastSearchDate:   "",
	}

	// A chamada de salvamento agora é uma única função que faz INSERT ou UPDATE
	err = b.Storage.AddOrUpdateUser(newUser)
	if err != nil {
		log.Printf("Erro ao adicionar usuário ao banco de dados: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ocorreu um erro ao salvar o usuário no banco de dados."))
		return
	}

	color.Green("   ⟫  USER/GROUP %d ADDED TO PREMIUM", targetID)
	escopo := "usuário"
	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		escopo = "grupo"
	}
	limitStr := "Ilimitado"
	if dailyLimit < 99999 {
		limitStr = strconv.Itoa(dailyLimit)
	}
	replyText := fmt.Sprintf("✅ Premium ativo para %s `%d` até %s.\nLimite de buscas diárias: %s.",
		escopo, targetID, endDate.Format("02/01/2006"), limitStr)
	reply := tgbotapi.NewMessage(msg.Chat.ID, replyText)
	reply.ParseMode = tgbotapi.ModeMarkdown
	b.API.Send(reply)
}

func (b *Bot) handleRemoveUserCommand(msg *tgbotapi.Message) {
	if msg.From.ID != b.Config.AdminUserID {
		return
	}
	var targetID int64
	var chatType string
	if msg.ReplyToMessage != nil {
		targetID = msg.ReplyToMessage.From.ID
		chatType = "user"
	} else {
		args := msg.CommandArguments()
		if args == "" {
			b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "Uso: /remove <user_id|@username> ou responda a uma mensagem com /remove"))
			return
		}
		id, err := strconv.ParseInt(args, 10, 64)
		if err != nil {
			chat, errGet := b.API.GetChat(tgbotapi.ChatInfoConfig{
				ChatConfig: tgbotapi.ChatConfig{SuperGroupUsername: strings.TrimPrefix(args, "@")},
			})
			if errGet != nil {
				b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Não encontrei o usuário ou grupo `%s`.", args)))
				return
			}
			targetID = chat.ID
			chatType = chat.Type
		} else {
			targetID = id
		}
	}

	// Verifica se o usuário existe antes de tentar remover
	user, err := b.Storage.GetUser(targetID)
	if err != nil {
		log.Printf("Erro ao verificar usuário para remoção: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Erro ao verificar ID `%d`.", targetID)))
		return
	}
	if user == nil {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("ℹ️ ID `%d` não possui uma assinatura ativa.", targetID)))
		return
	}

	// Remove do banco de dados
	if err := b.Storage.RemoveUser(targetID); err != nil {
		log.Printf("Erro ao remover usuário do DB: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Erro ao remover ID `%d` da base de dados.", targetID)))
		return
	}

	color.Red("   ⟫  REMOVING %d", targetID)
	escopo := "Usuário"
	if chatType == "group" || chatType == "supergroup" {
		escopo = "Grupo"
	}
	reply := fmt.Sprintf("✅ %s `%d` removido da base de assinantes.", escopo, targetID)
	b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, reply))
}

func (b *Bot) handleAdminCommand(msg *tgbotapi.Message) {
	// A verificação do ID do administrador permanece, é uma boa prática de segurança.
	// Certifique-se que o ADMIN_USER_ID_ENV no seu arquivo .env está correto.
	if msg.From.ID != b.Config.AdminUserID {
		return
	}

	// Texto de ajuda atualizado para incluir os novos comandos.
	adminHelpText := `<blockquote>👨‍💻 Painel de Controle do Administrador</blockquote>

Bem-vindo ao seu painel de controle. Aqui está um guia rápido sobre como gerenciar o bot:

<b>⁝⁝⁝ 👤 Gerenciamento de Acessos</b>

<code>/add &lt;ID|@user&gt; &lt;dias&gt; [limit:N]</code>
↳ Adiciona um <b>usuário</b> premium. O limite de buscas diárias é opcional (padrão: ilimitado).
<i>Ex: /add 123456 30 limit:100</i>

<code>/add &lt;dias&gt; [limit:N]</code> (usado dentro de um grupo)
↳ Adiciona o <b>grupo</b> inteiro como premium. O limite também é opcional.

<code>/remove &lt;ID|@user&gt;</code>
↳ Remove o acesso premium de um usuário ou grupo. Você também pode responder a uma mensagem do usuário com /remove.

<b>⁝⁝⁝ 🎟️ Sistema de Convites</b>

<code>/invite &lt;dias&gt; max:&lt;usos&gt;</code>
↳ Gera um link de convite único.
<i>Ex: /invite 7 max:20</i>

<b>⁝⁝⁝ ☁️ Gerenciamento de Logs</b>

<code>/cloud</code> (respondendo a um arquivo .txt)
↳ Adiciona o arquivo de log respondido diretamente à sua base de dados.

<code>/release &lt;termo&gt;</code>
↳ Libera espaço removendo <b>linhas</b> que correspondem a uma busca.
<i>Ex: /release inurl:exemplo.com</i>

<code>/release &lt;tamanho&gt;mb</code>
↳ Libera espaço removendo <b>arquivos</b> inteiros, começando pelos mais antigos.
<i>Ex: /release 500mb</i>

<b>⁝⁝⁝ 📢 Comunicação em Massa</b>

<code>/all</code> (respondendo a uma mensagem)
↳ Envia a mensagem respondida para <b>todos os chats</b> onde o bot já foi iniciado.
Use a sintaxe de botões se desejar:
<code>/all
[Google](https://google.com) | [Bing](https://bing.com)
[Yandex](https://yandex.com)</code>

<b>⁝⁝⁝ 🔍 Comandos Gerais (Visão do Usuário)</b>

• <b>/search &lt;termo&gt;</b>
• <b>/profile</b>
• <b>/info</b>
• <b>/help</b>

<i>Dica: Usar o ID numérico do usuário/grupo é sempre mais confiável do que o @username.</i>`

	reply := tgbotapi.NewMessage(msg.Chat.ID, adminHelpText)
	reply.ParseMode = tgbotapi.ModeHTML
	reply.ReplyToMessageID = msg.MessageID // Boa prática: responder ao comando do admin.

	if _, err := b.API.Send(reply); err != nil {
		// Adicionar um log de erro pode ajudar a diagnosticar problemas futuros.
		color.Red("   ⟫  FALHA AO ENVIAR A MENSAGEM DE ADMIN: %v", err)
	}
}

func (b *Bot) handleBroadcastCommand(msg *tgbotapi.Message) {
	if msg.From.ID != b.Config.AdminUserID {
		return
	}
	if msg.ReplyToMessage == nil {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❗️ Use este comando respondendo à mensagem que deseja enviar."))
		return
	}

	replyMarkup := b.parseAllButtons(msg.Text)
	originalMsg := msg.ReplyToMessage

	log.Println("   ⟫  SENT MARKETING SHOT 💸")

	// Busca todos os IDs de chat do banco de dados
	chatIDs, err := b.Storage.GetAllChatIDs()
	if err != nil {
		log.Printf("Erro ao buscar chats para broadcast: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Erro ao buscar a lista de chats no banco de dados."))
		return
	}

	sentCount := 0
	for _, chatID := range chatIDs {
		copyMsg := tgbotapi.NewCopyMessage(chatID, originalMsg.Chat.ID, originalMsg.MessageID)
		if replyMarkup != nil {
			copyMsg.ReplyMarkup = replyMarkup
		}
		if _, err := b.API.Send(copyMsg); err == nil {
			sentCount++
		} else {
			// log.Printf("Erro ao enviar broadcast para o chat %d: %v", chatID, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("📤 Mensagem enviada para %d chats.", sentCount)))
}

func (b *Bot) handleProfileCommand(msg *tgbotapi.Message) {
	userID := msg.From.ID
	userName := msg.From.FirstName

	// Usa a função de verificação que já busca/cria o usuário e reseta o limite diário
	_, _, user := b.checkAndResetSearchLimit(userID)

	var premiumStatus, expirationStr, limitStr, resultsLimitStr, footerMsg string
	var keyboard tgbotapi.InlineKeyboardMarkup
	var searchesToday, daysLeft int
	dailyLimit := 3 // Padrão para novos usuários

	if user != nil && b.isUserPremium(userID) {
		endDate, _ := time.Parse("2006-01-02", user.EndDate)
		daysLeft = int(time.Until(endDate).Hours() / 24)
		if daysLeft < 0 {
			daysLeft = 0
		}
		premiumStatus = "Premium ✨"
		expirationStr = endDate.Format("02/01/2006")
		searchesToday = user.SearchesToday
		dailyLimit = user.DailyLimit
		limitStr = "Ilimitado"
		if dailyLimit < 99999 {
			limitStr = strconv.Itoa(dailyLimit)
		}
		resultsLimitStr = "Todos os resultados"
		footerMsg = "<blockquote>Use /help para saber como usar o bot</blockquote>"
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("🛍️ ꜰᴀʟᴀʀ ᴄᴏᴍ ꜱᴜᴘᴏʀᴛᴇ", "https://t.me/"+b.Config.AdminMention),
			),
		)
	} else {
		premiumStatus = "Gratuito 🆓"
		if user != nil {
			searchesToday = user.SearchesToday
			dailyLimit = user.DailyLimit
		}
		limitStr = strconv.Itoa(dailyLimit)
		resultsLimitStr = "15 por busca"
		footerMsg = "<blockquote>Faça upgrade para buscas e resultados ilimitados!</blockquote>"
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🦅 ᴄᴏɴʜᴇᴄᴇʀ ᴘʟᴀɴᴏꜱ ᴘʀᴇᴍɪᴜᴍ", "show_plans"),
			),
		)
	}

	var messageBuilder strings.Builder
	messageBuilder.WriteString(fmt.Sprintf("<blockquote>Olá %s, essa é sua conta:</blockquote>\n\n", userName))
	messageBuilder.WriteString(fmt.Sprintf("⁝⁝⁝ ID: <code>%d</code>\n", userID))
	messageBuilder.WriteString(fmt.Sprintf("⁝⁝⁝ Plano: <b>%s</b>\n", premiumStatus))
	if user != nil && b.isUserPremium(userID) {
		messageBuilder.WriteString(fmt.Sprintf("⁝⁝⁝ Dias restantes: %d dias\n", daysLeft))
		messageBuilder.WriteString(fmt.Sprintf("⁝⁝⁝ Data de expiração: %s\n\n", expirationStr))
	} else {
		messageBuilder.WriteString("\n")
	}
	messageBuilder.WriteString(fmt.Sprintf("⁝⁝⁝ Buscas hoje: %d / %s\n", searchesToday, limitStr))
	messageBuilder.WriteString(fmt.Sprintf("⁝⁝⁝ Resultados por busca: %s\n\n", resultsLimitStr))
	messageBuilder.WriteString(footerMsg)

	b.sendMediaBanner(msg.Chat.ID, msg.MessageID, b.Config.ProfileBanner, messageBuilder.String(), &keyboard)
}

func (b *Bot) handlePlansCommand(msg *tgbotapi.Message) {
	messageText, replyMarkup := b.getPlansMessageAndKeyboard()
	reply := tgbotapi.NewMessage(msg.Chat.ID, messageText)
	reply.ParseMode = tgbotapi.ModeHTML
	reply.ReplyMarkup = &replyMarkup
	b.API.Send(reply)
}

func (b *Bot) parseAllButtons(text string) *tgbotapi.InlineKeyboardMarkup {
	re := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	// --- LÓGICA DE ANÁLISE CORRIGIDA ---
	// 1. Divide a mensagem inteira em linhas.
	allLines := strings.Split(text, "\n")

	// 2. Se tivermos menos de 2 linhas (apenas a linha do /all), não há botões.
	if len(allLines) < 2 {
		return nil
	}

	// 3. A parte dos botões são TODAS as linhas após a primeira linha do comando.
	buttonsPart := strings.Join(allLines[1:], "\n")
	// --- FIM DA CORREÇÃO LÓGICA ---

	var keyboardRows [][]tgbotapi.InlineKeyboardButton
	lines := strings.Split(buttonsPart, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var row []tgbotapi.InlineKeyboardButton
		// Encontra todos os padrões de botão [texto](link) na linha atual.
		allMatches := re.FindAllStringSubmatch(line, -1)

		for _, matches := range allMatches {
			if len(matches) == 3 {
				label := matches[1]
				data := matches[2]

				if strings.HasPrefix(data, "http://") || strings.HasPrefix(data, "https://") {
					row = append(row, tgbotapi.NewInlineKeyboardButtonURL(label, data))
				} else {
					row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, data))
				}
			}
		}

		if len(row) > 0 {
			keyboardRows = append(keyboardRows, row)
		}
	}

	if len(keyboardRows) > 0 {
		markup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}
		return &markup
	}
	return nil
}

func (b *Bot) resolveUserID(s string) (int64, error) {
	if id, err := strconv.ParseInt(s, 10, 64); err == nil {
		return id, nil
	}
	chat, err := b.API.GetChat(tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{SuperGroupUsername: strings.TrimPrefix(s, "@")},
	})
	if err != nil {
		return 0, err
	}
	return chat.ID, nil
}

// Funções de Busca
func (b *Bot) handleSearchCommand(msg *tgbotapi.Message) {
	query := msg.CommandArguments()
	if query == "" {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Forneça um termo de pesquisa."))
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID
	idToCheck := userID
	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		idToCheck = chatID
	}

	canSearch, limit, user := b.checkAndResetSearchLimit(idToCheck)
	if !canSearch {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("✨ ꜰᴀᴢᴇʀ ᴜᴘɢʀᴀᴅᴇ ᴅᴇ ᴘʟᴀɴᴏ", "https://t.me/"+b.Config.AdminMention),
			),
		)
		replyText := fmt.Sprintf("❌ Você atingiu seu limite de %d buscas diárias. Tente novamente amanhã ou faça um upgrade.", limit)
		reply := tgbotapi.NewMessage(chatID, replyText)
		reply.ReplyToMessageID = msg.MessageID
		reply.ReplyMarkup = &keyboard
		b.API.Send(reply)
		return
	}

	msgConfig := tgbotapi.NewMessage(chatID, "🔍 Pesquisando, seja paciente...")
	msgConfig.ReplyToMessageID = msg.MessageID
	loadingMsg, _ := b.API.Send(msgConfig)

	isPremium := b.isUserPremium(idToCheck)
	color.Blue("   ⟫  USER %d (Premium: %t) SEARCHED FOR '%s'", userID, isPremium, query)

	results := search.Search(query)

	// Incrementa o contador de buscas e salva no DB
	if user != nil {
		user.SearchesToday++
		if err := b.Storage.AddOrUpdateUser(user); err != nil {
			color.Red("Erro ao atualizar contador de buscas para user %d: %v", user.ID, err)
		}
	}

	if len(results) == 0 {
		b.API.Send(tgbotapi.NewEditMessageText(chatID, loadingMsg.MessageID, fmt.Sprintf("❌ Nenhum resultado encontrado para: %s", query)))
		return
	}

	b.stateMutex.Lock()
	newState := &SearchState{
		Term:                 query,
		Results:              results,
		Offset:               0,
		MessageID:            loadingMsg.MessageID,
		ChatID:               chatID,
		UserCommandMessageID: msg.MessageID,
	}
	b.UserSearchStates[userID] = newState
	b.stateMutex.Unlock()

	b.stateMutex.RLock()
	state, ok := b.UserSearchStates[userID]
	b.stateMutex.RUnlock()
	if !ok {
		return
	}

	if isPremium {
		b.sendPremiumResultsPage(state)
	} else {
		totalOriginal := len(state.Results)
		if len(state.Results) > 15 {
			freeResults := make([]string, 15)
			copy(freeResults, state.Results[:15])
			b.sendFreeResultsPage(state, freeResults, totalOriginal)
		} else {
			b.sendFreeResultsPage(state, state.Results, totalOriginal)
		}
	}
}

// sendPremiumResultsPage foi refatorada para aceitar o estado como argumento.
// Isso evita a necessidade de locks internos e previne deadlocks.
func (b *Bot) sendPremiumResultsPage(state *SearchState) {
	// log.Printf(">>> Tentando enviar resultados premium para userID %d, state messageID %d", state.ChatID, state.MessageID)

	totalResults := len(state.Results)
	totalPages := (totalResults + 29) / 30
	if totalPages == 0 && totalResults > 0 {
		totalPages = 1
	}
	currentPage := (state.Offset / 30) + 1

	start := state.Offset
	end := state.Offset + 30
	if end > totalResults {
		end = totalResults
	}

	if start >= totalResults {
		// log.Printf("Aviso: Offset (%d) inválido para total de resultados (%d)", start, totalResults)
		return
	}

	resultsToShow := state.Results[start:end]

	var textBuilder strings.Builder
	textBuilder.WriteString(fmt.Sprintf("<blockquote>🔎 | SUA PESQUISA RETORNOU %d RESULTADOS, EXIBINDO (%d/%d):</blockquote>\n\n", totalResults, currentPage, totalPages))
	for _, line := range resultsToShow {
		if data := search.ParseLine(line); data != nil {
			textBuilder.WriteString(fmt.Sprintf("🧭: <code>%s</code>\n", data.URL))
			textBuilder.WriteString(fmt.Sprintf("👤: <code>%s</code>\n", data.User))
			textBuilder.WriteString(fmt.Sprintf("🔑: <code>%s</code>\n-\n", data.Password))
		} else {
			textBuilder.WriteString(fmt.Sprintf("%s\n-\n", line))
		}
	}

	var keyboardRows [][]tgbotapi.InlineKeyboardButton
	navRow := []tgbotapi.InlineKeyboardButton{}
	if currentPage > 1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅ ᴘʀᴇᴠ", "prev"))
	}
	if currentPage < totalPages {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("ɴᴇxᴛ ➡", "next"))
	}
	if len(navRow) > 0 {
		keyboardRows = append(keyboardRows, navRow)
	}
	actionRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⏬🗂️", "download"),
		tgbotapi.NewInlineKeyboardButtonData("🗑️", "delete_search"),
	)
	keyboardRows = append(keyboardRows, actionRow)
	replyMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}

	editMsg := tgbotapi.NewEditMessageText(state.ChatID, state.MessageID, textBuilder.String())
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &replyMarkup

	_, err := b.API.Send(editMsg)
	if err != nil {
		// log.Printf("!!! ERRO DETECTADO ao tentar editar a mensagem de resultados: %v", err)
	} else {
		// log.Printf(">>> Mensagem de resultados enviada/editada com sucesso para messageID %d", state.MessageID)
	}
}

func (b *Bot) handleDownload(query *tgbotapi.CallbackQuery, state *SearchState) {
	// A lógica de buscar o estado e trancar o mutex foi removida, pois agora recebemos o 'state' pronto.

	var contentBuilder strings.Builder
	contentBuilder.WriteString("╔═════════════════════════════════════════════════════╗\n")
	contentBuilder.WriteString("║      __        __   __            __   __   __      ║\n")
	contentBuilder.WriteString("║     / _` |    /  \\ |__) \\ / |    /  \\ / _` /__`     ║\n")
	contentBuilder.WriteString("║     \\__> |___ \\__/ |  \\  |  |___ \\__/ \\__> .__/     ║\n")
	contentBuilder.WriteString("║                                                     ║\n")
	contentBuilder.WriteString("╚═════════════════════════════════════════════════════╝\n")
	contentBuilder.WriteString("\n")

	contentBuilder.WriteString(fmt.Sprintf("ʀᴇꜱᴜʟᴛꜱ ᴏʙᴛᴀɪɴᴇᴅ ꜰᴏʀ ›%s‹, ʙʏ ʙᴏᴛ https://t.me/%s\n", state.Term, b.Config.BotUsername))
	contentBuilder.WriteString(fmt.Sprintf("ʙʏ t.me/%s\n\n", b.Config.AdminMention))
	contentBuilder.WriteString(fmt.Sprintf("ᴜꜱᴇʀ ᴡʜᴏ ᴍᴀᴅᴇ ᴛʜᴇ ꜱᴇᴀʀᴄʜ: @%s\n\n", query.From.UserName))
	contentBuilder.WriteString(strings.Repeat("-", 50) + "\n")

	for _, line := range state.Results {
		if data := search.ParseLine(line); data != nil {
			contentBuilder.WriteString(fmt.Sprintf("%s:%s:%s\n", data.URL, data.User, data.Password))

		} else {
			contentBuilder.WriteString(line + "\n")
		}
	}

	contentBuilder.WriteString(strings.Repeat("-", 50) + "\n")
	contentBuilder.WriteString(fmt.Sprintf("ᴇɴᴅ ᴏꜰ ᴄᴏɴꜱᴜʟᴛᴀᴛɪᴏɴ, ᴄᴏɴᴛɪɴᴜᴇ ᴏɴ t.me/%s\n", b.Config.BotUsername))

	invalidChars := ` <>:"/\|?*`
	sanitizedTerm := state.Term
	for _, char := range invalidChars {
		sanitizedTerm = strings.ReplaceAll(sanitizedTerm, string(char), "_")
	}

	filename := fmt.Sprintf("%s-@%s.txt", sanitizedTerm, b.Config.BotUsername)
	fileBytes := []byte(contentBuilder.String())
	fileToSend := tgbotapi.FileBytes{Name: filename, Bytes: fileBytes}

	doc := tgbotapi.NewDocument(state.ChatID, fileToSend)
	doc.ReplyToMessageID = state.MessageID
	b.API.Send(doc)

	// Responde ao clique no botão para que ele pare de "carregar"
	answerCallback := tgbotapi.NewCallback(query.ID, "Arquivo de resultados enviado!")
	b.API.Request(answerCallback)
}
func (b *Bot) sendFreeResultsPage(state *SearchState, resultsToShow []string, totalOriginal int) {
	resposta := fmt.Sprintf("<blockquote>🔎 | SUA PESQUISA RETORNOU %d RESULTADOS;\n📌 | EXIBINDO %d/%d DO PLANO FREE;\n🦅 | <b>FAÇA UPGRADE PARA DOWNLOAD DE TODOS OS RESUTADOS NO FORMATO U:L:P</b></blockquote>\n\n", totalOriginal, len(resultsToShow), totalOriginal)
	var textBuilder strings.Builder
	textBuilder.WriteString(resposta)

	for _, line := range resultsToShow {
		if data := search.ParseLine(line); data != nil {
			textBuilder.WriteString(fmt.Sprintf("🧭: <code>%s</code>\n", data.URL))
			textBuilder.WriteString(fmt.Sprintf("👤: <code>%s</code>\n", data.User))
			textBuilder.WriteString(fmt.Sprintf("🔑: <code>%s</code>\n-\n", data.Password))
		} else {
			textBuilder.WriteString(fmt.Sprintf("%s\n-\n", line))
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✨ ꜰᴀᴢᴇʀ ᴜᴘɢʀᴀᴅᴇ ᴘᴀʀᴀ ᴘʀᴇᴍɪᴜᴍ", "show_plans"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(state.ChatID, state.MessageID, textBuilder.String())
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.ReplyMarkup = &keyboard

	if _, err := b.API.Send(editMsg); err != nil {
		if !strings.Contains(err.Error(), "message is not modified") {
			// log.Printf("Erro ao editar mensagem (free): %v", err)
		}
	}
}

func (b *Bot) checkAndResetSearchLimit(userID int64) (canSearch bool, limit int, user *storage.User) {
	if userID == b.Config.AdminUserID {
		return true, 99999, nil
	}

	user, err := b.Storage.GetUser(userID)
	if err != nil {
		log.Printf("Erro ao buscar usuário %d no DB: %v", userID, err)
		return false, 0, nil // Bloqueia em caso de erro no DB
	}

	// Se o usuário não existe, cria um novo (plano gratuito)
	if user == nil {
		user = &storage.User{
			ID:               userID,
			RegistrationDate: time.Now().UTC().Format("2006-01-02"),
			EndDate:          "",
			Premium:          "n",
			DailyLimit:       3, // Limite padrão para usuários gratuitos
			SearchesToday:    0,
			LastSearchDate:   "",
		}
	}

	// Reseta o limite diário se for um novo dia
	todayStr := time.Now().UTC().Format("2006-01-02")
	if user.LastSearchDate != todayStr {
		user.SearchesToday = 0
		user.LastSearchDate = todayStr
	}

	// Salva as alterações (seja um novo usuário ou um reset de data)
	if err := b.Storage.AddOrUpdateUser(user); err != nil {
		log.Printf("Erro ao salvar usuário %d no DB: %v", userID, err)
		return false, 0, nil // Bloqueia em caso de erro
	}

	if user.SearchesToday >= user.DailyLimit {
		return false, user.DailyLimit, user
	}

	return true, user.DailyLimit, user
}

// handleDeleteSearch apaga a mensagem de resultados e o comando original do usuário.
func (b *Bot) handleDeleteSearch(query *tgbotapi.CallbackQuery, state *SearchState) {
	// log.Printf("Usuário %d solicitou apagar a pesquisa (msg do bot: %d, msg do usuário: %d)", query.From.ID, state.MessageID, state.UserCommandMessageID)

	// Apaga a mensagem de resultados do bot
	botMsgDelete := tgbotapi.NewDeleteMessage(state.ChatID, state.MessageID)
	if _, err := b.API.Send(botMsgDelete); err != nil {
		// log.Printf("Erro ao apagar a mensagem do bot: %v", err)
	}

	// Apaga a mensagem original de /search do usuário
	userMsgDelete := tgbotapi.NewDeleteMessage(state.ChatID, state.UserCommandMessageID)
	if _, err := b.API.Send(userMsgDelete); err != nil {
		// log.Printf("Erro ao apagar a mensagem de comando do usuário: %v", err)
	}

	// Remove o estado da memória para invalidar ações futuras (como paginação em uma mensagem apagada)
	// A trava de escrita já está ativa em handleCallbackQuery, então esta operação é segura.
	delete(b.UserSearchStates, query.From.ID)
}

func (b *Bot) handleDeleteBroadcast(query *tgbotapi.CallbackQuery) {
	// log.Printf("Usuário %d solicitou apagar a mensagem de broadcast (ID: %d)", query.From.ID, query.Message.MessageID)

	// Cria a configuração para apagar a mensagem
	deleteMsg := tgbotapi.NewDeleteMessage(query.Message.Chat.ID, query.Message.MessageID)

	// Envia o pedido de exclusão para a API do Telegram
	if _, err := b.API.Send(deleteMsg); err != nil {
		// log.Printf("Erro ao apagar a mensagem de broadcast: %v", err)
		// Avisa ao usuário que não foi possível apagar
		callback := tgbotapi.NewCallback(query.ID, "Erro ao apagar a mensagem.")
		b.API.Request(callback)
	} else {
		// Apenas responde ao callback para confirmar a ação, sem texto.
		callback := tgbotapi.NewCallback(query.ID, "")
		b.API.Request(callback)
	}
}

// getSystemStats coleta informações em tempo real do hardware do servidor.
func getSystemStats() (map[string]string, error) {
	stats := make(map[string]string)

	// Informações do Host (Sistema Operacional)
	hostInfo, err := host.Info()
	if err != nil {
		return nil, err
	}
	stats["os"] = fmt.Sprintf("%s (%s)", hostInfo.Platform, hostInfo.PlatformFamily)
	stats["uptime"] = fmt.Sprintf("%d horas", hostInfo.Uptime/3600)

	// Informações da CPU
	cpuInfo, err := cpu.Info()
	if err != nil || len(cpuInfo) == 0 {
		return nil, err
	}
	stats["cpuModel"] = cpuInfo[0].ModelName
	stats["cpuCores"] = strconv.Itoa(len(cpuInfo))
	// Coleta o uso da CPU durante 1 segundo para obter um valor preciso
	cpuUsage, err := cpu.Percent(time.Second, false)
	if err != nil || len(cpuUsage) == 0 {
		return nil, err
	}
	stats["cpuUsage"] = fmt.Sprintf("%.2f%%", cpuUsage[0])

	// Informações de Memória (RAM)
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	stats["memTotal"] = fmt.Sprintf("%d GB", memInfo.Total/1024/1024/1024)
	stats["memUsage"] = fmt.Sprintf("%.2f%%", memInfo.UsedPercent)

	// Informações do Disco (partição raiz "/")
	diskInfo, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}
	stats["diskTotal"] = fmt.Sprintf("%d GB", diskInfo.Total/1024/1024/1024)
	stats["diskUsage"] = fmt.Sprintf("%.2f%%", diskInfo.UsedPercent)

	return stats, nil
}

func (b *Bot) handleCloudCommand(msg *tgbotapi.Message) {
	if msg.From.ID != b.Config.AdminUserID {
		return
	}
	if msg.ReplyToMessage == nil || msg.ReplyToMessage.Document == nil {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Uso: Responda a uma mensagem que contenha um arquivo .txt com o comando /cloud."))
		return
	}
	doc := msg.ReplyToMessage.Document
	if !strings.HasSuffix(strings.ToLower(doc.FileName), ".txt") {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ O arquivo precisa ser do tipo .txt."))
		return
	}

	// --- CORREÇÃO APLICADA AQUI ---
	// Passo 1: Obter informações do arquivo usando o FileID.
	fileConfig := tgbotapi.FileConfig{FileID: doc.FileID}
	file, err := b.API.GetFile(fileConfig)
	if err != nil {
		color.Red("Erro ao obter informações do arquivo: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Não foi possível obter as informações do arquivo no Telegram."))
		return
	}

	// Passo 2: Construir o link de m usando o FilePath retornado.
	// O método .Link() da biblioteca faz isso automaticamente.
	fileURL := file.Link(b.API.Token)
	// --- FIM DA CORREÇÃO ---

	color.Blue("Baixando arquivo de: %s", fileURL)
	resp, err := http.Get(fileURL)
	if err != nil {
		color.Red("Erro ao baixar o arquivo: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Falha ao baixar o arquivo a partir do link."))
		return
	}
	defer resp.Body.Close()

	localFilePath := filepath.Join(config.LogsPath, doc.FileName)
	outFile, err := os.Create(localFilePath)
	if err != nil {
		color.Red("Erro ao criar arquivo local: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Falha ao criar o arquivo no servidor."))
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		color.Red("Erro ao salvar conteúdo do arquivo: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Falha ao salvar o conteúdo no arquivo do servidor."))
		return
	}

	color.Green("✅ Arquivo '%s' baixado e adicionado à base de dados de logs.", doc.FileName)
	replyMsg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ Arquivo <code>%s</code> adicionado com sucesso à base de logs!", doc.FileName))
	replyMsg.ParseMode = tgbotapi.ModeHTML
	b.API.Send(replyMsg)
}

func (b *Bot) handleReleaseCommand(msg *tgbotapi.Message) {
	if msg.From.ID != b.Config.AdminUserID {
		return
	}

	args := msg.CommandArguments()
	if args == "" {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Uso: /release <XXXmb> ou /release <termo>\nEx: /release 100mb\nEx: /release inurl:example.com"))
		return
	}

	// Verifica se o argumento é um tamanho (ex: 100mb)
	if strings.HasSuffix(strings.ToLower(args), "mb") {
		sizeStr := strings.TrimSuffix(strings.ToLower(args), "mb")
		size, err := strconv.Atoi(sizeStr)
		if err != nil {
			b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Tamanho inválido. Use um número, como em `100mb`."))
			return
		}
		bytesToFree := int64(size) * 1024 * 1024
		freedBytes, filesDeleted := b.releaseBySize(bytesToFree)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ Operação concluída!\n- Arquivos Deletados: %d\n- Espaço Liberado: %.2f MB", filesDeleted, float64(freedBytes)/1024/1024)))
	} else {
		// Se não for um tamanho, trata como um termo de busca para apagar linhas
		linesDeleted := b.releaseByContent(args)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ Operação concluída!\n- Linhas correspondentes a '%s' foram deletadas.\n- Total de Linhas Removidas: %d", args, linesDeleted)))
	}
}

// 2. FUNÇÃO AJUDANTE PARA LIBERAR ESPAÇO POR TAMANHO
func (b *Bot) releaseBySize(bytesToFree int64) (int64, int) {
	var files []os.FileInfo
	filepath.Walk(config.LogsPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			files = append(files, info)
		}
		return nil
	})

	// Ordena os arquivos do mais antigo para o mais novo
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().Before(files[j].ModTime())
	})

	var totalFreed int64
	var filesProcessed int

	for _, fileInfo := range files {
		if totalFreed >= bytesToFree {
			break
		}

		filePath := filepath.Join(config.LogsPath, fileInfo.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			color.Red("NÃO FOI POSSÍVEL LER O ARQUIVO %s PARA LIBERAR ESPAÇO: %v", filePath, err)
			continue
		}

		bytesRemainingToFree := bytesToFree - totalFreed
		var bytesToRemoveFromFile int64

		if fileInfo.Size() <= bytesRemainingToFree {
			// Se o arquivo inteiro for menor que o que falta, ele será removido.
			bytesToRemoveFromFile = fileInfo.Size()
			if err := os.Remove(filePath); err == nil {
				color.Red("   ⟫  DELETED FILE: %s", fileInfo.Name())
				totalFreed += bytesToRemoveFromFile
				filesProcessed++
			}
		} else {
			// Se o arquivo for maior, removemos apenas as linhas necessárias do início.
			lines := strings.Split(string(content), "\n")
			var cutoffIndex int
			var bytesCounted int64

			for i, line := range lines {
				// +1 para contar o caractere de nova linha '\n'
				bytesCounted += int64(len(line) + 1)
				if bytesCounted >= bytesRemainingToFree {
					cutoffIndex = i + 1
					bytesToRemoveFromFile = bytesCounted
					break
				}
			}

			// Mantém apenas as linhas após o ponto de corte
			linesToKeep := lines[cutoffIndex:]
			newContent := strings.Join(linesToKeep, "\n")

			// Sobrescreve o arquivo com o conteúdo reduzido
			if err := os.WriteFile(filePath, []byte(newContent), 0644); err == nil {
				color.Red("   ⟫  REMOVED %d LINES FROM: %s", cutoffIndex, fileInfo.Name())
				totalFreed += bytesToRemoveFromFile
				filesProcessed++
			}
		}
	}

	// O segundo valor retornado agora é "arquivos processados" em vez de "arquivos deletados"
	return totalFreed, filesProcessed
}

// 3. FUNÇÃO AJUDANTE PARA LIBERAR ESPAÇO POR CONTEÚDO
func (b *Bot) releaseByContent(query string) int {
	criteria := search.ParseSearchQuery(query)
	if len(criteria) == 0 {
		return 0
	}

	var totalLinesDeleted int
	filepath.Walk(config.LogsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".txt") {
			return nil
		}

		file, _ := os.Open(path)
		var linesToKeep []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			lineData := search.ParseLine(line)
			// Se a linha NÃO corresponde ao critério, nós a mantemos
			if lineData == nil || !search.LineMatchesCriteria(lineData, criteria) {
				linesToKeep = append(linesToKeep, line)
			} else {
				totalLinesDeleted++
			}
		}
		file.Close()

		// Sobrescreve o arquivo original apenas com as linhas que devem ser mantidas
		if totalLinesDeleted > 0 {
			color.Red("   ⟫  REMOVING %d LINES WITH '%s' FROM %s", totalLinesDeleted, query, info.Name())
			os.WriteFile(path, []byte(strings.Join(linesToKeep, "\n")), 0644)
		}
		return nil
	})
	return totalLinesDeleted
}

// handleUsersCommand lista todos os usuários do bot para o admin.
func (b *Bot) handleUsersCommand(msg *tgbotapi.Message) {
	if msg.From.ID != b.Config.AdminUserID {
		return
	}

	users, err := b.Storage.GetAllUsers()
	if err != nil {
		log.Printf("Erro ao buscar todos os usuários: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Erro ao buscar a lista de usuários."))
		return
	}

	if len(users) == 0 {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "ℹ️ Nenhum usuário registrado no banco de dados."))
		return
	}

	var messageBuilder strings.Builder
	messageBuilder.WriteString(fmt.Sprintf("👥 <b>Lista de Usuários (%d)</b>\n\n", len(users)))

	for i, user := range users {
		premiumIcon := "❌"
		if user.Premium == "y" {
			premiumIcon = "✅"
		}

		// Constrói a linha para cada usuário
		userInfo := fmt.Sprintf(
			"<b>ID:</b> <code>%d</code>\n"+
				"<b>Link:</b> <a href=\"tg://user?id=%d\">Perfil</a>\n"+
				"<b>Premium:</b> %s\n"+
				"<b>Expira em:</b> %s\n"+
				"<b>Limite Diário:</b> %d\n"+
				"-----------------------------------\n",
			user.ID, user.ID, premiumIcon, user.EndDate, user.DailyLimit,
		)

		// Lógica para enviar mensagens em pedaços para não exceder o limite do Telegram
		if messageBuilder.Len()+len(userInfo) > 4000 {
			reply := tgbotapi.NewMessage(msg.Chat.ID, messageBuilder.String())
			reply.ParseMode = tgbotapi.ModeHTML
			b.API.Send(reply)
			messageBuilder.Reset()
			messageBuilder.WriteString("...(continuação)\n\n")
		}
		messageBuilder.WriteString(userInfo)

		// Envia a última parte se for o final da lista
		if i == len(users)-1 {
			reply := tgbotapi.NewMessage(msg.Chat.ID, messageBuilder.String())
			reply.ParseMode = tgbotapi.ModeHTML
			b.API.Send(reply)
		}
	}
}

// handleInvitesCommand lista todos os convites gerados.
func (b *Bot) handleInvitesCommand(msg *tgbotapi.Message) {
	if msg.From.ID != b.Config.AdminUserID {
		return
	}

	invites, err := b.Storage.GetAllInvites()
	if err != nil {
		log.Printf("Erro ao buscar todos os convites: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Erro ao buscar a lista de convites."))
		return
	}

	if len(invites) == 0 {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "ℹ️ Nenhum convite encontrado."))
		return
	}

	message := "🎟️ <b>Lista de Convites Ativos</b>\n\n"
	for _, invite := range invites {
		message += fmt.Sprintf(
			"<b>Código:</b> <code>%s</code>\n"+
				"<b>Link:</b> <code>https://t.me/%s?start=%s</code>\n"+
				"<b>Usos:</b> %d / %d\n"+
				"<b>Duração:</b> %d dias\n"+
				"-----------------------------------\n",
			invite.Code, b.Config.BotUsername, invite.Code, invite.Used, invite.Limit, invite.Days,
		)
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, message)
	reply.ParseMode = tgbotapi.ModeHTML
	b.API.Send(reply)
}

// handleTerminalCommand define o canal para receber os logs do sistema.
func (b *Bot) handleTerminalCommand(msg *tgbotapi.Message) {
	if msg.From.ID != b.Config.AdminUserID {
		return
	}

	// O comando deve ser usado no canal que o admin quer definir como terminal
	if msg.Chat.ID == msg.From.ID {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Este comando deve ser usado dentro do canal privado que você deseja usar como terminal de logs."))
		return
	}

	channelID := msg.Chat.ID
	err := b.Storage.SetSetting("terminal_channel_id", strconv.FormatInt(channelID, 10))
	if err != nil {
		log.Printf("Erro ao salvar ID do terminal no DB: %v", err)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Erro ao configurar o terminal."))
		return
	}

	successMsg := fmt.Sprintf("✅ Terminal configurado para este canal (<code>%d</code>).\n\nO bot precisa ser <b>reiniciado</b> na VM para que o encaminhamento de logs comece a funcionar.", channelID)
	reply := tgbotapi.NewMessage(channelID, successMsg)
	reply.ParseMode = tgbotapi.ModeHTML
	b.API.Send(reply)
}
