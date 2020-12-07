package journal

import (
	"github.com/shestakovda/fdbx/v2/db"
	"github.com/shestakovda/fdbx/v2/mvcc"
)

func newFdbxDriver(dbc db.Connection, journalID, crashID uint16) *fdbxDriver {
	return &fdbxDriver{
		dbc: dbc,
		cid: crashID,
		jid: journalID,
	}
}

type fdbxDriver struct {
	cid uint16
	jid uint16
	dbc db.Connection
}

func (d fdbxDriver) InsertEntry(e *Entry) (err error) {
	var tx mvcc.Tx

	if tx, err = mvcc.Begin(d.dbc); err != nil {
		return ErrInsert.WithReason(err)
	}
	defer tx.Cancel()

	if err = newFdbxFactory(tx, d.jid, d.cid).New().Import(e); err != nil {
		return ErrInsert.WithReason(err)
	}

	if err = tx.Commit(); err != nil {
		return ErrInsert.WithReason(err)
	}

	return nil
}
