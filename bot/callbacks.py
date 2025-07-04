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
    data = query.data

    # --- INÍCIO DA NOVA LÓGICA ---
    if data == "show_plans":
        message_text, reply_markup = get_plans_message_and_keyboard()
        try:
            # Tenta editar uma mensagem que tem foto (como /start e /profile)
            await query.edit_message_caption(
                caption=message_text,
                reply_markup=reply_markup,
                parse_mode=ParseMode.HTML
            )
        except BadRequest as e:
            if "message is not modified" in str(e).lower():
                pass # Ignora o erro se a mensagem já for a de planos
            else:
                try:
                    # Se falhar, tenta editar como uma mensagem de texto normal
                    await query.edit_message_text(
                        text=message_text,
                        reply_markup=reply_markup,
                        parse_mode=ParseMode.HTML
                    )
                except BadRequest as e_text:
                     print(f"Erro final ao tentar editar para mostrar planos: {e_text}")
        await query.answer()
        return
    # --- FIM DA NOVA LÓGICA ---
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

        # Etapa 1: Trata o download como um caso especial e final
    if data == "download":
        # Gera e envia o arquivo de resultados
        await gerar_arquivo_resultados(update, context, user_id)
        # Apenas responde ao callback para remover o "carregando" do botão
        await query.answer("Arquivo de resultados enviado!")
        # Retorna para não executar o código de paginação abaixo
        return

    # Etapa 2: Se não for download, processa a paginação
    if data == "next":
        estado["offset"] += 30
    elif data == "prev":
        estado["offset"] -= 30
    
    # Etapa 3: Atualiza a página (só será executado para 'next' e 'prev')
    await enviar_pagina(update, context, user_id)
    await query.answer()


