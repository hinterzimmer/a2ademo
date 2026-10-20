package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ANSI color escape codes
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorGray    = "\033[90m"
	colorBold    = "\033[1m"
)

var (
	mu sync.Mutex
)

// Tag represents a categorized logging event.
type Tag string

const (
	TagDiscovery    Tag = "A2A DISCOVERY"
	TagRequest      Tag = "A2A REQUEST"
	TagResponse     Tag = "A2A RESPONSE"
	TagBedrock      Tag = "BEDROCK LLM"
	TagOrchestrator Tag = "ORCHESTRATOR"
	TagAgent        Tag = "AGENT EXEC"
	TagInfo         Tag = "INFO"
	TagError        Tag = "ERROR"
)

func tagColor(tag Tag) string {
	switch tag {
	case TagDiscovery:
		return colorMagenta
	case TagRequest:
		return colorCyan
	case TagResponse:
		return colorGreen
	case TagBedrock:
		return colorYellow
	case TagOrchestrator:
		return colorBlue
	case TagAgent:
		return colorCyan
	case TagError:
		return colorRed
	default:
		return colorGray
	}
}

// Log prints a formatted, color-coded protocol trace log.
func Log(tag Tag, format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()

	ts := time.Now().Format("15:04:05.000")
	tc := tagColor(tag)
	msg := fmt.Sprintf(format, args...)

	fmt.Fprintf(os.Stdout, "%s%s%s %s[%-14s]%s %s\n",
		colorGray, ts, colorReset,
		tc+colorBold, string(tag), colorReset,
		msg,
	)
}

// Section prints a highlighted visual section separator.
func Section(title string) {
	mu.Lock()
	defer mu.Unlock()

	line := strings.Repeat("─", 60)
	fmt.Fprintf(os.Stdout, "\n%s%s%s\n%s%s %s%s\n%s%s%s\n",
		colorBlue, line, colorReset,
		colorBold, "▶", title, colorReset,
		colorBlue, line, colorReset,
	)
}
