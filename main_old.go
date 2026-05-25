package main1

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db      *sql.DB
	apiKey  = os.Getenv("SSTATS_API_KEY")
	adminID int64
)

// --- Структуры для SStats.net API ---
type SStatsResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

type SStatsTeam struct {
	Name string `json:"name"`
}

type SStatsMatch struct {
	ID         int        `json:"id"`
	HomeTeam   SStatsTeam `json:"homeTeam"`
	AwayTeam   SStatsTeam `json:"awayTeam"`
	HomeScore  *int       `json:"homeScore"`
	AwayScore  *int       `json:"awayScore"`
	Status     int        `json:"status"` // 0=NS, 1=1H, 2=2H, 3+=FT
	StartTime  string     `json:"startTime"`
	LeagueID   int        `json:"leagueId"`
	SeasonYear int        `json:"seasonYear"`
}

// --- Инициализация БД ---
func initDB() error {
	schema := `
        CREATE TABLE IF NOT EXISTS users (
                telegram_id INTEGER PRIMARY KEY,
                username TEXT,
                points INTEGER DEFAULT 0,
                first_seen TEXT DEFAULT CURRENT_TIMESTAMP,
                last_active TEXT DEFAULT CURRENT_TIMESTAMP
        );
        CREATE TABLE IF NOT EXISTS matches (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                api_fixture_id INTEGER UNIQUE,
                team_a TEXT NOT NULL,
                team_b TEXT NOT NULL,
                match_date TEXT,
                status TEXT DEFAULT 'open'
        );
        CREATE TABLE IF NOT EXISTS predictions (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                user_id INTEGER,
                match_id INTEGER,
                pred_score_a INTEGER,
                pred_score_b INTEGER,
                UNIQUE(user_id, match_id),
                FOREIGN KEY(user_id) REFERENCES users(telegram_id),
                FOREIGN KEY(match_id) REFERENCES matches(id)
        );`
	_, err := db.Exec(schema)
	return err
}

// --- Вспомогательные функции ---
func registerUser(userID int64, username string) error {
	query := `INSERT INTO users (telegram_id, username, last_active)
                  VALUES (?, ?, CURRENT_TIMESTAMP)
                  ON CONFLICT(telegram_id) DO UPDATE SET username = excluded.username, last_active = CURRENT_TIMESTAMP`
	_, err := db.Exec(query, userID, username)
	return err
}

func getUserStats(userID int64) (string, error) {
	var username sql.NullString
	var points int
	err := db.QueryRow("SELECT username, points FROM users WHERE telegram_id = ?", userID).Scan(&username, &points)
	if err != nil {
		return "", err
	}
	name := "Игрок"
	if username.Valid && username.String != "" {
		name = username.String
	}
	return fmt.Sprintf("👤 @%s\n🏆 Очков: %d", name, points), nil
}

func calculatePoints(pA, pB, aA, aB int) int {
	points := 0
	if getWinner(pA, pB) == getWinner(aA, aB) {
		points += 3
	}
	if pA == aA && pB == aB {
		points += 3
	}
	return points
}

func getWinner(a, b int) string {
	if a > b {
		return "home"
	} else if a < b {
		return "away"
	}
	return "draw"
}

func sendAndAutoDelete(bot *tgbotapi.BotAPI, chatID int64, text string, userMsgID int) {
	msg, err := bot.Send(tgbotapi.NewMessage(chatID, text))
	if err != nil {
		log.Printf("❌ Ошибка отправки: %v", err)
		return
	}
	go func() {
		time.Sleep(20 * time.Second)
		_, _ = bot.Send(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
		if userMsgID > 0 {
			_, _ = bot.Send(tgbotapi.NewDeleteMessage(msg.Chat.ID, userMsgID))
		}
	}()
}

// --- Обработчики команд ---
func handleMatches(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	rows, err := db.Query("SELECT id, team_a, team_b, match_date FROM matches WHERE status='open' ORDER BY match_date ASC LIMIT 10")
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка чтения БД"))
		return
	}
	defer rows.Close()

	var response strings.Builder
	response.WriteString("⚽ Открытые матчи:\n\n")
	count := 0

	for rows.Next() {
		var id int
		var teamA, teamB, date string
		rows.Scan(&id, &teamA, &teamB, &date)
		if len(date) >= 16 {
			date = date[:16]
		}
		response.WriteString(fmt.Sprintf("🆔 *%d* | %s 🆚 %s\n📅 %s\n", id, teamA, teamB, date))
		count++
	}

	if count == 0 {
		response.WriteString("(_Пока нет открытых матчей_)")
	}
	response.WriteString("\n📝 Прогноз: `/predict <id> <счёт>`\nПример: `/predict 1 2:1`")

	out := tgbotapi.NewMessage(msg.Chat.ID, response.String())
	out.ParseMode = "Markdown"
	bot.Send(out)
}

func handlePredict(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.Fields(msg.Text)
	if len(args) != 3 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка формата. Пример: `/predict 1 2:1`"))
		return
	}
	matchID, err := strconv.Atoi(args[1])
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ ID матча должен быть числом"))
		return
	}
	scorePattern := regexp.MustCompile(`^(\d+)[:\-](\d+)$`)
	matches := scorePattern.FindStringSubmatch(args[2])
	if matches == nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Счёт в формате X:Y или X-Y"))
		return
	}
	scoreA, _ := strconv.Atoi(matches[1])
	scoreB, _ := strconv.Atoi(matches[2])

	// 1. Проверяем, существует ли матч и открыт ли он
	var status string
	err = db.QueryRow("SELECT status FROM matches WHERE id=?", matchID).Scan(&status)
	if err == sql.ErrNoRows {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Матч с таким ID не найден. Проверь список: `/matches`"))
		return
	} else if err != nil {
		log.Printf("⚠️ Ошибка проверки матча: %v", err)
	} else if status != "open" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Матч уже завершён или закрыт."))
		return
	}

	// 2. Сохраняем прогноз (INSERT OR REPLACE работает на всех версиях SQLite)
	_, err = db.Exec(`INSERT OR REPLACE INTO predictions (user_id, match_id, pred_score_a, pred_score_b)
                          VALUES (?, ?, ?, ?)`,
		msg.From.ID, matchID, scoreA, scoreB)

	if err != nil {
		log.Printf("❌ Ошибка БД при сохранении: %v", err)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Ошибка сохранения: %v", err)))
	} else {
		var tA, tB string
		db.QueryRow("SELECT team_a, team_b FROM matches WHERE id=?", matchID).Scan(&tA, &tB)
		resp := fmt.Sprintf("✅ Прогноз принят!\n\n🆚 %s 🆚 %s\n🔮 Твой счёт: %d:%d", tA, tB, scoreA, scoreB)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, resp))
	}
}
func handleTop(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	rows, err := db.Query("SELECT username, points FROM users ORDER BY points DESC LIMIT 5")
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка чтения статистики"))
		return
	}
	defer rows.Close()

	var response strings.Builder
	response.WriteString("🏆 ТОП Прогнозистов:\n\n")
	i := 1
	for rows.Next() {
		var name string
		var pts int
		rows.Scan(&name, &pts)
		if name == "" {
			name = "Аноним"
		}
		response.WriteString(fmt.Sprintf("%d. %s — %d очков\n", i, name, pts))
		i++
	}
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, response.String()))
}

func handleAddMatch(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if msg.From.ID != adminID {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🔒 Только админ"))
		return
	}
	args := strings.Join(strings.Fields(msg.Text)[1:], " ")
	parts := strings.SplitN(args, " ", 2)
	if len(parts) != 2 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Формат: `/addmatch Команда1 Команда2`"))
		return
	}
	_, err := db.Exec("INSERT INTO matches (team_a, team_b) VALUES (?, ?)", parts[0], parts[1])
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка создания матча"))
	} else {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Матч создан! Используй `/matches` чтобы узнать ID."))
	}
}

// --- SStats.net API Клиент ---
func fetchSStatsFixtures(leagueID, year int) ([]SStatsMatch, error) {
	url := fmt.Sprintf("https://api.sstats.net/games/list?LeagueId=%d&Year=%d", leagueID, year)
	if apiKey != "" {
		url += "&apikey=" + apiKey
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw SStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	var matches []SStatsMatch
	if err := json.Unmarshal(raw.Data, &matches); err != nil {
		return nil, err
	}
	return matches, nil
}

func checkSStatsMatchStatus(matchID int) (*SStatsMatch, error) {
	url := fmt.Sprintf("https://api.sstats.net/games/%d", matchID)
	if apiKey != "" {
		url += "?apikey=" + apiKey
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw SStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	var match SStatsMatch
	if err := json.Unmarshal(raw.Data, &match); err != nil {
		return nil, err
	}
	return &match, nil
}

// --- Фоновая синхронизация результатов ---
func startBackgroundSync(bot *tgbotapi.BotAPI) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rows, err := db.Query("SELECT id, api_fixture_id, team_a, team_b FROM matches WHERE status='open' AND api_fixture_id > 0")
		if err != nil {
			log.Println("❌ Ошибка чтения матчей для синхронизации:", err)
			continue
		}

		for rows.Next() {
			var dbID, sstatsID int
			var tA, tB string
			rows.Scan(&dbID, &sstatsID, &tA, &tB)

			match, err := checkSStatsMatchStatus(sstatsID)
			if err != nil {
				log.Printf("⚠️ Не удалось проверить матч #%d: %v", sstatsID, err)
				continue
			}

			if match.Status >= 3 && match.HomeScore != nil && match.AwayScore != nil {
				log.Printf("🏁 Матч завершён: %s %d:%d %s", tA, *match.HomeScore, *match.AwayScore, tB)
				db.Exec("UPDATE matches SET status='closed' WHERE id=?", dbID)
				awardPointsAndNotify(bot, dbID, *match.HomeScore, *match.AwayScore)
			}
		}
		rows.Close()
	}
}

func awardPointsAndNotify(bot *tgbotapi.BotAPI, matchID, actA, actB int) {
	rows, _ := db.Query("SELECT user_id, pred_score_a, pred_score_b FROM predictions WHERE match_id=?", matchID)
	defer rows.Close()

	var report strings.Builder
	totalUsers := 0
	for rows.Next() {
		var uID, pA, pB int
		rows.Scan(&uID, &pA, &pB)
		pts := calculatePoints(pA, pB, actA, actB)
		db.Exec("UPDATE users SET points = points + ? WHERE telegram_id=?", pts, uID)
		if pts > 0 {
			report.WriteString(fmt.Sprintf("👤 ID:%d | Прогноз: %d:%d | +%d очков\n", uID, pA, pB, pts))
		}
		totalUsers++
	}

	if totalUsers == 0 {
		report.WriteString("Никто не сделал прогноз на этот матч.")
	}

	log.Printf("📊 Отчёт по матчу #%d (%d:%d):\n%s", matchID, actA, actB, report.String())
}

// --- Обработчик /sync для SStats ---
func handleSync(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if msg.From.ID != adminID {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🔒 Только админ"))
		return
	}
	args := strings.Fields(msg.Text)
	if len(args) != 3 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Формат: `/sync <LeagueId> <Year>`\nПример: `/sync 41 2026`"))
		return
	}
	leagueID, _ := strconv.Atoi(args[1])
	year, _ := strconv.Atoi(args[2])

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Загрузка матчей из SStats..."))
	matches, err := fetchSStatsFixtures(leagueID, year)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка API: "+err.Error()))
		return
	}

	tx, _ := db.Begin()
	stmt, _ := tx.Prepare(`INSERT OR IGNORE INTO matches (api_fixture_id, team_a, team_b, match_date, status) VALUES (?, ?, ?, ?, 'open')`)
	added := 0
	for _, m := range matches {
		if m.Status >= 3 {
			continue
		}
		_, _ = stmt.Exec(m.ID, m.HomeTeam.Name, m.AwayTeam.Name, m.StartTime)
		added++
	}
	stmt.Close()
	tx.Commit()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ Загружено %d будущих матчей. Бот будет проверять результаты автоматически.", added)))
}

// --- MAIN ---
func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("❌ Установи TELEGRAM_BOT_TOKEN")
	}
	adminID, _ = strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)

	var err error
	db, err = sql.Open("sqlite3", "predictions.db")
	if err != nil {
		log.Fatal("❌ Ошибка БД: ", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if err := initDB(); err != nil {
		log.Fatal("❌ Ошибка схемы: ", err)
	}
	log.Println("✅ БД подключена")

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	_, _ = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
	log.Printf("✅ Бот запущен: @%s", bot.Self.UserName)

	go startBackgroundSync(bot)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		_ = registerUser(update.Message.From.ID, update.Message.From.UserName)

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "foot":
				text := `👋 Привет! Я бот для прогнозов на футбол.

📋 Команды:
/foot — приветствие
/matches — список игр
/predict [ID] [Счёт] — прогноз
/top — таблица лидеров
/addmatch [К1] [К2] — создать матч (админ)
/sync [LeagueId] [Year] — загрузить матчи (админ)`
				sendAndAutoDelete(bot, update.Message.Chat.ID, text, update.Message.MessageID)

			case "stats":
				stats, err := getUserStats(update.Message.From.ID)
				if err != nil {
					sendAndAutoDelete(bot, update.Message.Chat.ID, "❌ Ошибка загрузки", update.Message.MessageID)
				} else {
					sendAndAutoDelete(bot, update.Message.Chat.ID, stats, update.Message.MessageID)
				}

			case "matches":
				handleMatches(bot, update.Message)
			case "predict":
				handlePredict(bot, update.Message)
			case "top":
				handleTop(bot, update.Message)
			case "addmatch":
				handleAddMatch(bot, update.Message)
			case "sync":
				handleSync(bot, update.Message)
			default:
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "❓ Неизвестная команда"))
			}
		} // ✅ Добавлена закрывающая скобка для if
	} // ✅ Добавлена закрывающая скобка для for
} // ✅ Добавлена закрывающая скобка для main
