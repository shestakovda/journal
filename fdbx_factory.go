package journal

import (
	"encoding/binary"
	"time"

	"github.com/shestakovda/errx"
	"github.com/shestakovda/fdbx/v2"
	"github.com/shestakovda/fdbx/v2/mvcc"
	"github.com/shestakovda/fdbx/v2/orm"
	"github.com/shestakovda/journal/crash"
	"github.com/shestakovda/typex"
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
	var row fdbx.Pair
	var uid typex.UUID

	if uid, err = typex.ParseUUID(id); err != nil {
		return nil, ErrValidate.WithReason(err).WithDetail("Некорректный формат идентификатора")
	}

	if row, err = f.tbl.Select(f.tx).ByID(fdbx.Bytes2Key(uid)).First(); err != nil {
		dbg := errx.Debug{"ID": uid.Hex()}

		if errx.Is(err, orm.ErrNotFound) {
			return nil, ErrNotFound.WithReason(err).WithDebug(dbg)
		}

		return nil, ErrSelect.WithReason(err).WithDebug(dbg)
	}

	return loadFdbxModel(f, uid, row.Value()), nil
}

func (f *fdbxFactory) ByModel(mtp ModelType, mid string) (res []Model, err error) {
	var rows []fdbx.Pair

	entp := make([]byte, 4)
	binary.BigEndian.PutUint32(entp, uint32(mtp.ID()))
	query := fdbx.String2Key(mid).LPart(entp...)

	if rows, err = f.tbl.Select(f.tx).ByIndex(IndexModel, query).All(); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"Тип модели":    mtp.String(),
			"Идентификатор": mid,
		})
	}

	res = make([]Model, len(rows))
	for i := range rows {
		res[i] = loadFdbxModel(f, typex.UUID(rows[i].Key().Bytes()), rows[i].Value())
	}

	return res, nil
}

func (f *fdbxFactory) Cursor(id string) (Cursor, error) {
	return loadFdbxCursor(f, id)
}

func (f *fdbxFactory) ByDate(from, last time.Time, page uint, services ...string) (_ Cursor, err error) {
	var qid string

	que := f.tbl.Select(f.tx).Page(int(page)).Reverse().ByIndexRange(
		IndexStart,
		fdbx.Bytes2Key(fdbx.Time2Byte(from)),
		fdbx.Bytes2Key(fdbx.Time2Byte(last)),
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

	key := fdbx.String2Key(mid).LPart(entp...)
	que := f.tbl.Select(f.tx).Page(int(page)).Reverse().ByIndexRange(
		IndexModel,
		key.RPart(fdbx.Time2Byte(from)...),
		key.RPart(fdbx.Time2Byte(last)...),
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
