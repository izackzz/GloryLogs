from bot.imports import *
from bot.config import *
from bot.helpers import *

userbot = {}
#
async def handle_invitation(update: Update, context: ContextTypes.DEFAULT_TYPE, code: str) -> None:
    """Processa a ativação de premium via link de convite."""
    user_id = update.effective_user.id
    user_name = update.effective_user.username or update.effective_user.first_name

    # 1. Verifica se o código existe
    if code not in INVITES:
        await update.message.reply_text("❌ Código de convite inválido ou expirado.")
        return

    invite_data = INVITES[code]

    # 2. Verifica se o usuário já é premium
    if is_user_premium(user_id):
        await update.message.reply_text("✅ Você já possui uma assinatura premium ativa!")
        return

    # 3. Verifica se o limite de uso do convite foi atingido
    if invite_data["used"] >= invite_data["limit"]:
        await update.message.reply_text("❌ Este link de convite já atingiu o limite máximo de usos.")
        return

    # 4. Ativa o premium para o usuário
    days = invite_data["days"]
    now = datetime.now(timezone.utc)
    start_date = now.date()
    end_date = (now + timedelta(days=days)).date()
    
    print("")
    print(f"\033[1;34m   ⟫  USER {user_id} JOIN WITH INVITE CODE {code}\033[m")

    USERS[user_id] = {
        "user": str(user_id),
        "registration-date": start_date.isoformat(),
        "end-date": end_date.isoformat(),
        "premium": "y",
        "daily_limit": 15,  # NOVO: Limite padrão para convites
        "searches_today": 0,
        "last_search_date": "",
    }
    save_users(USERS)

    # 5. Atualiza a contagem de uso do convite
    INVITES[code]["used"] += 1
    save_invites(INVITES)

    # 6. Envia mensagem de boas-vindas
    mensagem = (
        f"<blockquote>🎉 Parabéns, {user_name}</blockquote>!\n\n"
        ""
        f"Sua assinatura premium foi ativada com sucesso por <b>{days} dias</b>.\n\n"
        "Para começar, use o comando <b>/search</b>. Para ver todos os comandos, digite <b>/help</b>."
    )
    await update.message.reply_text(mensagem, parse_mode=ParseMode.HTML)
#
async def invite_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """Gera um link de convite para novos usuários (somente admin)."""
    # 1. Verifica se é o admin
    if update.effective_user.id != ADMIN_USER_ID:
        return

    # 2. Valida o formato do comando
    # Ex: /invite 30 max:10
    args = context.args
    if len(args) != 2 or not args[0].isdigit() or not args[1].lower().startswith("max:"):
        await update.message.reply_text(
            "⚠️ Uso incorreto. Formato: <code>/invite &lt;dias&gt; max:&lt;limite&gt;</code>\n"
            "Exemplo: <code>/invite 30 max:10</code>",
            parse_mode=ParseMode.HTML
        )
        return

    try:
        days = int(args[0])
        limit_str = args[1].split(':')[1]
        if not limit_str.isdigit():
            raise ValueError
        limit = int(limit_str)
    except (ValueError, IndexError):
        await update.message.reply_text("❌ Os valores de dias e limite devem ser números inteiros.")
        return

    # 3. Gera e salva o novo convite
    code = generate_invite_code()
    INVITES[code] = {
        "code": code,
        "days": days,
        "limit": limit,
        "used": 0
    }
    save_invites(INVITES)

    # 4. Envia o link para o admin
    link = f"https://t.me/{BOT_USERNAME}?start={code}"
    mensagem = (
        f"✅ Link de convite gerado com sucesso!\n\n"
        f"<b>Link:</b> <code>{link}</code>\n"
        f"<b>Duração:</b> {days} dias\n"
        f"<b>Limite de usos:</b> {limit}"
    )
    await update.message.reply_text(mensagem, parse_mode=ParseMode.HTML)
#
async def start(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    register_chat(update)
    user_id = update.effective_user.id

    # Verifica se o /start contém um código de convite
    if context.args and len(context.args) > 0:
        invitation_code = context.args[0]
        await handle_invitation(update, context, invitation_code)
        return

    # --- CORREÇÃO: Bloco de permissão removido ---
    # O comando /start agora é aberto a todos. A permissão será
    # verificada dentro de cada comando específico (ex: /search).

    print(f"\033[1;34m   ⟫  USER {user_id} STARTED A BOT\033[m")

    user_name = update.effective_user.username or update.effective_user.first_name

    # Mensagem de boas-vindas (sem alterações)
    mensagem = (
        f"Olá {user_name}, seja bem-vindo!\n\n"
        "<blockquote>Sou o Bot de consultas 𝐆𝐋𝐎𝐑𝐘 𝐋𝐎𝐆𝐒 👁‍🗨!</blockquote>\n"
        f"<i><b>by</b> @{ADMIN_MENTION}</i>\n\n"
        "🔍 Para realizar uma consulta, utilize o comando:\n"
        "<b>/search &lt;sua_busca&gt;</b>\n\n"
        "ℹ️ Utilize os operadores de busca avançada para refinar seus resultados:\n\n"
        "<code>inurl:</code> Busca na URL\n"
        "<code>intext:</code> Busca no usuário e senha\n"
        "<code>site:</code> Busca pelo domínio\n"
        "<code>filetype:</code> Busca por extensão de arquivo\n\n"
        "📌 Exemplo: <code>/search intext:facebook inurl:login site:example.com</code>\n\n"
        "<blockquote>➡️ Use as setas de navegação para ver mais resultados.</blockquote>\n\n"
        "❓ Para ver todos os comandos disponíveis, digite /help\n\n"
    )
    keyboard = [
        [
            InlineKeyboardButton("ᴀᴅᴍɪɴ", url="https://t.me/Prometheust"),
            InlineKeyboardButton("ᴘʟᴀɴᴏꜱ", url="https://t.me/yMusashi"),
        ],
    ]
    reply_markup = InlineKeyboardMarkup(keyboard)

    thread_id = getattr(update.effective_message, "message_thread_id", None)

    with open(START_BANNER, "rb") as photo_file:
        await context.bot.send_photo(
            chat_id=update.effective_chat.id,
            message_thread_id=thread_id,
            photo=photo_file,
            caption=mensagem,
            parse_mode=ParseMode.HTML,
            reply_markup=reply_markup,
        )
#
async def help_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    register_chat(update)
    user_id = update.effective_user.id
    chat_id = update.effective_chat.id


    if (
    update.effective_user.id != ADMIN_USER_ID
    and not is_user_premium(update.effective_user.id)
    and not is_chat_premium(chat_id)
    ):
        await update.message.reply_text(
            f"<blockquote>❌ VOCÊ NÃO TEM PERMISSÃO SUFICIENTE PARA USAR ESSE BOT, ENTRE EM CONTATO COM @{ADMIN_MENTION}</blockquote>",
            parse_mode=ParseMode.HTML
        )
        return

    mensagem = (
        "<blockquote>🎯 <b>Comandos Disponíveis:</b></blockquote>\n\n"
        "🔎 <b>/search</b> &lt;sua_busca&gt; - Realiza uma pesquisa nos logs.\n"
        "ℹ️ <b>/info</b> - Exibe informações sobre a base de dados.\n"
        "🗣️ <b>/profile</b> - Informações sobre seu plano e assinatura.\n\n"
        ""
        "📄 <b>Operadores de Busca Avançada:</b>\n"
        "⁝⁝⁝ <code>inurl:</code> Busca na URL\n"
        "⁝⁝⁝ <code>intext:</code> Busca no usuário e senha\n"
        "⁝⁝⁝ <code>site:</code> Busca pelo domínio\n"
        "⁝⁝⁝ <code>filetype:</code> Busca por extensão de arquivo\n\n"
        "📝 <b>Exemplo:</b> <code>/search intext:\"admin\" site:example.com</code>\n\n"
        "<blockquote>➡️ Use as setas de navegação para ver mais resultados durante a pesquisa.</blockquote>\n\n"
        "<blockquote>⏬🗂️ Faça download do resultado completo da busca.</blockquote>\n\n"
        ""
        # "<i>👤 Qualquer dúvida, entre em contato com @{ADMIN_MENTION}</i>"
    )
    keyboard = [
        # Linha 1: dois botões lado a lado
        [
            InlineKeyboardButton("ᴀᴅᴍɪɴ", url="https://t.me/yMusashi"),
            InlineKeyboardButton("ᴘʟᴀɴᴏꜱ", callback_data="show_plans"),
        ],
        # # Linha 2: botão único (vertical)
        # [InlineKeyboardButton("📞 Suporte", url="https://t.me/SupportChat")],
        # # Linha 3: mistura — pode ter quantas linhas quiser
        # [
        #     InlineKeyboardButton("🆕 Novidades", url="https://blog.example.com"),
        #     InlineKeyboardButton("🔧 Configurações", url="https://example.com/settings"),
        # ],
    ]
    reply_markup = InlineKeyboardMarkup(keyboard)

    # captura o thread_id caso seja um tópico em grupo
    thread_id = getattr(update.effective_message, "message_thread_id", None)

    with open(START_BANNER, "rb") as photo_file:
        await context.bot.send_photo(
            chat_id=update.effective_chat.id,
            message_thread_id=thread_id,
            photo=photo_file,
            caption=mensagem,
            parse_mode=ParseMode.HTML,
            reply_markup=reply_markup, 
        )
#
async def info_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    register_chat(update)
    user_id = update.effective_user.id
    chat_id = update.effective_chat.id

    if (
    update.effective_user.id != ADMIN_USER_ID
    and not is_user_premium(update.effective_user.id)
    and not is_chat_premium(chat_id)
):
        await update.message.reply_text(
            f"<blockquote>❌ VOCÊ NÃO TEM PERMISSÃO SUFICIENTE PARA USAR ESSE BOT, ENTRE EM CONTATO COM @{ADMIN_MENTION}</blockquote>",
            parse_mode=ParseMode.HTML
        )
        return

    # computa estatísticas
    total_files = total_lines = total_valid_entries = total_size = 0
    last_file_time = last_file_name = None

    today = datetime.utcnow().date()
    active_count = sum(
        1
        for rec in USERS.values()
        if rec["premium"] == "y"
        and datetime.strptime(rec["end-date"], "%Y-%m-%d").date() >= today
    )

    for root, _, files in os.walk(LOGS_PATH):
        for file in files:
            if not file.endswith(".txt"):
                continue
            path = os.path.join(root, file)
            total_files += 1

            try:
                with open(path, "r", encoding="utf-8") as f:
                    lines = f.readlines()
            except UnicodeDecodeError:
                try:
                    with open(path, "r", encoding="latin-1") as f:
                        lines = f.readlines()
                except UnicodeDecodeError:
                    continue

            total_lines += len(lines)
            for linha in lines:
                if parse_line(linha):
                    total_valid_entries += 1
            total_size += os.path.getsize(path)

            mtime = os.path.getmtime(path)
            if last_file_time is None or mtime > last_file_time:
                last_file_time = mtime
                last_file_name = file

    last_file_date = (
        datetime.fromtimestamp(last_file_time).strftime('%d/%m/%Y %H:%M:%S')
        if last_file_time else "N/A"
    )
    total_size_formatted = format_size(total_size)

    mensagem = (
        f"<blockquote>📊 <b>Informações da Base de Dados:</b></blockquote>\n\n"
        f""
        f"🗂️ Total de arquivos: <b>{total_files}</b>\n"
        f"📄 Total de linhas: <b>{total_lines}</b>\n"
        f"✅ Entradas válidas (URL:USER:PASS): <b>{total_valid_entries}</b>\n"
        f"💾 Tamanho aproximado da base de dados: <b>{total_size_formatted}</b>\n"
        f"📥 Último arquivo adicionado: <b>{last_file_name or 'N/A'}</b>\n"
        f"🕒 Data de entrada: <b>{last_file_date}</b>\n"
        f"👥 Usuários Ativos: <b>{active_count}</b>\n"
    )

    # respeita o tópico (thread) em grupos
    thread_id = getattr(update.effective_message, "message_thread_id", None)
    await context.bot.send_message(
        chat_id=update.effective_chat.id,
        message_thread_id=thread_id,
        text=mensagem,
        parse_mode=ParseMode.HTML,
    )
################
async def add_user(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    /add:
      - EM GRUPO:   /add <dias> limit:<limite>       → ativa premium para este grupo
      - NO PRIVADO: /add <@user|id> <dias> limit:<limite> → ativa premium para usuário
    Só ADMIN. O parâmetro de limite é opcional.
    """
    if update.effective_user.id != ADMIN_USER_ID:
        await update.message.reply_text("❌ Você não tem permissão para usar este comando.")
        return
    user_id = update.effective_user.id
    chat_id = update.effective_chat.id

    chat = update.effective_chat
    args = context.args or []
    
    print("")
    print(f"\033[1;34m   ⟫  USER/GROUP {user_id} | {chat_id} ADDED TO PREMIUM\033[m")
    
    # Valor padrão do limite de buscas
    daily_limit = 99999  # Padrão "ilimitado" para adições manuais

    # Extrai o limite dos argumentos, se existir
    limit_args = [arg for arg in args if arg.lower().startswith("limit:")]
    if limit_args:
        try:
            daily_limit = int(limit_args[0].split(':')[1])
            args.remove(limit_args[0]) # Remove o argumento de limite do resto do processamento
        except (ValueError, IndexError):
            await update.message.reply_text("❌ Formato de limite inválido. Use `limit:100`.")
            return

    # Processamento para Grupo
    if chat.type in ("group", "supergroup"):
        if len(args) != 1 or not args[0].isdigit():
            await update.message.reply_text("⚠️ Uso no grupo: /add <dias> [limit:<limite>]")
            return
        target_id = chat.id
        days = int(args[0])
    # Processamento para Privado
    else:
        if len(args) < 2 or not args[1].isdigit():
            await update.message.reply_text("⚠️ Uso no privado: /add <@user|id> <dias> [limit:<limite>]")
            return
        target_id = 0 # Inicializa
        days = int(args[1])
        target_user_str = args[0]
        # Resolve o target_id... (lógica existente)
        try:
            chat_obj = await context.bot.get_chat(target_user_str)
            target_id = chat_obj.id
        except Exception:
            await update.message.reply_text(f"❌ Usuário `{target_user_str}` não encontrado.")
            return

    # Datas
    now = datetime.now(timezone.utc)
    start_date = now.date()
    end_date = (now + timedelta(days=days)).date()

    # Grava no CSV com os novos campos
    USERS[target_id] = {
        "user":              str(target_id),
        "registration-date": start_date.isoformat(),
        "end-date":          end_date.isoformat(),
        "premium":           "y",
        "daily_limit":       daily_limit,
        "searches_today":    0,
        "last_search_date":  "",
    }
    save_users(USERS)

    escopo = "grupo" if chat.type in ("group", "supergroup") else "usuário"
    await update.message.reply_text(
        f"✅ Premium ativo para {escopo} `{target_id}` até {end_date.strftime('%d/%m/%Y')}.\n"
        f"Limite de buscas diárias: {'Ilimitado' if daily_limit >= 99999 else daily_limit}.",
        parse_mode=ParseMode.MARKDOWN
    )
#
async def remove_user(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """/remove <user> — remove o premium de um usuário ou grupo."""
    if update.effective_user.id != ADMIN_USER_ID:
        await update.message.reply_text("❌ Você não tem permissão.")
        return

    target_id_to_remove = None
    chat_obj = None # NOVO: Variável para armazenar o objeto do alvo

    # 1) Resolver o alvo via reply ou argumento
    if update.message.reply_to_message:
        target_id_to_remove = update.message.reply_to_message.from_user.id
        # Para replies, assumimos que é um usuário
        chat_obj = update.message.reply_to_message.from_user
    else:
        if not context.args:
            await update.message.reply_text("Uso: /remove <user_id|@username>")
            return
        target_arg = context.args[0]
        try:
            # MODIFICADO: Armazenamos o objeto retornado por get_chat
            chat_obj = await context.bot.get_chat(int(target_arg)) if target_arg.isdigit() else await context.bot.get_chat(target_arg.lstrip("@"))
            target_id_to_remove = chat_obj.id
        except Exception:
            await update.message.reply_text(f"❌ Não encontrei o usuário ou grupo `{target_arg}`.")
            return

    print("")
    print(f"\033[1;31m   ⟫  REMOVING {target_id_to_remove}\033[m")
    
    if target_id_to_remove not in USERS:
        await update.message.reply_text(f"ℹ️ ID `{target_id_to_remove}` não possui uma assinatura ativa.")
        return

    # Remove o alvo do dicionário
    del USERS[target_id_to_remove]
    save_users(USERS)

    # NOVO: Define o escopo (Usuário ou Grupo) para a mensagem de resposta
    escopo = "Grupo" if chat_obj and chat_obj.type in ('group', 'supergroup') else "Usuário"
    
    # MODIFICADO: Usa a variável 'escopo' na mensagem final
    await update.message.reply_text(f"✅ {escopo} `{target_id_to_remove}` removido da base de assinantes.")#
#
async def admin_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """Exibe um painel de ajuda completo exclusivo para o administrador."""
    # Etapa 1: Garante que apenas o administrador pode usar este comando
    if update.effective_user.id != ADMIN_USER_ID:
        return

    # Etapa 2: Monta a mensagem de ajuda detalhada com formatação HTML
    admin_help_text = (
        "<blockquote>👨‍💻 Painel de Controle do Administrador</blockquote>\n\n"
        ""
        "Bem-vindo ao seu painel de controle. Aqui está um guia rápido sobre como gerenciar o bot:\n\n"
        "<b>⁝⁝⁝ 👤 Gerenciamento de Acessos ---</b>\n\n"
        "<code>/add &lt;ID|@user&gt; &lt;dias&gt; [limit:N]</code>\n"
        "↳ Adiciona um <b>usuário</b> premium. O limite de buscas diárias é opcional (padrão: ilimitado).\n"
        "<i>Ex: /add 123456 30 limit:100</i>\n\n"
        
        "<code>/add &lt;dias&gt; [limit:N]</code> (usado dentro de um grupo)\n"
        "↳ Adiciona o <b>grupo</b> inteiro como premium. O limite também é opcional.\n\n"

        "<code>/remove &lt;ID|@user&gt;</code>\n"
        "↳ Remove o acesso premium de um usuário ou grupo. Você também pode responder a uma mensagem do usuário com /remove.\n\n"

        "<blockquote>⁝⁝⁝ 🎟️ Sistema de Convites ---</blockquote>\n\n"
        "<code>/invite &lt;dias&gt; max:&lt;usos&gt;</code>\n"
        "↳ Gera um link de convite único. Novos usuários que usarem o link receberão premium com duração definida e limite de buscas padrão (15/dia).\n"
        "<i>Ex: /invite 7 max:20</i> (cria um link para 20 pessoas com 7 dias de premium).\n\n"

        "<blockquote>⁝⁝⁝ 📢 Comunicação em Massa ---</blockquote>\n\n"
        "<code>/all</code> (respondendo a uma mensagem)\n"
        "↳ Envia a mensagem respondida para <b>todos os chats</b> onde o bot já foi iniciado. Use para anúncios importantes.\n\n"

        "<blockquote>⁝⁝⁝ 🔍 Comandos Gerais (Visão do Usuário) ---</blockquote>\n\n"
        "• <b>/search &lt;termo&gt;</b>: Realiza buscas na base de dados.\n"
        "• <b>/profile</b>: Exibe o status da assinatura e o uso diário de buscas.\n"
        "• <b>/info</b>: Mostra estatísticas da base de dados.\n"
        "• <b>/help</b>: Guia de comandos para o usuário comum.\n\n"
        
        "<i>Dica: Usar o ID numérico do usuário/grupo é sempre mais confiável do que o @username.</i>"
    )

    # Etapa 3: Envia a mensagem para o administrador
    await update.message.reply_text(admin_help_text, parse_mode=ParseMode.HTML)
#
async def enviar_pagina_free(
    context: ContextTypes.DEFAULT_TYPE,
    user_id: int,
    total_original: int
) -> None:
    """
    Formata e edita a mensagem de resultados para usuários do plano FREE.
    Exibe no máximo 15 resultados e não tem botões de paginação.
    """
    estado     = userbot[user_id]
    chat_id    = estado["chat_id"]
    message_id = estado["message_id"]
    resultados = estado["resultados"] # Já está limitado a 15

    # Monta a mensagem com o cabeçalho específico do plano FREE
    resposta = (
        f"🔎 | SUA PESQUISA RETORNOU {total_original} RESULTADOS;\n"
        f"📌 | EXIBINDO {len(resultados)}/{total_original} DO PLANO FREE:\n\n"
    )
    for linha in resultados:
        partes = parse_line(linha)
        if partes:
            url, user, senha = partes
            resposta += (
                f"🧭: <code>{html.escape(url)}</code>\n"
                f"👤: <code>{html.escape(user)}</code>\n"
                f"🔑: <code>{html.escape(senha)}</code>\n-\n"
            )
        else:
            resposta += f"{html.escape(linha)}\n-\n"

    # Botões para o plano FREE (Download removido, incentivo para upgrade)
    keyboard = [[InlineKeyboardButton("✨ Fazer Upgrade para Premium", callback_data="show_plans")]]
    reply_markup = InlineKeyboardMarkup(keyboard)

    try:
        await context.bot.edit_message_text(
            chat_id=chat_id,
            message_id=message_id,
            text=resposta,
            reply_markup=reply_markup,
            parse_mode=ParseMode.HTML,
        )
    except BadRequest as e:
        if "Message is not modified" in str(e):
            pass
        else:
            print(f"Erro ao editar mensagem (free): {e}")
##################
async def searchlogs(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """Handler do /search que diferencia a lógica para planos Premium e Free."""
    register_chat(update)
    user_id = update.effective_user.id
    chat_id = update.effective_chat.id
    id_to_check = chat_id if update.effective_chat.type in ('group', 'supergroup') else user_id

    # Validação do termo de busca (feita antes para todos)
    search_query = " ".join(context.args).strip()
    if not search_query:
        await update.message.reply_text("✅ Forneça um termo de pesquisa.")
        return

    # --- LÓGICA DE CONTROLE DE ACESSO E LIMITE ---
    is_premium = is_user_premium(id_to_check)

    if not check_and_reset_search_limit(id_to_check):
        user_rec = USERS.get(id_to_check, {})
        limit = user_rec.get('daily_limit', 3)
        keyboard = [[InlineKeyboardButton("✨ Fazer Upgrade de Plano", url="https://t.me/yMusashi")]]
        await update.message.reply_text(
            f"❌ Você atingiu seu limite de {limit} buscas diárias. Tente novamente amanhã ou faça um upgrade.",
            reply_markup=InlineKeyboardMarkup(keyboard)
        )
        return

    # Envia mensagem "Pesquisando..."
    loading_message = await context.bot.send_message(
        chat_id=chat_id,
        text="🔍 Pesquisando, seja paciente..."
    )

    # --- LÓGICA DE BUSCA (comum a todos) ---
    parsed_criteria = parse_search_query(search_query)
    resultados = []
    print(f"\033[1;34m   ⟫  USER {user_id} (Premium: {is_premium}) SEARCHED FOR '{search_query}'\033[m")
    for root, _, files in os.walk(LOGS_PATH):
        for file in files:
            if not file.endswith(".txt"):
                continue
            path = os.path.join(root, file)
            try:
                with open(path, "r", encoding="utf-8", errors="ignore") as f:
                    for linha in f:
                        partes = parse_line(linha)
                        if partes and line_matches_criteria(*partes, parsed_criteria):
                            resultados.append(linha.strip())
            except Exception as e:
                print(f"Erro ao processar o arquivo {path}: {e}")
                continue

    # Incrementa a contagem de buscas do dia
    if USERS.get(id_to_check):
        USERS[id_to_check]["searches_today"] += 1
        # Para usuários free, não salvamos no CSV, a alteração fica só na memória
        if is_premium:
            save_users(USERS)

    # --- LÓGICA DE RESPOSTA (diferenciada por plano) ---
    total_original = len(resultados)
    if total_original == 0:
        await context.bot.edit_message_text(
            chat_id=loading_message.chat_id,
            message_id=loading_message.message_id,
            text=f"❌ Nenhum resultado encontrado para: {search_query}"
        )
        return

    thread_id = getattr(update.effective_message, "message_thread_id", None)
    
    if is_premium:
        # Lógica para usuários PREMIUM
        userbot[user_id] = {
            "termo": search_query, "offset": 0, "resultados": resultados,
            "message_id": loading_message.message_id, "chat_id": chat_id, "thread_id": thread_id,
        }
        await enviar_pagina(update, context, user_id)
    else:
        # Lógica para usuários FREE
        resultados_free = resultados[:15] # Limita a 15 resultados
        userbot[user_id] = {
            "termo": search_query, "offset": 0, "resultados": resultados_free,
            "message_id": loading_message.message_id, "chat_id": chat_id, "thread_id": thread_id,
        }
        await enviar_pagina_free(context, user_id, total_original)#
# 
async def enviar_pagina(
    update: Update,
    context: ContextTypes.DEFAULT_TYPE,
    user_id: int
) -> None:
    """
    Edita a mensagem de resultados.
    """
    estado     = userbot[user_id]
    chat_id    = estado["chat_id"]
    message_id = estado["message_id"]
    # Pega o ID diretamente do estado
    resultados = estado["resultados"]
    offset     = estado["offset"]
    total      = len(resultados)

    # Lógica para montar a mensagem de resposta (existente)
    total_pages  = (len(estado["resultados"]) + 29) // 30
    current_page = estado["offset"] // 30 + 1
    fim          = min(estado["offset"] + 30, len(estado["resultados"]))
    mostrados    = estado["resultados"][estado["offset"]:fim]

    resposta = (
        f"<blockquote>🔎 | SUA PESQUISA RETORNOU {total} RESULTADOS, EXIBINDO ({current_page}/{total_pages}):</blockquote>\n\n"
    )
    for linha in mostrados:
        partes = parse_line(linha)
        if partes:
            url, user, senha = partes
            resposta += (
                f"🧭: <code>{html.escape(url)}</code>\n"
                f"👤: <code>{html.escape(user)}</code>\n"
                f"🔑: <code>{html.escape(senha)}</code>\n-\n"
            )
        else:
            resposta += f"{html.escape(linha)}\n-\n"

    # Monta navegação + download (lógica existente)
    keyboard_buttons = []
    nav_row = []
    if current_page > 1:
        nav_row.append(InlineKeyboardButton("⬅ --- ᴘʀᴇᴠ", callback_data="prev"))
    if current_page < total_pages:
        nav_row.append(InlineKeyboardButton("ɴᴇxᴛ --- ➡", callback_data="next"))
    
    if nav_row:
        keyboard_buttons.append(nav_row)
    
    keyboard_buttons.append([InlineKeyboardButton("⏬🗂️", callback_data="download")])
    reply_markup = InlineKeyboardMarkup(keyboard_buttons)

    # --- LÓGICA DE EDIÇÃO CORRIGIDA ---
    try:
        await context.bot.edit_message_text(
            chat_id=chat_id,
            message_id=message_id,
            text=resposta,
            reply_markup=reply_markup,
            parse_mode=ParseMode.HTML,
        )
    except BadRequest as e:
        if "Message is not modified" in str(e):
            pass
        else:
            print(f"Erro ao editar mensagem: {e}")
#
async def gerar_arquivo_resultados(
    update: Update,
    context: ContextTypes.DEFAULT_TYPE,
    user_id: int
) -> None:
    """
    Gera e envia o arquivo de resultados completo no mesmo tópico (thread)
    em que o usuário executou a busca, com um nome de arquivo sanitizado.
    """
    estado     = userbot[user_id]
    chat_id    = estado["chat_id"]
    thread_id  = estado.get("thread_id")
    resultados = estado["resultados"]
    termo      = estado["termo"]
    user_name  = update.effective_user.username or update.effective_user.first_name

    # Monta o conteúdo do arquivo (lógica existente)
    content  = (
        f"Resultados obtidos para ~{termo}~, pelo bot https://t.me/{BOT_USERNAME}\n"
        f"by t.me/{ADMIN_MENTION}\n\n"
    )
    content += f"Usuário que fez a busca: @{user_name}\n\n"
    content += "-" * 50 + "\n"
    for linha in resultados:
        partes = parse_line(linha)
        if partes:
            url, usr, pwd = partes
            content += f"{url}\n{usr}\n{pwd}\n-\n"
    content += "-" * 50 + "\n"
    content += f"Fim da consulta, continue em t.me/{BOT_USERNAME}\n"

    # Cria o arquivo em memória
    file_obj = StringIO()
    file_obj.write(content)
    file_obj.seek(0)

    # --- INÍCIO DA MODIFICAÇÃO ---

    # NOVO: Etapa 1 - Sanitizar o termo da busca para usar como nome de arquivo
    # Define uma lista de caracteres inválidos em nomes de arquivo
    invalid_chars = r'<>:"/\|?*'
    # Substitui cada caractere inválido no termo por um underscore '_'
    sanitized_termo = termo
    for char in invalid_chars:
        sanitized_termo = sanitized_termo.replace(char, '_')

    # NOVO: Etapa 2 - Cria o novo nome de arquivo dinâmico
    filename = f"{sanitized_termo}-@{BOT_USERNAME}.txt"

    # --- FIM DA MODIFICAÇÃO ---

    # Envia o documento no mesmo tópico/chat com o novo nome de arquivo
    await context.bot.send_document(
        chat_id=chat_id,
        message_thread_id=thread_id,
        # Usa a variável 'filename' que acabamos de criar
        document=InputFile(file_obj, filename=filename)
    )
###################
async def broadcast_all(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    /all — responda a uma mensagem com /all seguido das definições
    de botões em markdown; copia a mensagem original INTEIRA (preservando
    formatação) para todos os chats registrados, adicionando apenas os botões.
    """
    # só admin
    if update.effective_user.id != ADMIN_USER_ID:
        return

    # precisa ser reply
    if not update.message.reply_to_message:
        await update.message.reply_text(
            "❗️ Use respondendo à mensagem que deseja enviar e inclua"
            " abaixo, após /all, os botões em markdown."
        )
        return

    # monta o reply_markup a partir do próprio texto do comando
    reply_markup = parse_all_buttons(update.message.text or "")

    print("")
    print(f"\033[1;34m   ⟫  SENT MARKETING SHOT 💸\033[m")

    msg = update.message.reply_to_message
    sent = 0
    for cid in list(CHATS):
        try:
            await context.bot.copy_message(
                chat_id=cid,
                from_chat_id=msg.chat.id,
                message_id=msg.message_id,
                reply_markup=reply_markup
            )
            sent += 1
        except Exception:
            continue

    await update.message.reply_text(f"📤 Mensagem enviada para {sent} chats.")
#
async def profile_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    register_chat(update)
    user_id = update.effective_user.id
    chat_id = update.effective_chat.id
    user_name = update.effective_user.username or update.effective_user.first_name

    # Garante que a contagem de buscas do dia esteja atualizada
    check_and_reset_search_limit(user_id)
    rec = USERS.get(user_id, {})

    # --- LÓGICA DE PERFIL DINÂMICO ---
    if is_user_premium(user_id):
        # Lógica para usuários PREMIUM
        today = datetime.now(timezone.utc).date()
        end_date = datetime.fromisoformat(rec["end-date"]).date()
        days_left = max((end_date - today).days, 0)
        premium_status = "Premium ✨"
        expiration_str = end_date.strftime("%d/%m/%Y")
        searches_today = rec.get("searches_today", 0)
        daily_limit = rec.get("daily_limit", 0)
        limit_str = 'Ilimitado' if daily_limit >= 99999 else daily_limit
        results_limit_str = "Todos os resultados"

        mensagem = (
            f"<blockquote>Olá {user_name}, essa é sua conta:</blockquote>\n\n"
            f"⁝⁝⁝ ID: <code>{user_id}</code>\n"
            f"⁝⁝⁝ Plano: <b>{premium_status}</b>\n"
            f"⁝⁝⁝ Dias restantes: {days_left} dias\n"
            f"⁝⁝⁝ Data de expiração: {expiration_str}\n\n"
            f"⁝⁝⁝ Buscas hoje: {searches_today} / {limit_str}\n"
            f"⁝⁝⁝ Resultados por busca: {results_limit_str}\n\n"
            f"<blockquote>Use /help para saber como usar o bot</blockquote>"
        )
        keyboard = [[InlineKeyboardButton("Suporte com Admin", url="https://t.me/yMusashi")]]

    else:
        # Lógica para usuários FREE
        premium_status = "Gratuito 🆓"
        searches_today = rec.get("searches_today", 0)
        daily_limit = rec.get("daily_limit", 3)
        results_limit_str = "15 por busca"

        mensagem = (
            f"<blockquote>Olá {user_name}, essa é sua conta:</blockquote>\n\n"
            f"⁝⁝⁝ ID: <code>{user_id}</code>\n"
            f"⁝⁝⁝ Plano: <b>{premium_status}</b>\n\n"
            f"⁝⁝⁝ Buscas hoje: {searches_today} / {daily_limit}\n"
            f"⁝⁝⁝ Resultados por busca: {results_limit_str}\n\n"
            "<blockquote>Faça upgrade para buscas e resultados ilimitados!</blockquote>"
        )
        keyboard = [[InlineKeyboardButton("✨ Conhecer Planos Premium", callback_data="show_plans")]]

    # --- FIM DA LÓGICA DINÂMICA ---

    reply_markup = InlineKeyboardMarkup(keyboard)
    thread_id = getattr(update.effective_message, "message_thread_id", None)

    with open(PROFILE_BANNER, "rb") as photo_file:
        await context.bot.send_photo(
            chat_id=chat_id,
            message_thread_id=thread_id,
            photo=photo_file,
            caption=mensagem,
            parse_mode=ParseMode.HTML,
            reply_markup=reply_markup
        )
#
async def plans_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """Envia uma nova mensagem com a lista de planos de assinatura."""
    message_text, reply_markup = get_plans_message_and_keyboard()
    
    await update.message.reply_text(
        text=message_text,
        reply_markup=reply_markup,
        parse_mode=ParseMode.HTML
    )