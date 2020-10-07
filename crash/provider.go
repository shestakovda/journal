package crash

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/shestakovda/envx"
	"github.com/shestakovda/errx"
	"github.com/shestakovda/typex"
)

const urlTemplate = "https://test.platform.astral.ru/documentation?tab=4#api4001"

// NewTestProvider - без адреса, только для тестов
func NewTestProvider() Provider {
	return &provider{
		srv:  "testing",
		tpls: make([]*tpl, 0, 256),
	}
}

/*
	NewProvider - конструктор менеджера работы с ошибками.

	* srv - код текущего сервиса, для формирования кодов ошибок
*/
func NewProvider(env envx.Provider, srv string) (_ Provider, err error) {
	p := &provider{
		srv:  srv,
		tpls: make([]*tpl, 0, 256),
	}

	if p.url, err = env.URL(EnvBaseURL, ""); err != nil {
		return
	}

	return p, nil
}

type provider struct {
	sync.RWMutex
	tpls []*tpl

	url string
	srv string
}

func (p *provider) Register(status, number int, title string, triggers ...error) {
	if status < 400 || status >= 600 {
		panic("Некорректный статус")
	}

	if title == "" {
		panic("Некорректный заголовок")
	}

	if len(triggers) == 0 {
		panic("Пустой список ошибок")
	}

	code := fmt.Sprintf("%s%d%d", p.srv, status, number)

	p.Lock()
	defer p.Unlock()
	p.tpls = append(p.tpls, &tpl{
		Code:   code,
		Link:   p.url + "#" + code,
		Title:  title,
		Status: uint16(status),
		Errx:   triggers,
	})
}

func (p *provider) Report(err error) (r *Report) {
	if err == nil {
		return nil
	}

	t := p.getTpl(err)

	r = &Report{
		ID:      typex.NewUUID().Hex(),
		Code:    t.Code,
		Link:    t.Link,
		Title:   t.Title,
		Status:  t.Status,
		Created: time.Now().UTC(),
		Entries: make([]*ReportEntry, 0, 8),
	}

	if e, ok := err.(errx.Error); ok {
		v := e.Export()
		for {
			r.Entries = append(r.Entries, &ReportEntry{
				Text:   v.Text,
				Detail: v.Detail,
				Stack:  v.Stack,
				Debug:  v.Debug,
			})

			if v = v.Next; v == nil {
				break
			}
		}
	} else {
		r.Entries = append(r.Entries, &ReportEntry{Text: err.Error()})
	}

	return r
}

func (p *provider) getTpl(err error) *tpl {
	p.RLock()
	defer p.RUnlock()

	for i := range p.tpls {
		if errx.Is(err, p.tpls[i].Errx...) {
			return p.tpls[i]
		}
	}

	code := fmt.Sprintf("%s%d%d", p.srv, http.StatusInternalServerError, 9)

	return &tpl{
		Code:   code,
		Link:   p.url + "#" + code,
		Title:  UnknownErrMsg,
		Status: uint16(http.StatusInternalServerError),
	}
}

type tpl struct {
	Code   string
	Link   string
	Title  string
	Status uint16
	Errx   []error
}
