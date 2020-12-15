package crash

import (
	"encoding/binary"
	"encoding/json"
	"strings"
	"time"

	"github.com/shestakovda/errx"
	"github.com/shestakovda/fdbx"
	"github.com/shestakovda/typex"
)

const verCrashV1 uint8 = 1

/*
	NewFactoryFDB - конструктор фабрики моделей в рамках транзакции

	* db - текущий объект транзакции, траслируется в основную базу
*/
func NewFactoryFDB(db fdbx.DB) Factory {
	return &fdbFactory{
		db: db.At(DatabaseAPI),
	}
}

type fdbFactory struct {
	db fdbx.DB
}

func (f *fdbFactory) New() Model { return &fdbModel{fac: f} }

func (f *fdbFactory) ByID(id string) (_ Model, err error) {
	var uid typex.UUID

	if uid, err = typex.ParseUUID(id); err != nil {
		return nil, ErrIDValidate.WithReason(err)
	}

	m := &fdbModel{ID: uid.Hex(), fac: f}

	if err = f.db.Load(nil, m); err != nil {
		dbg := errx.Debug{"ID": m.ID}

		if errx.Is(err, fdbx.ErrRecordNotFound) {
			return nil, ErrNotFound.WithReason(err).WithDebug(dbg)
		}

		return nil, ErrSelect.WithReason(err).WithDebug(dbg)
	}

	return m, nil
}

func (f *fdbFactory) ByDateCode(from, to time.Time, code string) (_ []Model, err error) {
	var ids []string

	opts := []fdbx.Option{fdbx.From(Unix(from)), fdbx.To(Unix(to))}

	if ids, err = f.db.SelectIDs(IndexCrashCreated, opts...); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"От момента": from.UTC().Format(time.RFC3339Nano),
			"До момента": to.UTC().Format(time.RFC3339Nano),
		})
	}

	if len(ids) == 0 {
		return nil, nil
	}

	if code != "" {
		var byCode []string

		if byCode, err = f.db.SelectIDs(IndexCrashCode, fdbx.Query(fdbx.S2B(code))); err != nil {
			return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
				"По коду": code,
			})
		}

		if ids = Intersect(ids, byCode); len(ids) == 0 {
			return nil, nil
		}
	}

	mods := make([]Model, len(ids))
	recs := make([]fdbx.Record, len(ids))

	for i := range ids {
		mod := &fdbModel{ID: ids[i], fac: f}
		mods[i] = mod
		recs[i] = mod
	}

	if err = f.db.Load(nil, recs...); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"По коду":    code,
			"От момента": from.UTC().Format(time.RFC3339Nano),
			"До момента": to.UTC().Format(time.RFC3339Nano),
		})
	}

	return mods, nil
}

func (f *fdbFactory) ImportReports(reports ...*Report) (err error) {
	return errx.ErrNotImplemented.WithStack()
}

func (f *fdbFactory) newRecord(ver uint8, id string) (fdbx.Record, error) {
	return &fdbModel{ID: id, fac: f}, nil
}

type fdbModel struct {
	ID      string      `json:"id"`
	Code    string      `json:"code,omitempty"`
	Link    string      `json:"link,omitempty"`
	Title   string      `json:"title"`
	Status  uint16      `json:"status"`
	Created time.Time   `json:"created"`
	Errx    []*fdbError `json:"chain"`

	fac *fdbFactory
}

type fdbError struct {
	Text   string            `json:"text"`
	Detail string            `json:"detail,omitempty"`
	Stack  []string          `json:"stack,omitempty"`
	Debug  map[string]string `json:"debug,omitempty"`
}

func (m *fdbModel) Import(r *Report) (err error) {
	m.ID = r.ID
	m.Code = r.Code
	m.Link = r.Link
	m.Title = r.Title
	m.Status = r.Status
	m.Created = r.Created
	m.Errx = make([]*fdbError, len(r.Entries))

	for i := range r.Entries {
		m.Errx[i] = &fdbError{
			Text:   r.Entries[i].Text,
			Detail: r.Entries[i].Detail,
			Stack:  r.Entries[i].Stack,
			Debug:  r.Entries[i].Debug,
		}
	}

	if err = m.fac.db.Save(nil, m); err != nil {
		return ErrInsert.WithReason(err)
	}

	return nil
}

func (m *fdbModel) Export() (r *Report) {
	r = &Report{
		ID:      m.ID,
		Code:    m.Code,
		Link:    m.Link,
		Title:   m.Title,
		Status:  m.Status,
		Created: m.Created,
		Entries: make([]*ReportEntry, len(m.Errx)),
	}

	for i := range m.Errx {
		r.Entries[i] = &ReportEntry{
			Text:   m.Errx[i].Text,
			Detail: m.Errx[i].Detail,
			Stack:  m.Errx[i].Stack,
			Debug:  m.Errx[i].Debug,
		}
	}

	return r
}

func (m *fdbModel) ExportRFC() *RFC {
	v := &RFC{
		ID:      m.ID,
		Code:    m.Code,
		Link:    m.Link,
		Title:   m.Title,
		Status:  m.Status,
		Created: m.Created.Format(time.RFC3339Nano),
	}

	if v.Link == "" {
		v.Link = BlankLink
	}

	parts := make([]string, 0, len(m.Errx))
	for i := len(m.Errx) - 1; i >= 0; i-- {
		if m.Errx[i].Detail != "" {
			parts = append(parts, m.Errx[i].Detail)
		}
	}

	v.Detail = strings.Join(parts, " => ")
	return v
}

func (m *fdbModel) ExportMonitoring() *ViewMonitoring {
	v := &ViewMonitoring{
		ID:      m.ID,
		Code:    m.Code,
		Link:    m.Link,
		Title:   m.Title,
		Status:  int(m.Status),
		Created: m.Created,
	}

	v.Entries = make([]*ViewMonitoringEntry, len(m.Errx))

	for i := range v.Entries {
		v.Entries[i] = &ViewMonitoringEntry{
			Text:   m.Errx[i].Text,
			Detail: m.Errx[i].Detail,
			Stack:  m.Errx[i].Stack,
			Debug:  m.Errx[i].Debug,
		}
	}

	return v
}

func (m *fdbModel) FdbxID() string { return m.ID }
func (m *fdbModel) FdbxType() fdbx.RecordType {
	return fdbx.RecordType{
		ID:  ModelCrash,
		Ver: verCrashV1,
		New: m.fac.newRecord,
	}
}
func (m *fdbModel) FdbxIndex(idx fdbx.Indexer) error {
	idx.Index(IndexCrashCreated, Unix(m.Created))
	idx.Index(IndexCrashCode, fdbx.S2B(strings.ToLower(m.Code)))
	idx.Index(IndexCrashMessage, fdbx.S2B(strings.ToLower(m.Title)))

	for i := range m.Errx {
		idx.Index(IndexCrashMessage, fdbx.S2B(strings.ToLower(m.Errx[i].Text)))
	}

	return nil
}
func (m *fdbModel) FdbxMarshal() ([]byte, error)   { return json.Marshal(m) }
func (m *fdbModel) FdbxUnmarshal(buf []byte) error { return json.Unmarshal(buf, m) }

// Intersect - пересечение двух списков айдишек
func Intersect(ids1, ids2 []string) []string {
	var ok bool
	var short, long []string

	if len(ids1) == 0 || len(ids2) == 0 {
		return nil
	}

	if len(ids1) < len(ids2) {
		short = ids1
		long = ids2
	} else {
		short = ids2
		long = ids1
	}

	join := make([]string, 0, len(short))
	index := make(map[string]struct{}, len(short))

	for i := range short {
		index[short[i]] = struct{}{}
	}

	for i := range long {
		if _, ok = index[long[i]]; ok {
			join = append(join, long[i])
		}
	}

	return join
}

func Unix(t time.Time) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(t.UTC().UnixNano()))
	return buf[:]
}
