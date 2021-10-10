package journal

import (
	"encoding/binary"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/shestakovda/errx"
	"github.com/shestakovda/fdbx/v2"
	"github.com/shestakovda/fdbx/v2/mvcc"
	"github.com/shestakovda/fdbx/v2/orm"
	"github.com/shestakovda/typex"

	"github.com/shestakovda/journal/crash"
)

func newFdbxFactory(tx mvcc.Tx, journalID, crashID uint16) *fdbxFactory {
	return &fdbxFactory{
		tx:  tx,
		crf: crash.NewFdbxFactory(tx, crashID),
		tbl: orm.NewTable(journalID, orm.BatchIndex(idxJournal)),
	}
}

type fdbxFactory struct {
	tx  mvcc.Tx
	tbl orm.Table
	crf crash.Factory
}

func (f *fdbxFactory) New() Model {
	return newFdbxModel(f)
}

func (f *fdbxFactory) ByID(id string) (_ Model, err error) {
	var row fdb.KeyValue
	var uid typex.UUID

	if uid, err = typex.ParseUUID(id); err != nil {
		return nil, ErrValidate.WithReason(err).WithDetail("Некорректный формат идентификатора")
	}

	if row, err = f.tbl.Select(f.tx).ByID(fdb.Key(uid)).First(); err != nil {
		dbg := errx.Debug{"ID": uid.Hex()}

		if errx.Is(err, orm.ErrNotFound) {
			return nil, ErrNotFound.WithReason(err).WithDebug(dbg)
		}

		return nil, ErrSelect.WithReason(err).WithDebug(dbg)
	}

	return loadFdbxModel(f, uid, row.Value), nil
}

func (f *fdbxFactory) ByModel(mtp ModelType, mid string) (res []Model, err error) {
	var rows []fdb.KeyValue

	entp := make([]byte, 4)
	binary.BigEndian.PutUint32(entp, uint32(mtp.ID()))
	query := fdbx.AppendRight(entp, []byte(mid)...)

	if rows, err = f.tbl.Select(f.tx).ByIndex(IndexModel, query).All(); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"Тип модели":    mtp.String(),
			"Идентификатор": mid,
		})
	}

	res = make([]Model, len(rows))
	for i := range rows {
		res[i] = loadFdbxModel(f, typex.UUID(rows[i].Key), rows[i].Value)
	}

	return res, nil
}

func (f *fdbxFactory) Cursor(id string) (Cursor, error) {
	return loadFdbxCursor(f, id)
}

func (f *fdbxFactory) ByDate(from, last time.Time, page uint, _ ...string) (_ Cursor, err error) {
	var qid string

	que := f.tbl.Select(f.tx).Page(int(page)).Reverse().ByIndexRange(
		IndexStart,
		fdbx.Time2Byte(from),
		fdbx.Time2Byte(last),
	)

	if qid, err = que.Save(); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"От момента":      from.UTC().Format(time.RFC3339Nano),
			"До момента":      last.UTC().Format(time.RFC3339Nano),
			"Размер страницы": page,
		})
	}

	return newFdbxCursor(f, qid, que), nil
}

func (f *fdbxFactory) ByDateSortable(from, to time.Time, page uint, desc bool) (_ Cursor, err error) {
	var qid string

	que := f.tbl.Select(f.tx).Page(int(page)).ByIndexRange(
		IndexStart,
		fdbx.Time2Byte(from),
		fdbx.Time2Byte(to),
	)

	if desc {
		que.Reverse()
	}

	if qid, err = que.Save(); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"От момента":      from.UTC().Format(time.RFC3339Nano),
			"До момента":      to.UTC().Format(time.RFC3339Nano),
			"Размер страницы": page,
		})
	}

	return newFdbxCursor(f, qid, que), nil
}

func (f *fdbxFactory) ByModelDate(
	mtp ModelType,
	mid string,
	from time.Time,
	last time.Time,
	page uint,
	_ ...string,
) (_ Cursor, err error) {
	var qid string

	entp := make([]byte, 4)
	binary.BigEndian.PutUint32(entp, uint32(mtp.ID()))

	key := fdbx.AppendRight(entp, []byte(mid)...)
	que := f.tbl.Select(f.tx).Page(int(page)).Reverse().ByIndexRange(
		IndexModel,
		fdbx.AppendRight(key, fdbx.Time2Byte(from)...),
		fdbx.AppendRight(key, fdbx.Time2Byte(last)...),
	)

	if qid, err = que.Save(); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"Тип модели":      mtp.String(),
			"Идентификатор":   mid,
			"От момента":      from.UTC().Format(time.RFC3339Nano),
			"До момента":      last.UTC().Format(time.RFC3339Nano),
			"Размер страницы": page,
		})
	}

	return newFdbxCursor(f, qid, que), nil
}

func (f *fdbxFactory) ImportEntries(entries ...*Entry) (err error) {
	var mod *fdbxModel
	var reps []*crash.Report

	rows := make([]fdb.KeyValue, len(entries))
	fails := make([]*crash.Report, 0, len(entries))

	for i := range entries {
		mod = newFdbxModel(f)

		if reps, err = mod.setEntry(entries[i]); err != nil {
			return
		}

		rows[i] = mod.pair()
		fails = append(fails, reps...)
	}

	if err = f.crf.ImportReports(fails...); err != nil {
		return ErrInsert.WithReason(err)
	}

	if err = f.tbl.Upsert(f.tx, rows...); err != nil {
		return ErrInsert.WithReason(err)
	}

	return nil
}

func (f *fdbxFactory) Delete(ids ...string) (err error) {
	if len(ids) == 0 {
		return nil
	}

	keys := make([]fdb.Key, 0, len(ids))
	var uid typex.UUID

	for i := range ids {
		if uid, err = typex.ParseUUID(ids[i]); err != nil {
			return ErrValidate.WithReason(err).WithDetail("Некорректный формат идентификатора")
		}

		keys = append(keys, fdb.Key(uid))
	}

	if err = f.tbl.Delete(f.tx, keys...); err != nil {
		return ErrDelete.WithReason(err)
	}

	return nil
}
