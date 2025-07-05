package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// ADICIONAMOS O CAMPO PlansMessage E PlansBanner
type Config struct {
	BotToken     string
	AdminUserID  int64
	BotUsername  string
	AdminMention string
	PlansMessage string
	// Caminhos para os banners, agora carregados da config
	StartBanner   string
	ProfileBanner string
	PlansBanner   string
}

const (
	LogsPath      = "logs"
	StartBanner   = "bg/bg.png"
	ProfileBanner = "bg/mkt.jpg"
	UsersCSV      = "db/users.csv"
	ChatsFile     = "db/chats.txt"
	InvitesCSV    = "db/invites.csv"
)

// Fieldnames (cabeçalhos) para os arquivos CSV.
var (
	UserFieldnames = []string{
		"user", "registration-date", "end-date", "premium",
		"daily_limit", "searches_today", "last_search_date",
	}
	InviteFieldnames = []string{"code", "days", "limit", "used"}
)

// LoadConfig carrega as configurações a partir do arquivo .env.
func LoadConfig() Config {
	_ = godotenv.Load()

	botToken := os.Getenv("BOT_TOKEN_ENV")
	adminIDStr := os.Getenv("ADMIN_USER_ID_ENV")

	if botToken == "" || adminIDStr == "" {
		log.Fatalf("Erro: BOT_TOKEN_ENV e ADMIN_USER_ID_ENV devem ser definidos no arquivo .env")
	}

	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Erro: ADMIN_USER_ID_ENV no arquivo .env deve ser um número inteiro: %v", err)
	}

	// Carrega os caminhos dos banners ou usa um valor padrão
	startBanner := os.Getenv("START_BANNER_ENV")
	if startBanner == "" {
		startBanner = "bg/bg.png" // Padrão
	}
	profileBanner := os.Getenv("PROFILE_BANNER_ENV")
	if profileBanner == "" {
		profileBanner = "bg/mkt.jpg" // Padrão
	}
	plansBanner := os.Getenv("PLANS_BANNER_ENV")
	if plansBanner == "" {
		plansBanner = "bg/plans_banner.jpg" // Padrão
	}

	return Config{
		BotToken:      botToken,
		AdminUserID:   adminID,
		BotUsername:   os.Getenv("BOT_USERNAME_ENV"),
		AdminMention:  os.Getenv("ADMIN_MENTION_ENV"),
		PlansMessage:  os.Getenv("PLANS_MESSAGE_ENV"),
		StartBanner:   startBanner,
		ProfileBanner: profileBanner,
		PlansBanner:   plansBanner,
	}
}
