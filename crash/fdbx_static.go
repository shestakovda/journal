package crash

import (
	"context"

	"github.com/shestakovda/fdbx/v2/db"
	"github.com/shestakovda/fdbx/v2/orm"
)

func Autovacuum(ctx context.Context, dbc db.Connection, crashID uint16, vcPack int, opts ...orm.Option) {
	go orm.NewTable(crashID, orm.BatchIndex(idxCrash)).Autovacuum(ctx, dbc, vcPack, opts...)
}

func Vacuum(dbc db.Connection, crashID uint16, vcPack int) error {
	return orm.NewTable(crashID, orm.BatchIndex(idxCrash)).Vacuum(dbc, vcPack)
}
