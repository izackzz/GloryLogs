from bot.imports import *
from bot.config import *

# ---------- FUNÇÕES SIMPLES DE TRABALHO DO BOT ---------- #


def load_users() -> dict[int, dict]:
    """Carrega o CSV de usuários, criando-o se não existir."""
    users: dict[int, dict] = {}
    if not os.path.exists(USERS_CSV):
        with open(USERS_CSV, "w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=FIELDNAMES)
            writer.writeheader()
        return users  # retorna dict vazio

    with open(USERS_CSV, "r", newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            uid = int(row["user"])
            users[uid] = row
    return users

def save_users(users: dict[int, dict]) -> None:
    """Salva o dicionário de usuários de volta no CSV."""
    with open(USERS_CSV, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=FIELDNAMES)
        writer.writeheader()
        for uid, rec in users.items():
            writer.writerow({
                "user":              uid,
                "registration-date": rec["registration-date"],
                "end-date":          rec["end-date"],
                "premium":           rec["premium"],
            })

USERS = load_users()
save_users(USERS)

def load_chats() -> set[int]:
    chats: set[int] = set()
    if os.path.exists(CHATS_FILE):
        with open(CHATS_FILE, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if line:
                    chats.add(int(line))
    return chats

def save_chats(chats: set[int]) -> None:
    with open(CHATS_FILE, "w", encoding="utf-8") as f:
        for cid in sorted(chats):
            f.write(f"{cid}\n")

CHATS = load_chats()

def register_chat(update: Update) -> None:
    """Registra o chat (privado ou grupo) onde houve interação."""
    cid = update.effective_chat.id
    if cid not in CHATS:
        CHATS.add(cid)
        save_chats(CHATS)

def is_user_premium(user_id: int) -> bool:
    """Retorna True se este usuário individual tiver premium ativo."""
    rec = USERS.get(user_id)
    if not rec or rec.get("premium") != "y":
        return False

    # parse da data de fim e comparação com agora (UTC timezone-aware)
    end_date = datetime.fromisoformat(rec["end-date"]).date()
    today_utc = datetime.now(timezone.utc).date()
    return end_date >= today_utc

def is_chat_premium(chat_id: int) -> bool:
    """Retorna True se este chat (grupo) tiver premium ativo."""
    rec = USERS.get(chat_id)
    if not rec or rec.get("premium") != "y":
        return False

    end_date = datetime.fromisoformat(rec["end-date"]).date()
    today_utc = datetime.now(timezone.utc).date()
    return end_date >= today_utc

def parse_search_query(query):
    pattern = r"(?P<operator>inurl:|intext:|site:|filetype:)?(?P<term>\"[^\"]+\"|\S+)"
    criteria = []
    for match in re.finditer(pattern, query):
        op = match.group("operator")
        term = match.group("term")
        if term.startswith('"') and term.endswith('"'):
            term = term.strip('"')
            if not op:
                op = "phrase"
            else:
                op = op[:-1]
        else:
            if op:
                op = op[:-1]
                term = term.strip('"')
            else:
                op = "term"
                term = term.strip('"')
        criteria.append((op.lower(), term))
    return criteria

def extract_domain(url):
    try:
        parsed_url = urlparse(url)
        return parsed_url.netloc
    except Exception:
        return url  # Retorna a URL inteira se falhar ao extrair o domínio

def line_matches_criteria(url, user, password, criteria):
    for op, term in criteria:
        term_lower = term.lower()
        if op == "inurl":
            if term_lower not in url.lower():
                return False
        elif op == "site":
            domain = extract_domain(url).lower()
            if term_lower not in domain:
                return False
        elif op == "filetype":
            if not url.lower().endswith("." + term_lower):
                return False
        elif op == "intext":
            if term_lower not in user.lower() and term_lower not in password.lower():
                return False
        elif op == "phrase":
            if (
                term_lower not in url.lower()
                and term_lower not in user.lower()
                and term_lower not in password.lower()
            ):
                return False
        elif op == "term":
            if (
                term_lower not in url.lower()
                and term_lower not in user.lower()
                and term_lower not in password.lower()
            ):
                return False
        else:
            return False
    return True

def parse_line(linha):
    linha = linha.strip()
    parts = linha.rsplit(":", 2)
    if len(parts) == 3:
        url, user, senha = parts
        return url, user, senha
    else:
        return None

def format_size(size_bytes):
    if size_bytes == 0:
        return "0B"
    size_name = ("B", "KB", "MB", "GB", "TB")
    i = 0
    p = size_bytes
    while p >= 1024 and i < len(size_name) - 1:
        p /= 1024.0
        i += 1
    return f"{p:.2f} {size_name[i]}"

def parse_all_buttons(command_text: str) -> InlineKeyboardMarkup | None:
    """
    Espera command_text contendo, após '/all', linhas de '[label](url)' separadas
    por '|' (mesma fileira) ou newline (fileiras distintas). Retorna um InlineKeyboardMarkup.
    """
    pattern = r'\[([^\]]+)\]\((https?://[^\)]+)\)'
    # remove o '/all' e tudo até o primeiro newline ou espaço
    parts = command_text.split(None, 1)
    buttons_part = parts[1] if len(parts) > 1 else ""
    rows: list[list[InlineKeyboardButton]] = []
    for line in buttons_part.splitlines():
        cells = []
        for segment in line.split("|"):
            seg = segment.strip()
            m = re.match(pattern, seg)
            if m:
                label, url = m.group(1), m.group(2)
                cells.append(InlineKeyboardButton(label, url=url))
        if cells:
            rows.append(cells)
    return InlineKeyboardMarkup(rows) if rows else None
