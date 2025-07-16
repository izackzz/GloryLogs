package search

import (
	"bufio"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"glorylogs-bot/internal/config"
)

// Criterion representa uma única condição de busca, como "inurl:google".
type Criterion struct {
	Operator string
	Term     string
}

// lineData armazena as partes de uma linha de log analisada.
type lineData struct {
	URL      string
	User     string
	Password string
}

// searchRegex é a expressão regular para analisar a query de busca.
var searchRegex = regexp.MustCompile(`(?P<operator>inurl:|intext:|site:|filetype:)?(?P<term>"[^"]+"|\S+)`)

// ParseSearchQuery analisa a string de busca do usuário e a converte em uma lista de critérios.
func ParseSearchQuery(query string) []Criterion {
	criteria := []Criterion{}
	matches := searchRegex.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		op := match[1]   // O grupo 'operator'
		term := match[2] // O grupo 'term'

		// Limpa o operador e o termo
		if op != "" {
			op = strings.TrimSuffix(op, ":")
		}
		term = strings.Trim(term, `"`)

		// Define o operador padrão
		if op == "" {
			op = "term" // 'term' é o nosso operador padrão, como no Python
		}

		criteria = append(criteria, Criterion{
			Operator: strings.ToLower(op),
			Term:     strings.ToLower(term),
		})
	}
	return criteria
}

// parseLine divide uma linha de texto no formato URL:USER:PASS.
func ParseLine(line string) *lineData {
	line = strings.TrimSpace(line)
	parts := strings.Split(line, ":")
	if len(parts) < 3 {
		return nil
	}
	// Junta todas as partes, exceto as duas últimas, para formar a URL
	// Isso lida com casos onde a URL contém ":" (ex: https://...)
	urlPart := strings.Join(parts[:len(parts)-2], ":")
	userPart := parts[len(parts)-2]
	passPart := parts[len(parts)-1]

	return &lineData{URL: urlPart, User: userPart, Password: passPart}
}

// extractDomain extrai o domínio (host) de uma URL.
func extractDomain(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "" // Retorna vazio se a URL for inválida
	}
	return parsedURL.Hostname()
}

// LineMatchesCriteria verifica se uma linha de log corresponde a todos os critérios de busca.
func LineMatchesCriteria(data *lineData, criteria []Criterion) bool {
	urlLower := strings.ToLower(data.URL)
	userLower := strings.ToLower(data.User)
	passLower := strings.ToLower(data.Password)

	for _, c := range criteria {
		match := false
		switch c.Operator {
		case "inurl":
			if strings.Contains(urlLower, c.Term) {
				match = true
			}
		case "site":
			domain := extractDomain(urlLower)
			if strings.Contains(domain, c.Term) {
				match = true
			}
		case "filetype":
			if strings.HasSuffix(urlLower, "."+c.Term) {
				match = true
			}
		case "intext":
			if strings.Contains(userLower, c.Term) || strings.Contains(passLower, c.Term) {
				match = true
			}
		case "term": // Operador padrão
			if strings.Contains(urlLower, c.Term) || strings.Contains(userLower, c.Term) || strings.Contains(passLower, c.Term) {
				match = true
			}
		}
		// Se qualquer critério não for satisfeito, a linha inteira não corresponde.
		if !match {
			return false
		}
	}
	// Se todos os critérios foram satisfeitos.
	return true
}

// Search executa a busca na pasta de logs com base na query fornecida.
func Search(query string) []string {

	// log.Println("-> Iniciando busca nos arquivos...")

	criteria := ParseSearchQuery(query)
	if len(criteria) == 0 {
		return []string{} // Retorna vazio se a query for vazia ou inválida
	}

	var results []string

	err := filepath.Walk(config.LogsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Pula diretórios e foca apenas em arquivos .txt
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".txt") {
			file, err := os.Open(path)
			if err != nil {

				// log.Printf("... Processando arquivo: %s", path)

				log.Printf("Aviso: Não foi possível abrir o arquivo %s: %v", path, err)
				return nil // Continua para o próximo arquivo
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				lineData := ParseLine(line)
				if lineData != nil && LineMatchesCriteria(lineData, criteria) {
					results = append(results, line)
				}
			}
			if err := scanner.Err(); err != nil {
				log.Printf("Aviso: Erro ao ler o arquivo %s: %v", path, err)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Erro ao percorrer a pasta de logs: %v", err)
	}

	// log.Printf("-> Busca finalizada. Encontrados %d resultados.", len(results))

	return results
}
