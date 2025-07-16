# bot/__init__.py

""" 

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
└── .env                 \# Variáveis de ambiente (TOKEN, ADMIN_ID, etc.)

"""