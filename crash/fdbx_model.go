package crash

import (
	"strings"
	"time"

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

	fac *fdbxFactory
}

func (m *fdbxModel) Import(r *Report) (err error) {
	if m.uid, err = typex.ParseUUID(r.ID); err != nil {
		return ErrIDValidate.WithReason(err)
	}

	m.code = r.Code
	m.link = r.Link
	m.title = r.Title
	m.status = r.Status
	m.created = r.Created.UTC()
	m.steps = make([]*fdbxStep, len(r.Entries))

	for i := range r.Entries {
		m.steps[i] = fdbxNewStep(r.Entries[i])
	}

	return m.save()
}

func (m *fdbxModel) Export() *Report {
	res := &Report{
		ID:      m.uid.Hex(),
		Code:    m.code,
		Link:    m.link,
		Title:   m.title,
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

func (m *fdbxModel) save() (err error) {
	obj := &models.FdbxCrashT{
		Code:    m.code,
		Link:    m.link,
		Title:   m.title,
		Status:  m.status,
		Created: m.created.UTC().UnixNano(),
		Steps:   make([]*models.FdbxStepT, len(m.steps)),
	}

	for i := range m.steps {
		obj.Steps[i] = m.steps[i].dump()
	}

	if err = m.fac.tbl.Upsert(m.fac.tx, fdbx.NewPair(fdbx.Bytes2Key(m.uid), fdbx.FlatPack(obj))); err != nil {
		return ErrInsert.WithReason(err)
	}

	return nil
}
