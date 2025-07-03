from bot.imports import *
from bot.config import *
from bot.helpers import *
from bot.commands import *

# ---------- FUNÇÕES DE CALLBACKS DO BOT ---------- #


async def callback_query_handler(
    update: Update, context: ContextTypes.DEFAULT_TYPE
) -> None:
    query = update.callback_query
    user_id = query.from_user.id
    chat_id = query.message.chat.id

    if (
    update.effective_user.id != ADMIN_USER_ID
    and not is_user_premium(update.effective_user.id)
    and not is_chat_premium(chat_id)
):
        await query.answer("Você não tem permissão para usar este bot.")
        return

    if user_id not in userbot:
        await query.answer("Nenhuma pesquisa em andamento.")
        return

    estado = userbot[user_id]
    data = query.data

    if data == "next":
        estado["offset"] += 30
    elif data == "prev":
        estado["offset"] -= 30
    elif data == "download":
        await gerar_arquivo_resultados(update, context, user_id)
    await enviar_pagina(update, context, user_id)
    await query.answer()


