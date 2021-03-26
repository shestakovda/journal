package crash

import (
	"strings"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/shestakovda/fdbx/v2"
	"github.com/shestakovda/journal/models"
	"github.com/shestakovda/typex"
)

func newFdbxModel(fac *fdbxFactory) *fdbxModel {
	return &fdbxModel{
		fac: fac,
		uid: typex.NewUUID(),
	}
}

func loadFdbxModel(fac *fdbxFactory, uid typex.UUID, buf []byte) *fdbxModel {
	obj := models.GetRootAsFdbxCrash(buf, 0).UnPack()

	mod := &fdbxModel{
		uid:     uid,
		code:    obj.Code,
		link:    obj.Link,
		title:   obj.Title,
		status:  obj.Status,
		created: time.Unix(0, obj.Created).UTC(),
		steps:   make([]*fdbxStep, len(obj.Steps)),

		fac: fac,
	}

	for i := range obj.Steps {
		mod.steps[i] = fdbxLoadStep(obj.Steps[i])
	}

	if len(obj.Debug) != 0 {
		mod.debug = make(map[string]string, len(obj.Debug))

		for i := range obj.Debug {
			mod.debug[obj.Debug[i].Name] = obj.Debug[i].Text
		}
	}

	return mod
}

type fdbxModel struct {
	uid     typex.UUID
	code    string
	link    string
	title   string
	status  uint16
	created time.Time
	steps   []*fdbxStep
	debug   map[string]string

	fac *fdbxFactory
}

func (m *fdbxModel) Import(r *Report) (err error) {
	if err = m.setReport(r); err != nil {
		return
	}

	return m.save()
}

func (m *fdbxModel) Export() *Report {
	res := &Report{
		ID:      m.uid.Hex(),
		Code:    m.code,
		Link:    m.link,
		Title:   m.title,
		Debug:   m.debug,
		Status:  m.status,
		Created: m.created,
		Entries: make([]*ReportEntry, len(m.steps)),
	}

	for i := range m.steps {
		res.Entries[i] = m.steps[i].Export()
	}

	return res
}

func (m *fdbxModel) ExportMonitoring() *ViewMonitoring {
	res := &ViewMonitoring{
		ID:      m.uid.Hex(),
		Code:    m.code,
		Link:    m.link,
		Title:   m.title,
		Debug:   m.debug,
		Status:  int(m.status),
		Created: m.created,
		Entries: make([]*ViewMonitoringEntry, len(m.steps)),
	}

	for i := range m.steps {
		res.Entries[i] = m.steps[i].ExportMonitoring()
	}

	return res
}

func (m *fdbxModel) ExportRFC() *RFC {
	res := &RFC{
		ID:      m.uid.Hex(),
		Code:    m.code,
		Link:    m.link,
		Title:   m.title,
		Status:  m.status,
		Created: m.created.Format(time.RFC3339Nano),
	}

	if res.Link == "" {
		res.Link = BlankLink
	}

	parts := make([]string, 0, len(m.steps))
	for i := len(m.steps) - 1; i >= 0; i-- {
		if m.steps[i].detail != "" {
			parts = append(parts, m.steps[i].detail)
		}
	}

	res.Detail = strings.Join(parts, " => ")
	return res
}

func (m *fdbxModel) setReport(r *Report) (err error) {
	if m.uid, err = typex.ParseUUID(r.ID); err != nil {
		return ErrIDValidate.WithReason(err)
	}

	m.code = r.Code
	m.link = r.Link
	m.title = r.Title
	m.status = r.Status
	m.debug = r.Debug
	m.created = r.Created.UTC()
	m.steps = make([]*fdbxStep, len(r.Entries))

	for i := range r.Entries {
		m.steps[i] = fdbxNewStep(r.Entries[i])
	}

	return nil
}

func (m *fdbxModel) pair() fdb.KeyValue {
	obj := &models.FdbxCrashT{
		Code:   m.code,
		Link:   m.link,
		Title:  m.title,
		Status: m.status,
		Steps:  make([]*models.FdbxStepT, len(m.steps)),
	}

	if !m.created.IsZero() {
		obj.Created = m.created.UTC().UnixNano()
	}

	for i := range m.steps {
		obj.Steps[i] = m.steps[i].dump()
	}

	if len(m.debug) != 0 {
		obj.Debug = make([]*models.FdbxDebugT, 0, len(m.debug))

		for i := range m.debug {
			obj.Debug = append(obj.Debug, &models.FdbxDebugT{
				Name: i,
				Text: m.debug[i],
			})
		}
	}

	return fdb.KeyValue{Key: fdb.Key(m.uid), Value: fdbx.FlatPack(obj)}
}

func (m *fdbxModel) save() (err error) {
	if err = m.fac.tbl.Upsert(m.fac.tx, m.pair()); err != nil {
		return ErrInsert.WithReason(err)
	}

	return nil
}
