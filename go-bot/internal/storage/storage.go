package storage

import (
	"database/sql"
	"fmt"

	// "log"
	"os"
	// "time"

	"github.com/fatih/color"
	_ "github.com/mattn/go-sqlite3" // Importa o driver do SQLite

	"glorylogs-bot/internal/config"
)

// User representa a estrutura de dados de um usuário (corresponde à tabela 'users')
type User struct {
	ID               int64
	RegistrationDate string
	EndDate          string
	Premium          string // "y" ou "n"
	DailyLimit       int
	SearchesToday    int
	LastSearchDate   string
}

// Invite representa a estrutura de um convite (corresponde à tabela 'invites')
type Invite struct {
	Code  string
	Days  int
	Limit int
	Used  int
}

// Storage agora gerencia a conexão com o banco de dados.
type Storage struct {
	DB *sql.DB
}

// NewStorage inicializa a conexão com o banco de dados e cria as tabelas se não existirem.
func NewStorage() (*Storage, error) {
	// Garante que o diretório db/ exista
	if err := os.MkdirAll("db", 0755); err != nil {
		return nil, fmt.Errorf("falha ao criar diretório db: %w", err)
	}

	db, err := sql.Open("sqlite3", config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir o banco de dados: %w", err)
	}

	storage := &Storage{DB: db}
	if err := storage.initTables(); err != nil {
		return nil, fmt.Errorf("falha ao inicializar tabelas: %w", err)
	}

	color.Green("   ⟫  CONEXÃO COM BANCO DE DADOS SQLITE ESTABELECIDA COM SUCESSO")
	return storage, nil
}

// initTables cria as tabelas necessárias no banco de dados.
func (s *Storage) initTables() error {
	// Tabela de Usuários
	usersTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY,
		registration_date TEXT,
		end_date TEXT,
		premium TEXT,
		daily_limit INTEGER,
		searches_today INTEGER,
		last_search_date TEXT
	);`
	if _, err := s.DB.Exec(usersTableSQL); err != nil {
		return err
	}

	// Tabela de Convites
	invitesTableSQL := `
	CREATE TABLE IF NOT EXISTS invites (
		code TEXT PRIMARY KEY,
		days INTEGER,
		max_uses INTEGER,
		used_count INTEGER
	);`
	if _, err := s.DB.Exec(invitesTableSQL); err != nil {
		return err
	}

	// Tabela de Chats para Broadcast
	chatsTableSQL := `CREATE TABLE IF NOT EXISTS chats (id INTEGER PRIMARY KEY);`
	if _, err := s.DB.Exec(chatsTableSQL); err != nil {
		return err
	}

	// --- NOVA TABELA ---
	// Tabela de Configurações para armazenar o ID do canal do terminal e outras futuras configs.
	settingsTableSQL := `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);`
	_, err := s.DB.Exec(settingsTableSQL)
	return err
}

// Close fecha a conexão com o banco de dados.
func (s *Storage) Close() {
	s.DB.Close()
}

// --- MÉTODOS PARA GERENCIAR USUÁRIOS ---

func (s *Storage) GetUser(userID int64) (*User, error) {
	user := &User{}
	query := `SELECT id, registration_date, end_date, premium, daily_limit, searches_today, last_search_date FROM users WHERE id = ?`

	row := s.DB.QueryRow(query, userID)
	err := row.Scan(&user.ID, &user.RegistrationDate, &user.EndDate, &user.Premium, &user.DailyLimit, &user.SearchesToday, &user.LastSearchDate)

	if err == sql.ErrNoRows {
		return nil, nil // Retorna nil se o usuário não for encontrado, não é um erro
	}
	return user, err
}

func (s *Storage) AddOrUpdateUser(user *User) error {
	query := `
	INSERT INTO users (id, registration_date, end_date, premium, daily_limit, searches_today, last_search_date)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		registration_date = excluded.registration_date,
		end_date = excluded.end_date,
		premium = excluded.premium,
		daily_limit = excluded.daily_limit,
		searches_today = excluded.searches_today,
		last_search_date = excluded.last_search_date;
	`
	_, err := s.DB.Exec(query, user.ID, user.RegistrationDate, user.EndDate, user.Premium, user.DailyLimit, user.SearchesToday, user.LastSearchDate)
	return err
}

func (s *Storage) RemoveUser(userID int64) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := s.DB.Exec(query, userID)
	return err
}

func (s *Storage) GetAllUsers() ([]*User, error) {
	query := `SELECT id, registration_date, end_date, premium, daily_limit, searches_today, last_search_date FROM users ORDER BY id`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.ID, &user.RegistrationDate, &user.EndDate, &user.Premium, &user.DailyLimit, &user.SearchesToday, &user.LastSearchDate)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// --- MÉTODOS PARA GERENCIAR CONVITES ---

func (s *Storage) GetInvite(code string) (*Invite, error) {
	invite := &Invite{}
	query := `SELECT code, days, max_uses, used_count FROM invites WHERE code = ?`

	row := s.DB.QueryRow(query, code)
	err := row.Scan(&invite.Code, &invite.Days, &invite.Limit, &invite.Used)

	if err == sql.ErrNoRows {
		return nil, nil // Convite não encontrado
	}
	return invite, err
}

func (s *Storage) AddInvite(invite *Invite) error {
	query := `INSERT INTO invites (code, days, max_uses, used_count) VALUES (?, ?, ?, ?)`
	_, err := s.DB.Exec(query, invite.Code, invite.Days, invite.Limit, invite.Used)
	return err
}

func (s *Storage) IncrementInviteUsage(code string) error {
	query := `UPDATE invites SET used_count = used_count + 1 WHERE code = ?`
	_, err := s.DB.Exec(query, code)
	return err
}

func (s *Storage) GetAllInvites() ([]*Invite, error) {
	query := `SELECT code, days, max_uses, used_count FROM invites ORDER BY code`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []*Invite
	for rows.Next() {
		invite := &Invite{}
		err := rows.Scan(&invite.Code, &invite.Days, &invite.Limit, &invite.Used)
		if err != nil {
			return nil, err
		}
		invites = append(invites, invite)
	}
	return invites, nil
}

// --- MÉTODOS PARA GERENCIAR CHATS (BROADCAST) ---

func (s *Storage) AddChat(chatID int64) error {
	// ON CONFLICT DO NOTHING ignora o erro se o chatID já existir, que é o que queremos.
	query := `INSERT INTO chats (id) VALUES (?) ON CONFLICT(id) DO NOTHING`
	_, err := s.DB.Exec(query, chatID)
	return err
}

func (s *Storage) GetAllChatIDs() ([]int64, error) {
	query := `SELECT id FROM chats`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chatIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		chatIDs = append(chatIDs, id)
	}
	return chatIDs, nil
}

func (s *Storage) SetSetting(key, value string) error {
	query := `INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value;`
	_, err := s.DB.Exec(query, key, value)
	return err
}

func (s *Storage) GetSetting(key string) (string, error) {
	var value string
	query := `SELECT value FROM settings WHERE key = ?`
	err := s.DB.QueryRow(query, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil // Chave não encontrada, retorna string vazia sem erro
	}
	return value, err
}
