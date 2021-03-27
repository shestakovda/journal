package crash

import (
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"

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
	var row fdb.KeyValue
	var uid typex.UUID

	if uid, err = typex.ParseUUID(id); err != nil {
		return nil, ErrIDValidate.WithReason(err)
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

func (f *fdbxFactory) ByDateCode(from, last time.Time, code string) (res []Model, err error) {
	var rows []fdb.KeyValue

	if code != "" {
		pref := fdb.Key(code)
		rows, err = f.tbl.Select(f.tx).ByIndexRange(
			IndexCode,
			fdbx.AppendRight(pref, fdbx.Time2Byte(from)...),
			fdbx.AppendRight(pref, fdbx.Time2Byte(last)...),
		).All()
	} else {
		rows, err = f.tbl.Select(f.tx).ByIndexRange(
			IndexDate,
			fdbx.Time2Byte(from),
			fdbx.Time2Byte(last),
		).All()
	}
	if err != nil {
		return nil, ErrSelect.WithReason(err)
	}

	res = make([]Model, len(rows))
	for i := range rows {
		res[i] = loadFdbxModel(f, typex.UUID(rows[i].Key), rows[i].Value)
	}

	return res, nil
}

func (f *fdbxFactory) ImportReports(reports ...*Report) (err error) {
	var mod *fdbxModel

	rows := make([]fdb.KeyValue, len(reports))

	for i := range reports {
		mod = newFdbxModel(f)

		if err = mod.setReport(reports[i]); err != nil {
			return
		}

		rows[i] = mod.pair()
	}

	if err = f.tbl.Upsert(f.tx, rows...); err != nil {
		return ErrInsert.WithReason(err)
	}

	return nil
}
