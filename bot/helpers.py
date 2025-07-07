from bot.imports import *
from bot.config import *

# ---------- FUNÇÕES SIMPLES DE TRABALHO DO BOT ---------- #

# internal/storage/storage.go
def load_users() -> dict[int, dict]:
    """Carrega o CSV de usuários, criando-o se não existir e tratando novos campos."""
    users: dict[int, dict] = {}
    if not os.path.exists(USERS_CSV):
        with open(USERS_CSV, "w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=FIELDNAMES)
            writer.writeheader()
        return users

    with open(USERS_CSV, "r", newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            uid = int(row["user"])
            # Define valores padrão para os novos campos se eles não existirem (compatibilidade)
            row.setdefault("daily_limit", "100")
            row.setdefault("searches_today", "0")
            row.setdefault("last_search_date", "")
            
            # Converte para os tipos corretos
            row["daily_limit"] = int(row["daily_limit"])
            row["searches_today"] = int(row["searches_today"])
            
            users[uid] = row
    return users

def save_users(users: dict[int, dict]) -> None:
    """Salva o dicionário de usuários de volta no CSV com todos os campos."""
    with open(USERS_CSV, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=FIELDNAMES)
        writer.writeheader()
        for uid, rec in users.items():
            writer.writerow({
                "user":              uid,
                "registration-date": rec["registration-date"],
                "end-date":          rec["end-date"],
                "premium":           rec["premium"],
                # Garante que os novos campos existam ao salvar
                "daily_limit":       rec.get("daily_limit", 99999),
                "searches_today":    rec.get("searches_today", 0),
                "last_search_date":  rec.get("last_search_date", ""),
            })

USERS = load_users()

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

# ---------- NOVAS FUNÇÕES PARA O SISTEMA DE CONVITES ---------- #

def generate_invite_code(length: int = 8) -> str:
    """Gera um código de convite alfanumérico aleatório."""
    alphabet = string.ascii_letters + string.digits
    return ''.join(secrets.choice(alphabet) for _ in range(length))

def load_invites() -> dict[str, dict]:
    """Carrega o CSV de convites, criando-o se não existir."""
    invites: dict[str, dict] = {}
    if not os.path.exists(INVITES_CSV):
        with open(INVITES_CSV, "w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=INVITE_FIELDNAMES)
            writer.writeheader()
        return invites

    with open(INVITES_CSV, "r", newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            # Converte os valores numéricos para o tipo correto
            row['days'] = int(row['days'])
            row['limit'] = int(row['limit'])
            row['used'] = int(row['used'])
            invites[row["code"]] = row
    return invites

def save_invites(invites: dict[str, dict]) -> None:
    """Salva o dicionário de convites de volta no CSV."""
    with open(INVITES_CSV, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=INVITE_FIELDNAMES)
        writer.writeheader()
        for code, data in invites.items():
            writer.writerow(data)

# Carrega os convites na inicialização do bot
INVITES = load_invites()

# ---------- LIMITE DIÁRIO DE BUSCAS ---------- #

def check_and_reset_search_limit(user_id: int) -> bool:
    """
    Verifica se o usuário pode realizar uma busca. Reseta a contagem se for um novo dia.
    Agora também lida com usuários 'free' que não são premium.
    Retorna True se a busca for permitida, False caso contrário.
    """
    # Admin não tem limite
    if user_id == ADMIN_USER_ID:
        return True
    
    # Se o usuário não existe no CSV, cria um registro temporário para o plano FREE
    if user_id not in USERS:
        USERS[user_id] = {
            "user": str(user_id),
            "registration-date": "N/A",
            "end-date": "N/A",
            "premium": "n",
            "daily_limit": 3,  # Limite do plano FREE
            "searches_today": 0,
            "last_search_date": "",
        }

    rec = USERS[user_id]
    today_str = datetime.now(timezone.utc).date().isoformat()
    last_search_date = rec.get("last_search_date", "")

    # Se a última busca foi em um dia diferente, reseta a contagem
    if last_search_date != today_str:
        rec["searches_today"] = 0
        rec["last_search_date"] = today_str

    # Verifica se o limite foi atingido
    if rec["searches_today"] >= rec.get("daily_limit", 3): # Usa 3 como fallback
        return False  # Limite atingido

    return True  # Busca permitida

def get_plans_message_and_keyboard() -> tuple[str, InlineKeyboardMarkup]:
    """
    Monta e retorna a mensagem de texto e o teclado (botões) para a página de planos.
    """
    # Você pode personalizar esta mensagem com os detalhes dos seus planos
    message_text = (
        "<blockquote>✨ CONHEÇA O PREMIUM ✨</blockquote>\n\n"
        "⁝⁝⁝ Desbloqueie todo o potencial do bot com acesso ilimitado e as melhores funcionalidades.\n\n"
        
        "<blockquote>⁝⁝⁝ <b>Plano Mensal - R$ 25,00</b>\n"
        "✓ Acesso completo por 30 dias\n"
        "✓ Buscas diárias ilimitadas\n"
        "✓ Resultados ilimitados por busca\n"
        "✓ Suporte prioritário</blockquote>\n\n"
        
        "<blockquote>⁝⁝⁝ <b>Plano Trimestral - R$ 60,00</b> (Economize 20%!)\n"
        "✓ Todos os benefícios do plano mensal\n"
        "✓ Acesso garantido por 90 dias</blockquote>\n\n"
        
        "<blockquote>⁝⁝⁝ Para adquirir ou tirar dúvidas, fale com um administrador.</blockquote>"
    )
    
    keyboard = [[InlineKeyboardButton("🛍️ Falar com Administrador", url="https://t.me/yMusashi")]]
    reply_markup = InlineKeyboardMarkup(keyboard)
    
    return message_text, reply_markup