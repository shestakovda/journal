package journal

import (
	"github.com/shestakovda/errx"
	"github.com/shestakovda/fdbx/v2"
	"github.com/shestakovda/fdbx/v2/orm"
	"github.com/shestakovda/typex"
)

func newFdbxCursor(fac *fdbxFactory, qid string, que orm.Query) *fdbxCursor {
	return &fdbxCursor{
		qid: qid,
		que: que,
		fac: fac,
	}
}

func loadFdbxCursor(fac *fdbxFactory, qid string) (cur *fdbxCursor, err error) {
	cur = &fdbxCursor{
		qid: qid,
		fac: fac,
	}

	if cur.que, err = fac.tbl.Cursor(fac.tx, qid); err != nil {
		if errx.Is(err, orm.ErrNotFound) {
			return nil, errx.ErrNotFound.WithReason(err).WithDebug(errx.Debug{
				"Курсор": qid,
			})
		}

		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"Курсор": qid,
		})
	}

	return cur, nil
}

type fdbxCursor struct {
	qid string
	que orm.Query
	fac *fdbxFactory
}

func (c *fdbxCursor) ID() string {
	return c.qid
}

func (c *fdbxCursor) Empty() bool {
	return c.que.Empty()
}

func (c *fdbxCursor) NextPage(size uint, services ...string) (res []Model, err error) {
	var rows []fdbx.Pair

	if len(services) > 0 {
		c.que = c.que.Where(filterByService(services))
	}

	if rows, err = c.que.Page(int(size)).Next(); err != nil {
		return nil, errx.ErrInternal.WithReason(err).WithDebug(errx.Debug{
			"Сервисы": services,
		})
	}

	res = make([]Model, len(rows))
	for i := range rows {
		res[i] = loadFdbxModel(c.fac, typex.UUID(rows[i].Key().Bytes()), rows[i].Value())
	}

	return res, nil
}
