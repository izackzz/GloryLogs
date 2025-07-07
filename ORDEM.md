# ORDEM BOT GLORY TO GOLANG


<!-- 
1.  **Configurações (`internal/config/config.go`)**
    * **O quê:** Traduzir a lógica de `bot/config.py`.
    * **Por quê:** É o alicerce. Precisamos carregar o token do bot e outras variáveis de ambiente do `.env` antes de qualquer outra coisa.
 -->
<!-- 2.  **Armazenamento (`internal/storage/storage.go`)**
    * **O quê:** Traduzir as funções de manipulação de arquivos de `bot/helpers.py` (como `load_users`, `save_users`, `load_chats`, `load_invites`, etc.).
    * **Por quê:** Esta camada de dados é a dependência principal para quase todos os comandos. Ter a leitura e escrita dos "bancos de dados" em CSV funcionando isoladamente facilitará muito a próxima etapa. -->

<!-- 3.  **Motor de Busca (`internal/search/engine.go`)**
    * **O quê:** Migrar a lógica de pesquisa de `bot/helpers.py` (funções como `parse_search_query`, `line_matches_criteria`).
    * **Por quê:** O coração do bot é a busca. Isolar e traduzir essa lógica complexa garante que a funcionalidade mais crítica funcione bem antes de integrá-la aos comandos do Telegram. -->

4.  **Lógica do Bot e Handlers (`internal/bot/bot.go` e `internal/bot/handlers.go`)**
    * **O quê:** Traduzir o conteúdo de `bot/commands.py` e `bot/callbacks.py`.
    * **Por quê:** Com as configurações, o armazenamento e a busca prontos, agora podemos focar em como o bot interage com o usuário. O arquivo `bot.go` cuidará da inicialização do bot e `handlers.go` conterá a tradução direta dos comandos (`/start`, `/search`, etc.) e das ações dos botões.

5.  **Ponto de Entrada (`cmd/glorylogs-bot/main.go`)**
    * **O quê:** Traduzir a lógica de `bot/start.py`.
    * **Por quê:** Este é o último passo. O arquivo `main.go` irá "amarrar" todas as partes que criamos: carregar a configuração, iniciar a conexão com o Telegram, registrar os *handlers* (comandos) e colocar o bot para rodar.

Sugiro começarmos pelo **passo 1: `internal/config/config.go`**. Quando estiver pronto, me avise para prosseguirmos.
