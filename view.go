package journal

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shestakovda/journal/crash"
)

// Entry - основное представление записи журнала
type Entry struct {
	ID      string
	Host    string
	Service string
	Start   time.Time
	Total   time.Duration
	Chain   []*Stage
}

func (v Entry) String() string {
	const newline byte = '\n'
	var buf strings.Builder

	// Примерная прикидка, чтобы сэкономить в большинстве случаев
	buf.Grow(64 + len(v.ID) + 32 + len(v.Service) + 256*len(v.Chain))

	// Заголовок записи
	buf.WriteString("Запись: ")
	buf.WriteString(v.ID)
	buf.WriteString(" Хост: ")
	buf.WriteString(v.Host)
	buf.WriteString(" Старт: ")
	buf.WriteString(v.Start.Format(time.RFC3339Nano))
	buf.WriteString(" Сервис: ")
	buf.WriteString(v.Service)
	buf.WriteByte(newline)

	// Контрольные точки
	for i := range v.Chain {
		buf.WriteString(v.Chain[i].String())
		buf.WriteByte(newline)
	}

	// Подвал
	buf.WriteString("Всего: ")
	buf.WriteString(v.Total.String())
	buf.WriteByte(newline)

	// Итоговый результат
	return buf.String()
}

// Stage - основное представление отметки в записи журнала
type Stage struct {
	EnID string
	Text string
	Wait time.Duration
	Verb int
	Type int
	Fail *crash.Report
}

func (v Stage) String() string {
	const wlen = 16
	const elen = 54
	const space = " "

	var eid string
	var buf strings.Builder

	// Примерная прикидка, чтобы сэкономить в большинстве случаев
	buf.Grow(80 + 2*len(v.Text))
	buf.WriteString("+ ")

	// Время контрольной точки с выравниванием пробелами справа
	wait := v.Wait.String()
	buf.WriteString(wait + space)

	if diff := wlen - utf8.RuneCountInString(wait) - 1; diff > 0 {
		buf.WriteString(strings.Repeat(space, diff))
	}

	// Идентификатор с выравниванием пробелами справа
	if v.EnID != "" {
		eid = v.EnID + " (" + strconv.Itoa(v.Type) + ") "
		buf.WriteString(eid)
	}

	if diff := elen - utf8.RuneCountInString(eid); diff > 0 {
		buf.WriteString(strings.Repeat(space, diff))
	}

	// Если ошибка - печатаем её, если нет - комментарий
	if v.Fail != nil {
		buf.WriteString(fmt.Sprintf("%+v", v.Fail))
	} else {
		buf.WriteString(v.Text)
	}

	// Итоговый результат
	return buf.String()
}

type API struct {
	ID     string            `json:"id,omitempty"`
	Start  time.Time         `json:"start"`
	Total  time.Duration     `json:"total"`
	Name   string            `json:"name"`
	User   string            `json:"user,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
	Keys   map[string]string `json:"keys,omitempty"`
	Stages []*StageAPI       `json:"stages"`
}

type StageAPI struct {
	Wait time.Duration `json:"wait"`
	Name string        `json:"name"`
	Type string        `json:"type,omitempty"`
	EnID string        `json:"enid,omitempty"`
}

type ViewMonitoring struct {
	ID      string             `json:"id,omitempty"`
	Host    string             `json:"host"`
	Service string             `json:"service"`
	Total   string             `json:"total"`
	Name    string             `json:"name"`
	User    string             `json:"user,omitempty"`
	Start   time.Time          `json:"start"`
	Time    uint64             `json:"time"`
	Tags    map[string]string  `json:"tags,omitempty"`
	Keys    map[string]string  `json:"keys,omitempty"`
	Stages  []*StageMonitoring `json:"stages"`
}

type StageMonitoring struct {
	Wait string `json:"wait"`
	Name string `json:"name"`
	Time uint64 `json:"time"`
	Type string `json:"type,omitempty"`
	EnID string `json:"enid,omitempty"`
}
