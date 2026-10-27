package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TelegramPayload adalah struct untuk body JSON yang dikirim ke API Telegram.
// Di Golang, struct tag `json:"..."` mirip dengan properti array/resource di Laravel.
type TelegramPayload struct {
	ChatID                string `json:"chat_id"`
	MessageThreadID       int    `json:"message_thread_id,omitempty"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

// GitInfo menyimpan informasi seputar repository & aktivitas Git saat ini.
type GitInfo struct {
	Action      string
	RepoName    string
	Branch      string
	UserName    string
	UserEmail   string
	LastCommit  string
	CommitHash  string
	ChangedStat string
	RemoteURL   string
}

func main() {
	// 1. Baca argumen aksi Git dari command line (misal: "push", "pull", "commit", "merge", atau "test")
	action := "test"
	if len(os.Args) > 1 {
		action = strings.ToLower(os.Args[1])
	}

	// 2. Muat konfigurasi dari file .env (otomatis cari di folder tempat binary git-notif.exe berada)
	loadEnv()

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if botToken == "" || chatID == "" || strings.Contains(botToken, "ISI_TOKEN") {
		fmt.Println("[Git-Notif] ⚠️ TELEGRAM_BOT_TOKEN atau TELEGRAM_CHAT_ID belum diisi di file .env!")
		os.Exit(0) // Exit 0 supaya tidak menggagalkan perintah git Anda
	}

	// 3. Kumpulkan data Git dari folder repository tempat perintah dijalankan
	info := collectGitInfo(action)

	// 4. Format pesan HTML yang rapi untuk dikirim ke grup Telegram
	message := formatTelegramMessage(info)

	// 5. Kirim pesan ke Telegram via HTTP POST
	err := sendTelegramNotification(botToken, chatID, message)
	if err != nil {
		fmt.Printf("[Git-Notif] ❌ Gagal kirim notifikasi Telegram: %v\n", err)
		os.Exit(0)
	}

	fmt.Printf("[Git-Notif] ✅ Notifikasi '%s' berhasil dikirim ke grup Telegram!\n", strings.ToUpper(action))
}

// loadEnv membaca file .env di lokasi binary maupun working directory saat ini
func loadEnv() {
	var paths []string

	// Cari .env di lokasi tempat file .exe berada (penting saat dipanggil dari repo lain via Git Hook)
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		paths = append(paths, filepath.Join(exeDir, ".env"))
	}
	// Cari juga di folder kerja saat ini
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, ".env"))
	}

	for _, p := range paths {
		content, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
				if os.Getenv(key) == "" {
					os.Setenv(key, val)
				}
			}
		}
		break
	}
}

// runGitCommand menjalankan perintah git dan mengembalikan output string-nya
func runGitCommand(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// collectGitInfo mengambil detail repo, user, branch, dan commit terakhir
func collectGitInfo(action string) GitInfo {
	// Nama folder repository
	repoRoot := runGitCommand("rev-parse", "--show-toplevel")
	repoName := filepath.Base(repoRoot)
	if repoRoot == "" {
		if wd, err := os.Getwd(); err == nil {
			repoName = filepath.Base(wd)
		} else {
			repoName = "Unknown-Repo"
		}
	}

	branch := runGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if branch == "" {
		branch = "main"
	}

	userName := runGitCommand("config", "user.name")
	if userName == "" {
		userName = os.Getenv("USERNAME")
	}
	userEmail := runGitCommand("config", "user.email")

	commitHash := runGitCommand("log", "-1", "--pretty=format:%h")
	lastCommit := runGitCommand("log", "-1", "--pretty=format:%s")
	changedStat := runGitCommand("diff", "--shortstat", "HEAD~1", "HEAD")
	if changedStat == "" {
		changedStat = runGitCommand("show", "--stat", "--oneline", "-1")
	}

	remoteURL := runGitCommand("config", "--get", "remote.origin.url")

	return GitInfo{
		Action:      action,
		RepoName:    repoName,
		Branch:      branch,
		UserName:    userName,
		UserEmail:   userEmail,
		LastCommit:  lastCommit,
		CommitHash:  commitHash,
		ChangedStat: changedStat,
		RemoteURL:   remoteURL,
	}
}

// formatTelegramMessage menyusun tampilan notifikasi yang bersih di grup Telegram
func formatTelegramMessage(info GitInfo) string {
	var icon, title string

	switch info.Action {
	case "push":
		icon = "🚀"
		title = "GIT PUSH"
	case "pull":
		icon = "📥"
		title = "GIT PULL (SYNC)"
	case "merge":
		icon = "🔀"
		title = "GIT MERGE"
	case "commit":
		icon = "📝"
		title = "GIT COMMIT"
	default:
		icon = "🔔"
		title = "GIT NOTIFICATION (" + strings.ToUpper(info.Action) + ")"
	}

	timestamp := time.Now().Format("02 Jan 2006 • 15:04:05 WIB")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s <b>%s</b>\n", icon, title))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("📦 <b>Repository:</b> <code>%s</code>\n", escapeHTML(info.RepoName)))
	sb.WriteString(fmt.Sprintf("🌿 <b>Branch:</b> <code>%s</code>\n", escapeHTML(info.Branch)))
	if info.UserEmail != "" {
		sb.WriteString(fmt.Sprintf("👤 <b>Developer:</b> %s (<code>%s</code>)\n", escapeHTML(info.UserName), escapeHTML(info.UserEmail)))
	} else {
		sb.WriteString(fmt.Sprintf("👤 <b>Developer:</b> %s\n", escapeHTML(info.UserName)))
	}

	if info.LastCommit != "" {
		sb.WriteString(fmt.Sprintf("💬 <b>Commit Terakhir:</b> <code>%s</code> - %s\n", escapeHTML(info.CommitHash), escapeHTML(info.LastCommit)))
	}
	if info.ChangedStat != "" {
		lines := strings.Split(info.ChangedStat, "\n")
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		sb.WriteString(fmt.Sprintf("📊 <b>Perubahan:</b> <i>%s</i>\n", escapeHTML(lastLine)))
	}
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("🕒 <i>%s</i>", timestamp))

	return sb.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func sendTelegramNotification(botToken, chatID, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := TelegramPayload{
		ChatID:                chatID,
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	}

	if topicStr := strings.TrimSpace(os.Getenv("TELEGRAM_TOPIC_ID")); topicStr != "" {
		if tid, err := strconv.Atoi(topicStr); err == nil && tid > 0 {
			payload.MessageThreadID = tid
		}
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
