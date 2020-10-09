package journal

import (
	"encoding/binary"
	"strconv"
	"time"

	"github.com/shestakovda/errx"
	"github.com/shestakovda/fdbx"
	"github.com/shestakovda/journal/crash"
	"github.com/shestakovda/journal/models"
	"github.com/shestakovda/typex"

	fbs "github.com/google/flatbuffers/go"
)

/*
	NewDriverFDB - конструктор помощника сохранения журнала

	* fdb - основное подключение к БД
*/
func NewDriverFDB(fdb fdbx.Conn) Driver {
	return &fdbDriver{
		fdb: fdb.At(crash.DatabaseAPI),
	}
}

type fdbDriver struct {
	fdb fdbx.Conn
}

func (d *fdbDriver) InsertEntry(e *Entry) error {
	if err := d.fdb.Tx(func(db fdbx.DB) error { return NewFactoryFDB(d.fdb, db).New().Import(e) }); err != nil {
		return ErrInsert.WithReason(err)
	}
	return nil
}

/*
	NewFactoryFDB - конструктор фабрики моделей в рамках транзакции

	* db - текущий объект транзакции, траслируется в основную базу
*/
func NewFactoryFDB(cn fdbx.Conn, db fdbx.DB) Factory {
	return &fdbFactory{
		db: db.At(crash.DatabaseAPI),
		cn: cn.At(crash.DatabaseAPI),
	}
}

type fdbFactory struct {
	db fdbx.DB
	cn fdbx.Conn
}

func (f *fdbFactory) New() Model {
	return &fdbModel{
		fac: f,
	}
}

func (f *fdbFactory) Cursor(id string) (_ Cursor, err error) {
	var uid typex.UUID

	if uid, err = typex.ParseUUID(id); err != nil {
		return nil, ErrValidate.WithReason(err).WithDetail("Некорректный формат идентификатора")
	}

	res := &fdbCursor{fac: f}

	if res.Cursor, err = f.cn.LoadCursor(uid.Hex(), f.newRecord); err == nil {
		return res, nil
	}

	if errx.Is(err, fdbx.ErrRecordNotFound) {
		return nil, errx.ErrNotFound.WithReason(err).WithDebug(errx.Debug{
			"Курсор": uid.Hex(),
		})
	}

	return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
		"Курсор": uid.Hex(),
	})
}

func (f *fdbFactory) ByID(id string) (_ Model, err error) {
	var uid typex.UUID

	if uid, err = typex.ParseUUID(id); err != nil {
		return nil, ErrValidate.WithReason(err).WithDetail("Некорректный формат идентификатора")
	}

	m := &fdbModel{id: uid, fac: f}

	if err = f.db.Load(nil, m); err != nil {
		dbg := errx.Debug{"ID": m.id}

		if errx.Is(err, fdbx.ErrRecordNotFound) {
			return nil, ErrNotFound.WithReason(err).WithDebug(dbg)
		}

		return nil, ErrSelect.WithReason(err).WithDebug(dbg)
	}

	return m, nil
}

func (f *fdbFactory) ByModel(mtp ModelType, mid string) (res []Model, err error) {
	var recs []fdbx.Record

	query := make([]byte, 2+len(mid))
	binary.BigEndian.PutUint16(query[:2], uint16(mtp.ID()))
	copy(query[2:], fdbx.S2B(mid))

	rtp := fdbx.RecordType{ID: IndexJournalEntity, Ver: verJournalV1, New: f.newRecord}

	if recs, err = f.db.Select(rtp, fdbx.Query(query)); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"Тип модели":    mtp.String(),
			"Идентификатор": mid,
		})
	}

	return f.recs2list(recs), nil
}

func (f *fdbFactory) ByDate(from, to time.Time, page uint, services ...string) (_ Cursor, err error) {
	res := &fdbCursor{fac: f}
	rtp := fdbx.RecordType{ID: IndexJournalStart, Ver: verJournalV1, New: f.newRecord}
	qto := fdbx.To(crash.Unix(to))
	qfrom := fdbx.From(crash.Unix(from))

	if res.Cursor, err = f.cn.Cursor(rtp, qfrom, qto, fdbx.Reverse(), fdbx.Page(page),
		fdbx.Filter(func(record fdbx.Record) (bool, error) {
			if len(services) != 0 {
				model := record.(*fdbModel)

				for i := range services {
					if model.service == services[i] {
						return true, nil
					}
				}
				return false, nil
			}
			return true, nil
		})); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"От момента":      from.UTC().Format(time.RFC3339Nano),
			"До момента":      to.UTC().Format(time.RFC3339Nano),
			"Размер страницы": page,
		})
	}

	return res, nil
}

func (f *fdbFactory) ByModelDate(mtp ModelType, mid string,
	from, to time.Time, page uint, services ...string) (_ Cursor, err error) {
	var mtpb [2]byte
	binary.BigEndian.PutUint16(mtpb[:], uint16(mtp.ID()))

	res := &fdbCursor{fac: f}
	rtp := fdbx.RecordType{ID: IndexJournalEntity, Ver: verJournalV1, New: f.newRecord}
	qto := fdbx.To(Concat(mtpb[:], fdbx.S2B(mid), crash.Unix(to)))
	qfrom := fdbx.From(Concat(mtpb[:], fdbx.S2B(mid), crash.Unix(from)))

	if res.Cursor, err = f.cn.Cursor(rtp, qfrom, qto, fdbx.Reverse(), fdbx.Page(page),
		fdbx.Filter(func(record fdbx.Record) (bool, error) {
			if len(services) != 0 {
				model := record.(*fdbModel)

				for i := range services {
					if model.service == services[i] {
						return true, nil
					}
				}
				return false, nil
			}
			return true, nil
		})); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"Тип модели":      mtp.String(),
			"Идентификатор":   mid,
			"От момента":      from.UTC().Format(time.RFC3339Nano),
			"До момента":      to.UTC().Format(time.RFC3339Nano),
			"Размер страницы": page,
		})
	}

	return res, nil
}

func (f *fdbFactory) recs2list(recs []fdbx.Record) []Model {
	res := make([]Model, len(recs))
	for i := range recs {
		res[i] = recs[i].(Model)
	}
	return res
}

func (f *fdbFactory) newRecord(ver uint8, id string) (rec fdbx.Record, err error) {
	m := &fdbModel{fac: f}
	return m, m.setID(id)
}

type fdbModel struct {
	id      typex.UUID  // uuid - уникальная метка записи для поиска
	service string      // наименование сервиса инициатора записи
	start   uint64      // unixnano - метка времени старта в UTC
	total   uint64      // наносекунд от старта до завершения
	chain   []*fdbStage // цепочка этапов запроса

	fac *fdbFactory
}

type fdbStage struct {
	wait uint64 // наносекунд от предыдущей метки или от старта
	enTP uint16 // тип модели для поиска по ID и отображения в мониторинге
	verb uint8  // уровень отладки записи, для фильтрации в отображении
	flag uint8  // резерв на будущее и для выравнивания байт
	enID string // идентификатор привязанной сущности, если присутствует
	text string // произвольный текст записи для отображения
}

func (m *fdbModel) Import(e *Entry) (err error) {
	if err = m.setID(e.ID); err != nil {
		return
	}

	m.start = uint64(e.Start.UTC().UnixNano())
	m.total = uint64(e.Total)
	m.chain = make([]*fdbStage, len(e.Chain))
	m.service = e.Service

	for i := range e.Chain {
		m.chain[i] = &fdbStage{
			enID: e.Chain[i].EnID,
			text: e.Chain[i].Text,
			verb: uint8(e.Chain[i].Verb),
			wait: uint64(e.Chain[i].Wait),
			enTP: uint16(e.Chain[i].Type),
		}

		if e.Chain[i].Fail != nil {
			if err = crash.NewFactoryFDB(m.fac.db).New().Import(e.Chain[i].Fail); err != nil {
				return ErrInsert.WithReason(err)
			}
		}
	}

	if err = m.fac.db.Save(nil, m); err != nil {
		return ErrInsert.WithReason(err)
	}

	return nil
}

func (m *fdbModel) Export(withCrash bool) (e *Entry, err error) {
	var crm crash.Model

	e = &Entry{
		ID:      m.id.Hex(),
		Service: m.service,
		Start:   time.Unix(0, int64(m.start)).UTC(),
		Total:   time.Duration(m.total),
		Chain:   make([]*Stage, len(m.chain)),
	}

	for i := range m.chain {
		e.Chain[i] = &Stage{
			EnID: m.chain[i].enID,
			Text: m.chain[i].text,
			Wait: time.Duration(m.chain[i].wait),
			Verb: int(m.chain[i].verb),
			Type: int(m.chain[i].enTP),
		}

		if withCrash && e.Chain[i].Type == crash.ModelTypeCrash {
			if crm, err = crash.NewFactoryFDB(m.fac.db).ByID(e.Chain[i].EnID); err != nil {
				return nil, ErrSelect.WithReason(err)
			}

			e.Chain[i].Fail = crm.Export()
		}
	}

	return e, nil
}

func (m *fdbModel) ExportAPI(log Provider) *API {
	var name string

	stages := make([]*StageAPI, len(m.chain))

	for i := range m.chain {
		if i == 0 {
			name = m.chain[i].text
		}

		if log.V(int(m.chain[i].verb)) {
			stages[i] = &StageAPI{
				Name: m.chain[i].text,
				Wait: time.Duration(m.chain[i].wait),
			}
		}
	}

	return &API{
		ID:     m.id.Hex(),
		Start:  time.Unix(0, int64(m.start)).UTC(),
		Total:  time.Duration(m.total),
		Name:   name,
		Stages: stages,
	}
}

func (m *fdbModel) ExportMonitoring(log Provider) *ViewMonitoring {
	var name string

	stages := make([]*StageMonitoring, len(m.chain))

	for i := range m.chain {
		if i == 0 {
			name = m.chain[i].text
		}

		if log.V(int(m.chain[i].verb)) {
			stages[i] = &StageMonitoring{
				Name: m.chain[i].text,
				Wait: time.Duration(m.chain[i].wait).String(),
			}

			mt := int(m.chain[i].enTP)

			if mt != crash.ModelTypeUnknown {
				stages[i].Type = strconv.Itoa(mt)
				stages[i].EnID = m.chain[i].enID
			}
		}
	}

	return &ViewMonitoring{
		ID:      m.id.Hex(),
		Start:   time.Unix(0, int64(m.start)).UTC(),
		Total:   time.Duration(m.total).String(),
		Name:    name,
		Stages:  stages,
		Service: m.service,
	}
}

func (m *fdbModel) FdbxID() string {
	return m.id.Hex()
}

func (m *fdbModel) FdbxType() fdbx.RecordType {
	return fdbx.RecordType{
		ID:  ModelJournal,
		Ver: verJournalV1,
		New: m.fac.newRecord,
	}
}

func (m *fdbModel) FdbxIndex(idx fdbx.Indexer) error {
	var entp [2]byte

	start := crash.Unix(time.Unix(0, int64(m.start)))

	idx.Grow(8 + 36*len(m.chain))
	idx.Index(IndexJournalStart, start)

	for _, stage := range m.chain {
		if stage.enID != "" {
			binary.BigEndian.PutUint16(entp[:], stage.enTP)
			idx.Index(IndexJournalEntity, Concat(entp[:], fdbx.S2B(stage.enID), start))
		}
	}

	return nil
}

func (m *fdbModel) FdbxMarshal() ([]byte, error) {
	rec := models.JournalT{
		ID:      m.id,
		Start:   m.start,
		Total:   m.total,
		Service: m.service,
		Chain:   make([]*models.StageT, len(m.chain)),
	}

	for i := range m.chain {
		rec.Chain[i] = &models.StageT{
			Wait: m.chain[i].wait,
			Type: m.chain[i].enTP,
			Verb: m.chain[i].verb,
			Flag: m.chain[i].flag,
			EnID: m.chain[i].enID,
			Text: m.chain[i].text,
		}
	}

	buf := fbsPool.Get().(*fbs.Builder)
	buf.Finish(rec.Pack(buf))
	res := buf.FinishedBytes()
	buf.Reset()
	fbsPool.Put(buf)
	return res, nil
}

func (m *fdbModel) FdbxUnmarshal(buf []byte) error {
	rec := models.GetRootAsJournal(buf, 0).UnPack()
	m.id = rec.ID
	m.start = rec.Start
	m.total = rec.Total
	m.service = rec.Service
	m.chain = make([]*fdbStage, len(rec.Chain))

	for i := range rec.Chain {
		m.chain[i] = &fdbStage{
			wait: rec.Chain[i].Wait,
			enTP: rec.Chain[i].Type,
			verb: rec.Chain[i].Verb,
			flag: rec.Chain[i].Flag,
			enID: rec.Chain[i].EnID,
			text: rec.Chain[i].Text,
		}
	}

	return nil
}

func (m *fdbModel) setID(id string) (err error) {
	if m.id, err = typex.ParseUUID(id); err != nil {
		return ErrValidate.WithReason(err).WithDetail("Некорректный формат идентификатора")
	}

	return nil
}

type fdbCursor struct {
	fdbx.Cursor
	fac *fdbFactory
}

func (f fdbCursor) ID() string  { return f.FdbxID() }
func (f fdbCursor) Empty() bool { return f.Cursor.Empty() }
func (f fdbCursor) NextPage(size uint, services ...string) (_ []Model, err error) {
	var recs []fdbx.Record

	if err = f.ApplyOpts(fdbx.Page(size),
		fdbx.Filter(func(record fdbx.Record) (bool, error) {
			if len(services) != 0 {
				model := record.(*fdbModel)

				for i := range services {
					if model.service == services[i] {
						return true, nil
					}
				}
				return false, nil
			}
			return true, nil
		})); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"ID курсора": f.FdbxID(),
		})
	}

	if recs, err = f.Next(f.fac.db, 0); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"ID курсора": f.FdbxID(),
		})
	}

	return f.fac.recs2list(recs), nil
}

func Concat(parts ...[]byte) []byte {
	var size, from, to int

	for i := range parts {
		size += len(parts[i])
	}

	res := make([]byte, size)

	for i := range parts {
		to = from + len(parts[i])
		copy(res[from:to], parts[i])
		from = to
	}

	return res
}
