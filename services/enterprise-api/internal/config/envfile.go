package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// envSource checks the process environment first, then values parsed from a dotenv file.
type envSource struct {
	file map[string]string
}

func (s envSource) lookup(key string) (string, bool) {
	if value, ok := os.LookupEnv(key); ok {
		return value, true
	}
	value, ok := s.file[key]
	return value, ok
}

func (s envSource) getenv(key, fallback string) string {
	if value, ok := s.lookup(key); ok {
		return value
	}
	return fallback
}

func loadEnvFiles() (envSource, error) {
	if value, ok := os.LookupEnv("ENV_FILE"); ok {
		if value == "" {
			return envSource{}, nil
		}
		file, err := parseEnvFile(value)
		if err != nil {
			return envSource{}, err
		}
		return envSource{file: file}, nil
	}

	candidates := []string{
		".env",
		filepath.Join("..", ".env"),
		filepath.Join("..", "..", ".env"),
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		file, err := parseEnvFile(candidate)
		if err != nil {
			return envSource{}, err
		}
		return envSource{file: file}, nil
	}
	return envSource{}, nil
}

func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open env file: %w", err)
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if lineNo == 1 {
			line = strings.TrimPrefix(line, "\uFEFF")
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok || !validEnvKey(strings.TrimSpace(key)) {
			return nil, fmt.Errorf("invalid env file line %d", lineNo)
		}
		key = strings.TrimSpace(key)
		if _, exists := values[key]; exists {
			continue
		}
		values[key] = unquoteEnv(strings.TrimSpace(value))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}
	return values, nil
}

func validEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if r == '_' || unicode.IsLetter(r) || (unicode.IsDigit(r) && i > 0) {
			continue
		}
		return false
	}
	return true
}

func unquoteEnv(value string) string {
	if len(value) < 2 {
		return value
	}
	if value[0] == '"' && value[len(value)-1] == '"' {
		if parsed, err := strconv.Unquote(value); err == nil {
			return parsed
		}
	}
	if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}
