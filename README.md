
# 🤖 Glory Logs Bot 👁‍🗨

Um bot de Telegram multifuncional escrito em Python usando a biblioteca `python-telegram-bot`. Ele serve como um motor de busca privado para uma base de dados local de arquivos de texto, com um sistema de gerenciamento de usuários premium, planos de assinatura, convites e ferramentas administrativas.

## ✨ Funcionalidades Principais

- **Sistema de Planos**: Suporte para usuários **Premium** e **Gratuitos** com diferentes níveis de acesso.
- **Limites Configuráveis**:
  - Usuários Premium têm limites de busca diária customizáveis.
  - Usuários Gratuitos têm um limite padrão de 3 buscas diárias e 15 resultados por busca.
- **Busca Avançada**: Motor de busca que varre arquivos `.txt` locais com suporte a operadores como `inurl:`, `intext:`, `site:` e `filetype:`.
- **Interface Rica**:
  - Respostas interativas com botões e paginação (`Próximo`/`Anterior`).
  - Edição de mensagens para uma experiência fluida (ex: "Pesquisando..." -> Resultados).
  - Páginas internas para comandos como `/plans`.
- **Ferramentas de Administrador**:
  - Painel de ajuda exclusivo com o comando `/admin`.
  - Adicionar (`/add`) e remover (`/remove`) assinaturas de usuários e grupos.
  - Gerar links de convite (`/invite`) com duração e limite de usos customizáveis.
  - Enviar mensagens em massa (`/all`) para todos os usuários do bot.
- **Perfis de Usuário**: Comando `/profile` que mostra o status da assinatura e os limites de uso.

## 📁 Estrutura do Projeto

```

GloryLogs/
├── bot/
│   ├── **init**.py
│   ├── start.py         \# Ponto de entrada da aplicação
│   ├── commands.py      \# Lógica dos comandos (/start, /search, etc.)
│   ├── callbacks.py     \# Lógica dos botões (inline keyboards)
│   ├── config.py        \# Configurações (TOKEN, ADMIN\_ID, etc.)
│   ├── helpers.py       \# Funções utilitárias (gerenciar CSV, buscas, etc.)
│   └── imports.py       \# Central de importações de bibliotecas
├── db/
│   ├── users.csv        \# (será criado automaticamente)
│   ├── chats.txt        \# (será criado automaticamente)
│   └── invites.csv      \# (será criado automaticamente)
├── logs/
│   └── (coloque seus arquivos .txt de logs aqui)
├── bg/
│   ├── bg.png           \# Banner do /start e /help
│   └── mkt.jpg          \# Banner do /profile
├── requirements.txt     \# Dependências do projeto
└── README.md            \# Este arquivo

```

## 🚀 Como Configurar e Iniciar

Siga estes passos para configurar e rodar o bot em sua máquina.

### 1. Pré-requisitos

- **Python 3.10** ou superior instalado.
- **Git** instalado.

### 2. Criando o `requirements.txt`

Antes de instalar, você precisa listar as dependências do seu projeto. Se ainda não tem este arquivo, crie-o na raiz do projeto com o seguinte conteúdo:

**`requirements.txt`**:
```

python-telegram-bot==21.3

```


### 3. Passos para Instalação

**a. Clone o Repositório**
```bash
git clone https://github.com/izackzz/GloryLogs.git
cd /GloryLogs>
````

**b. Crie e Ative um Ambiente Virtual (Recomendado)**

Isso isola as dependências do seu projeto.

  - No Windows:
    ```bash
    python -m venv venv
    .\venv\Scripts\activate
    ```
  - No macOS/Linux:
    ```bash
    python3 -m venv venv
    source venv/bin/activate
    ```

**c. Instale as Dependências**

```bash
pip install -r requirements.txt
```

### 4\. Configurando as Variáveis

Esta é a etapa mais importante. Abra o arquivo `bot/config.py` e edite as seguintes variáveis:

  - **`BOT_TOKEN`**: Substitua o token de exemplo pelo token real do seu bot. Você pode obter um com o [@BotFather](https://t.me/BotFather) no Telegram.

    ```python
    BOT_TOKEN = "7962833687:AAGv8E6p9gC2MSjpRHukV7SeMRiCR6xiaRM" # <-- COLOQUE SEU TOKEN AQUI
    ```

  - **`ADMIN_USER_ID`**: Coloque o seu ID de usuário do Telegram aqui. Para descobrir seu ID, envie `/start` para o bot [@userinfobot](https://t.me/userinfobot).

    ```python
    ADMIN_USER_ID = 5486349822 # <-- COLOQUE SEU ID DE ADMIN AQUI
    ```

  - **`BOT_USERNAME`**: Certifique-se de que este valor corresponde exatamente ao nome de usuário do seu bot (sem o `@`). Isso é crucial para que os links de convite funcionem.

    ```python
    BOT_USERNAME = "GloryLogsBot" # <-- COLOQUE O USERNAME DO SEU BOT AQUI
    ```

### 5\. Preparando os Dados

  - Certifique-se de que a pasta `logs/` existe na raiz do projeto e coloque seus arquivos `.txt` dentro dela.
  - A pasta `db/` será criada e gerenciada automaticamente pelo bot.
  - Coloque os banners `bg.png` e `mkt.jpg` na pasta `bg/`.

### 6\. Executando o Bot

Com o ambiente virtual ativado e as variáveis configuradas, execute o seguinte comando a partir da pasta raiz do projeto (`GloryLogs/`):

```bash
python -m bot.start
```

Se tudo estiver configurado corretamente, você verá uma mensagem no terminal indicando que o bot está online.

-----

## 📖 Comandos Disponíveis

### Comandos de Administrador (`/admin`)

  - **`/add <ID|@user> <dias> [limit:N]`**: Ativa premium para um usuário.
  - **`/add <dias> [limit:N]`**: Ativa premium para o grupo onde o comando foi enviado.
  - **`/remove <ID|@user>`**: Remove o acesso premium de um usuário ou grupo.
  - **`/invite <dias> max:<usos>`**: Gera um link de convite para novos usuários.
  - **`/all` (respondendo a uma mensagem)**: Envia uma mensagem para todos os usuários do bot.

### Comandos para Usuários

  - **`/start`**: Inicia o bot e mostra a mensagem de boas-vindas.
  - **`/help`**: Mostra os comandos disponíveis e operadores de busca.
  - **`/plans`**: Exibe os planos de assinatura premium.
  - **`/search <termo>`**: Realiza uma busca na base de dados.
  - **`/profile`**: Exibe o status da sua assinatura (Premium ou Gratuito) e seus limites de uso.
  - **`/info`**: Mostra estatísticas sobre a base de dados do bot.

<!-- end list -->
