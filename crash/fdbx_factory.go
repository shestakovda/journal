package crash

import (
	"time"

	"github.com/shestakovda/errx"
	"github.com/shestakovda/fdbx/v2"
	"github.com/shestakovda/fdbx/v2/mvcc"
	"github.com/shestakovda/fdbx/v2/orm"
	"github.com/shestakovda/typex"
)

func newFdbxFactory(tx mvcc.Tx, crashID uint16) *fdbxFactory {
	return &fdbxFactory{
		tx:  tx,
		tbl: orm.NewTable(crashID, orm.BatchIndex(idxCrash)),
	}
}

type fdbxFactory struct {
	tx  mvcc.Tx
	tbl orm.Table
}

func (f *fdbxFactory) New() Model {
	return newFdbxModel(f)
}

func (f *fdbxFactory) ByID(id string) (_ Model, err error) {
	var row fdbx.Pair
	var uid typex.UUID

	if uid, err = typex.ParseUUID(id); err != nil {
		return nil, ErrIDValidate.WithReason(err)
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

func (f *fdbxFactory) ByDateCode(from, last time.Time, code string) (res []Model, err error) {
	var rows []fdbx.Pair

	if code != "" {
		pref := fdbx.String2Key(code)
		rows, err = f.tbl.Select(f.tx).ByIndexRange(
			IndexCode,
			pref.RPart(fdbx.Time2Byte(from)...),
			pref.RPart(fdbx.Time2Byte(last)...),
		).All()
	} else {
		rows, err = f.tbl.Select(f.tx).ByIndexRange(
			IndexDate,
			fdbx.Bytes2Key(fdbx.Time2Byte(from)),
			fdbx.Bytes2Key(fdbx.Time2Byte(last)),
		).All()
	}

	res = make([]Model, len(rows))
	for i := range rows {
		res[i] = loadFdbxModel(f, typex.UUID(rows[i].Key().Bytes()), rows[i].Value())
	}

	return res, nil
}
