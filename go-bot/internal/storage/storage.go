package storage

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/fatih/color"

	"glorylogs-bot/internal/config"
)

// User representa a estrutura de dados de um usuário no arquivo users.csv.
type User struct {
	ID               int64
	RegistrationDate string
	EndDate          string
	Premium          string
	DailyLimit       int
	SearchesToday    int
	LastSearchDate   string
}

// Invite representa a estrutura de dados de um convite no arquivo invites.csv.
type Invite struct {
	Code  string
	Days  int
	Limit int
	Used  int
}

// Storage gerencia todos os dados da aplicação carregados em memória.
// Ele usa mutex para garantir o acesso seguro aos dados em ambientes concorrentes.
type Storage struct {
	mu      sync.RWMutex
	Users   map[int64]*User
	Invites map[string]*Invite
	Chats   map[int64]bool
}

// NewStorage cria e inicializa uma nova instância de Storage, carregando todos os dados.
func NewStorage() *Storage {
	s := &Storage{
		Users:   make(map[int64]*User),
		Invites: make(map[string]*Invite),
		Chats:   make(map[int64]bool),
	}
	s.loadUsers()
	s.loadInvites()
	s.loadChats()
	return s
}

// Lock bloqueia o mutex para escrita.
func (s *Storage) Lock() {
	s.mu.Lock()
}

// Unlock desbloqueia o mutex de escrita.
func (s *Storage) Unlock() {
	s.mu.Unlock()
}

// RLock bloqueia o mutex para leitura.
func (s *Storage) RLock() {
	s.mu.RLock()
}

// RUnlock desbloqueia o mutex de leitura.
func (s *Storage) RUnlock() {
	s.mu.RUnlock()
}

// ensureFileExists verifica se um arquivo existe e o cria com um cabeçalho se não existir.
func ensureFileExists(filePath string, header []string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		color.Green("   ⟫  FILE %s NOT FOUND, CREATING...", filePath)
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("falha ao criar arquivo %s: %w", filePath, err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("falha ao escrever cabeçalho em %s: %w", filePath, err)
		}
		writer.Flush()
		color.Green("   ⟫  FILE %s CREATED WITH SUCCESS...", filePath)
	}
	return nil
}

// --- Funções de Usuários ---

func (s *Storage) loadUsers() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ensureFileExists(config.UsersCSV, config.UserFieldnames); err != nil {
		log.Fatalf("Erro ao garantir a existência do arquivo de usuários: %v", err)
	}

	file, err := os.Open(config.UsersCSV)
	if err != nil {
		log.Fatalf("Erro ao abrir o arquivo de usuários: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Read() // Pula o cabeçalho

	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Erro ao ler registros do CSV de usuários: %v", err)
	}

	for _, record := range records {
		id, _ := strconv.ParseInt(record[0], 10, 64)
		dailyLimit, _ := strconv.Atoi(record[4])
		searchesToday, _ := strconv.Atoi(record[5])

		s.Users[id] = &User{
			ID:               id,
			RegistrationDate: record[1],
			EndDate:          record[2],
			Premium:          record[3],
			DailyLimit:       dailyLimit,
			SearchesToday:    searchesToday,
			LastSearchDate:   record[6],
		}
	}
	color.Green("   ⟫  LOADED %d USERS...", len(s.Users))
}

func (s *Storage) SaveUsers() {
	// A trava (lock) foi removida daqui. A função que chama SaveUsers é responsável por travar.
	file, err := os.Create(config.UsersCSV)
	if err != nil {
		color.Red("   ⟫  ERRO AO SALVAR ARQUIVO DE USUÁRIOS: %v", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write(config.UserFieldnames)

	for _, user := range s.Users {
		record := []string{
			strconv.FormatInt(user.ID, 10),
			user.RegistrationDate,
			user.EndDate,
			user.Premium,
			strconv.Itoa(user.DailyLimit),
			strconv.Itoa(user.SearchesToday),
			user.LastSearchDate,
		}
		writer.Write(record)
	}
	writer.Flush()
}

// --- Funções de Convites ---

func (s *Storage) loadInvites() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ensureFileExists(config.InvitesCSV, config.InviteFieldnames); err != nil {
		log.Fatalf("Erro ao garantir a existência do arquivo de convites: %v", err)
	}

	file, err := os.Open(config.InvitesCSV)
	if err != nil {
		log.Fatalf("Erro ao abrir arquivo de convites: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Read() // Pula cabeçalho

	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Erro ao ler registros do CSV de convites: %v", err)
	}

	for _, record := range records {
		days, _ := strconv.Atoi(record[1])
		limit, _ := strconv.Atoi(record[2])
		used, _ := strconv.Atoi(record[3])

		invite := &Invite{
			Code:  record[0],
			Days:  days,
			Limit: limit,
			Used:  used,
		}
		s.Invites[invite.Code] = invite
	}
	color.Green("   ⟫  LOADED %d INVITES...", len(s.Invites))
}

func (s *Storage) SaveInvites() {
	// A trava (lock) foi removida daqui.
	file, err := os.Create(config.InvitesCSV)
	if err != nil {
		color.Red("   ⟫  ERRO AO SALVAR ARQUIVO DE CONVITES: %v", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write(config.InviteFieldnames)

	for _, invite := range s.Invites {
		record := []string{
			invite.Code,
			strconv.Itoa(invite.Days),
			strconv.Itoa(invite.Limit),
			strconv.Itoa(invite.Used),
		}
		writer.Write(record)
	}
	writer.Flush()
}

// --- Funções de Chats ---

func (s *Storage) loadChats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.OpenFile(config.ChatsFile, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("Erro ao abrir ou criar o arquivo de chats: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if id, err := strconv.ParseInt(line, 10, 64); err == nil {
			s.Chats[id] = true
		}
	}
	color.Green("   ⟫  LOADED %d CHATS...", len(s.Chats))
}

func (s *Storage) SaveChats() {
	// A trava (lock) foi removida daqui.
	file, err := os.Create(config.ChatsFile)
	if err != nil {
		color.Red("   ⟫  ERRO AO SALVAR ARQUIVO DE CHATS: %v", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for id := range s.Chats {
		fmt.Fprintln(writer, id)
	}
	writer.Flush()
}
