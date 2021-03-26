package crash

import (
	"fmt"
	"strings"
	"time"
)

// Report - основное представление внешней ошибки
type Report struct {
	ID      string
	Code    string
	Link    string
	Title   string
	Status  uint16
	Created time.Time
	Entries []*ReportEntry
	Debug   map[string]string
}

// ReportEntry - основное представление ошибки в цепочке
type ReportEntry struct {
	Text   string
	Detail string
	Stack  []string
	Debug  map[string]string
}

type RFC struct {
	ID      string      `json:"id"`
	Code    string      `json:"code"`
	Link    string      `json:"type"`
	Title   string      `json:"title"`
	Status  uint16      `json:"status"`
	Created string      `json:"created"`
	Detail  string      `json:"detail,omitempty"`
	Events  interface{} `json:"events,omitempty"`
}

type ViewMonitoring struct {
	ID      string                 `json:"id"`
	Status  int                    `json:"status"`
	Code    string                 `json:"code"`
	Link    string                 `json:"link,omitempty"`
	Title   string                 `json:"title"`
	Debug   map[string]string      `json:"debug"`
	Created time.Time              `json:"created"`
	Entries []*ViewMonitoringEntry `json:"entries,omitempty"`
}

type ViewMonitoringEntry struct {
	Text   string            `json:"text"`
	Detail string            `json:"detail,omitempty"`
	Stack  []string          `json:"stack,omitempty"`
	Debug  map[string]string `json:"debug,omitempty"`
}

func (r *Report) AsRFC() *RFC {
	rfc := &RFC{
		ID:      r.ID,
		Code:    r.Code,
		Created: r.Created.Format(time.RFC3339Nano),
		Link:    r.Link,
		Title:   r.Title,
		Status:  r.Status,
	}

	if rfc.Link == "" {
		rfc.Link = BlankLink
	}

	if r.Status < 500 {
		parts := make([]string, 0, len(r.Entries))
		for i := len(r.Entries) - 1; i >= 0; i-- {
			if r.Entries[i].Detail != "" {
				parts = append(parts, r.Entries[i].Detail)
			}
		}

		rfc.Detail = strings.Join(parts, " => ")
	}

	return rfc
}

func (r *Report) Format(f fmt.State, verb rune) {
	// Сначала всегда на той же строке основное сообщение
	fmt.Fprintf(f, "[ %d ] %s", r.Status, r.Title)

	// Если не нужна детальная инфа, на этом все
	if verb != 'v' {
		return
	}

	// Затем, на каждой строчке со сдвигом и кареткой, отладка (если есть)
	for i := range r.Entries {
		if f.Flag('+') {
			fmt.Fprintf(f, "%+v", r.Entries[i])
		} else {
			fmt.Fprintf(f, "%v", r.Entries[i])
		}
	}
}

func (r *ReportEntry) Format(f fmt.State, verb rune) {
	// Сначала всегда на той же строке основное сообщение
	fmt.Fprintf(f, "\n|-> %s", r.Text)

	// Затем в скобках детализация для пользователя
	if r.Detail != "" {
		fmt.Fprintf(f, " (%s)", r.Detail)
	}

	// Затем, на каждой строчке со сдвигом и кареткой, отладка (если есть)
	for key := range r.Debug {
		fmt.Fprintf(f, "\n|   %s: %s", key, r.Debug[key])
	}

	// Затем, если нужны подробности, выводим стек
	if f.Flag('+') && len(r.Stack) > 0 {
		fmt.Fprintf(f, "\n|       %s", strings.Join(r.Stack, "\n|       "))
	}
}
