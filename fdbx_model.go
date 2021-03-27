package journal

import (
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/shestakovda/fdbx/v2"
	"github.com/shestakovda/typex"

	"github.com/shestakovda/journal/crash"
	"github.com/shestakovda/journal/models"
)

func newFdbxModel(fac *fdbxFactory) *fdbxModel {
	return &fdbxModel{
		fac: fac,
	}
}

func loadFdbxModel(fac *fdbxFactory, uid typex.UUID, buf []byte) *fdbxModel {
	obj := models.GetRootAsFdbxJournal(buf, 0).UnPack()

	mod := &fdbxModel{
		uid:   uid,
		fac:   fac,
		sid:   obj.Service,
		host:  obj.Host,
		start: time.Unix(0, obj.Start).UTC(),
		total: time.Duration(obj.Total),
		chain: make([]*fdbxStage, len(obj.Chain)),
	}

	for i := range obj.Chain {
		mod.chain[i] = loadFdbxStage(obj.Chain[i])
	}

	return mod
}

type fdbxModel struct {
	uid   typex.UUID
	sid   string
	host  string
	start time.Time
	total time.Duration
	chain []*fdbxStage

	fac *fdbxFactory
}

func (m *fdbxModel) Import(e *Entry) (err error) {
	var reps []*crash.Report

	if reps, err = m.setEntry(e); err != nil {
		return
	}

	if len(reps) > 0 {
		if err = m.fac.crf.ImportReports(reps...); err != nil {
			return ErrInsert.WithReason(err)
		}
	}

	return m.save()
}

func (m *fdbxModel) Export(withCrash bool) (e *Entry, err error) {
	var crm crash.Model

	e = &Entry{
		ID:      m.uid.Hex(),
		Host:    m.host,
		Service: m.sid,
		Start:   m.start,
		Total:   m.total,
		Chain:   make([]*Stage, len(m.chain)),
	}

	for i := range m.chain {
		e.Chain[i] = m.chain[i].Export()

		if withCrash && e.Chain[i].Type == ModelTypeCrash.ID() {
			if crm, err = m.fac.crf.ByID(e.Chain[i].EnID); err != nil {
				return nil, ErrSelect.WithReason(err)
			}

			e.Chain[i].Fail = crm.Export()
		}
	}

	return e, nil
}

func (m *fdbxModel) ExportAPI(_ Provider) (a *API) {
	a = &API{
		ID:     m.uid.Hex(),
		Start:  m.start,
		Total:  m.total,
		Stages: make([]*StageAPI, len(m.chain)),
	}

	for i := range m.chain {
		a.Stages[i] = m.chain[i].ExportAPI()

		if i == 0 {
			a.Name = a.Stages[i].Name
		}
	}

	return a
}

func (m *fdbxModel) ExportMonitoring(_ Provider) (v *ViewMonitoring) {
	v = &ViewMonitoring{
		ID:      m.uid.Hex(),
		Host:    m.host,
		Service: m.sid,
		Start:   m.start,
		Total:   m.total.String(),
		Time:    uint64(m.total),
		Stages:  make([]*StageMonitoring, len(m.chain)),
	}

	for i := range m.chain {
		v.Stages[i] = m.chain[i].ExportMonitoring()

		if i == 0 {
			v.Name = v.Stages[i].Name
		}
	}

	return v
}

func (m *fdbxModel) setEntry(e *Entry) (reps []*crash.Report, err error) {
	if m.uid, err = typex.ParseUUID(e.ID); err != nil {
		return nil, ErrValidate.WithReason(err).WithDetail("Некорректный формат идентификатора")
	}

	m.sid = e.Service
	m.host = e.Host
	m.total = e.Total
	m.start = e.Start.UTC()
	m.chain = make([]*fdbxStage, len(e.Chain))

	reps = make([]*crash.Report, 0, len(e.Chain))
	for i := range e.Chain {
		m.chain[i] = newFdbxStage(e.Chain[i])

		if e.Chain[i].Fail != nil {
			e.Chain[i].Fail.Debug = e.Debug // прокидываем дебаг для ошибки
			reps = append(reps, e.Chain[i].Fail)
		}
	}

	return reps, nil
}

func (m *fdbxModel) pair() fdb.KeyValue {
	obj := &models.FdbxJournalT{
		Host:    m.host,
		Service: m.sid,
		Total:   int64(m.total),
		Start:   m.start.UnixNano(),
		Chain:   make([]*models.FdbxStageT, len(m.chain)),
	}

	for i := range m.chain {
		obj.Chain[i] = m.chain[i].dump()
	}

	return fdb.KeyValue{Key: fdb.Key(m.uid), Value: fdbx.FlatPack(obj)}
}

func (m *fdbxModel) save() (err error) {
	if err = m.fac.tbl.Upsert(m.fac.tx, m.pair()); err != nil {
		return ErrInsert.WithReason(err)
	}

	return nil
}
