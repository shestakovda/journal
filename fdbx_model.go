package journal

import (
	"time"

	"github.com/shestakovda/fdbx/v2"
	"github.com/shestakovda/journal/crash"
	"github.com/shestakovda/journal/models"
	"github.com/shestakovda/typex"
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
	start time.Time
	total time.Duration
	chain []*fdbxStage

	fac *fdbxFactory
}

func (m *fdbxModel) Import(e *Entry) (err error) {
	if m.uid, err = typex.ParseUUID(e.ID); err != nil {
		return ErrValidate.WithReason(err).WithDetail("Некорректный формат идентификатора")
	}

	m.sid = e.Service
	m.total = e.Total
	m.start = e.Start.UTC()
	m.chain = make([]*fdbxStage, len(e.Chain))

	for i := range e.Chain {
		m.chain[i] = newFdbxStage(e.Chain[i])

		if e.Chain[i].Fail != nil {
			if err = m.fac.crf.New().Import(e.Chain[i].Fail); err != nil {
				return ErrInsert.WithReason(err)
			}
		}
	}

	return m.save()
}

func (m *fdbxModel) Export(withCrash bool) (e *Entry, err error) {
	var crm crash.Model

	e = &Entry{
		ID:      m.uid.Hex(),
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

func (m *fdbxModel) ExportAPI(log Provider) (a *API) {
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

func (m *fdbxModel) ExportMonitoring(log Provider) (v *ViewMonitoring) {
	v = &ViewMonitoring{
		ID:      m.uid.Hex(),
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

func (m *fdbxModel) save() (err error) {
	obj := &models.FdbxJournalT{
		Service: m.sid,
		Total:   int64(m.total),
		Start:   m.start.UnixNano(),
		Chain:   make([]*models.FdbxStageT, len(m.chain)),
	}

	for i := range m.chain {
		obj.Chain[i] = m.chain[i].dump()
	}

	if err = m.fac.tbl.Upsert(m.fac.tx, fdbx.NewPair(fdbx.Bytes2Key(m.uid), fdbx.FlatPack(obj))); err != nil {
		return ErrInsert.WithReason(err)
	}

	return nil
}
