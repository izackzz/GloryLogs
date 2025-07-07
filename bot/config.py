
from bot.imports import *
from dotenv import load_dotenv

load_dotenv()

BOT_TOKEN = os.getenv("BOT_TOKEN_ENV")
ADMIN_USER_ID = os.getenv("ADMIN_USER_ID_ENV")
BOT_USERNAME = os.getenv("BOT_USERNAME_ENV")
ADMIN_MENTION = os.getenv("ADMIN_MENTION_ENV")

if not BOT_TOKEN or not ADMIN_USER_ID:
    raise ValueError("Erro: BOT_TOKEN_ENV e ADMIN_USER_ID_ENV devem ser definidos no arquivo .env")

try:
    ADMIN_USER_ID = int(ADMIN_USER_ID)
except (ValueError, TypeError):
    raise ValueError("Erro: ADMIN_USER_ID_ENV no arquivo .env deve ser um número inteiro.")


# PATHS
LOGS_PATH = "logs"
START_BANNER = "bg/bg.png"
PROFILE_BANNER = "bg/mkt.jpg"

# USERS DATABASE FILES
USERS_CSV = "db/users.csv"
FIELDNAMES = [
    "user", "registration-date", "end-date", "premium",
    "daily_limit", "searches_today", "last_search_date"
]

# CHAT INTERATION FILE
CHATS_FILE = "db/chats.txt"

# INVITES FILE
INVITES_CSV = "db/invites.csv"
INVITE_FIELDNAMES = ["code", "days", "limit", "used"]



userbot_state = {}
