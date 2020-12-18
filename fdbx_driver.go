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
	if err = mvcc.WithTx(d.dbc, func(tx mvcc.Tx) error {
		return newFdbxFactory(tx, d.jid, d.cid).New().Import(e)
	}); err != nil {
		return ErrInsert.WithReason(err)
	}

	return nil
}
