package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken     string
	AdminUserID  int64
	BotUsername  string
	AdminMention string
	PlansMessage string
	StartBanner  string
	ProfileBanner string
	PlansBanner  string
}

const (
	LogsPath    = "logs"
	DBPath      = "db/glorylogs.db" // <-- NOVO: Caminho para o arquivo do banco de dados
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

	startBanner := os.Getenv("START_BANNER_ENV")
	if startBanner == "" {
		startBanner = "bg/bg.png"
	}
	profileBanner := os.Getenv("PROFILE_BANNER_ENV")
	if profileBanner == "" {
		profileBanner = "bg/mkt.jpg"
	}
	plansBanner := os.Getenv("PLANS_BANNER_ENV")
	if plansBanner == "" {
		plansBanner = "bg/plans_banner.jpg"
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