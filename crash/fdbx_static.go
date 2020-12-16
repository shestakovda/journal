package crash

import (
	"context"

	"github.com/shestakovda/fdbx/v2/db"
	"github.com/shestakovda/fdbx/v2/orm"
)

func Autovacuum(ctx context.Context, dbc db.Connection, crashID uint16) {
	go orm.NewTable(crashID, orm.BatchIndex(idxCrash)).Autovacuum(ctx, dbc)
}
