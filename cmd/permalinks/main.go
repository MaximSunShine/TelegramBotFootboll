// cmd/permalinks/main.go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// Config хранит параметры запуска
type Config struct {
	repo      string
	branch    string
	output    string
	filterExt string
	verbose   bool
}

func main() {
	cfg := parseFlags()

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	if cfg.verbose {
		fmt.Fprintf(os.Stderr, "✅ Готово! Ссылки сохранены в %s\n", cfg.output)
	}
}

func parseFlags() Config {
	var cfg Config
	flag.StringVar(&cfg.branch, "b", "HEAD", "Ветка или коммит (по умолчанию: HEAD)")
	flag.StringVar(&cfg.output, "o", "permalinks.txt", "Имя выходного файла")
	flag.StringVar(&cfg.filterExt, "ext", "", "Фильтр по расширению (например: .go)")
	flag.BoolVar(&cfg.verbose, "v", false, "Подробный вывод в stderr")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go run cmd/permalinks/main.go owner/repo [flags]")
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	cfg.repo = flag.Arg(0)
	return cfg
}

func run(cfg Config) error {
	// 1. Получаем SHA коммита
	commit, err := runGit("rev-parse", cfg.branch)
	if err != nil {
		return fmt.Errorf("failed to resolve commit for %q: %w", cfg.branch, err)
	}
	commit = strings.TrimSpace(commit)

	if cfg.verbose {
		fmt.Fprintf(os.Stderr, "🔍 Branch: %s → Commit: %s\n", cfg.branch, commit[:8])
	}

	// 2. Получаем список файлов
	filesRaw, err := runGit("ls-tree", "-r", "--name-only", cfg.branch)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	// 3. Подготовка к записи
	outFile, err := os.Create(cfg.output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

	baseURL := fmt.Sprintf("https://github.com/%s/blob/%s", cfg.repo, commit)
	scanner := bufio.NewScanner(strings.NewReader(filesRaw))

	var count int
	for scanner.Scan() {
		filePath := scanner.Text()

		// Фильтр по расширению
		if cfg.filterExt != "" && !strings.HasSuffix(filePath, cfg.filterExt) {
			continue
		}

		// Кодируем путь: сохраняем '/', но экранируем пробелы и спецсимволы
		escaped := escapePath(filePath)
		permalink := baseURL + "/" + escaped

		// Пишем в файл и в консоль
		_, err := writer.WriteString(permalink + "\n")
		if err != nil {
			return fmt.Errorf("failed to write permalink: %w", err)
		}
		fmt.Println(permalink)
		count++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file list: %w", err)
	}

	if cfg.verbose {
		fmt.Fprintf(os.Stderr, "📊 Всего файлов: %d (отфильтровано: %s)\n", count,
			map[bool]string{true: cfg.filterExt, false: "нет"}[cfg.filterExt != ""])
	}

	return nil
}

// escapePath кодирует путь для GitHub URL:
// - сохраняет '/' как разделитель папок
// - экранирует пробелы, кириллицу, #, ?, % и другие спецсимволы
func escapePath(path string) string {
	// Разбиваем на сегменты, кодируем каждый, собираем обратно
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		// PathEscape закодирует всё, кроме букв/цифр/-_./~
		// Но нам нужно сохранить '/' между сегментами — поэтому обрабатываем по частям
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = "." // явно указываем рабочую директорию
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %v failed: %s", args, string(exitErr.Stderr))
		}
		return "", err
	}
	return string(out), nil
}
