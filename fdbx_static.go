package journal

import (
	"context"

	"github.com/shestakovda/fdbx/v2/db"
	"github.com/shestakovda/fdbx/v2/orm"

	"github.com/shestakovda/journal/crash"
)

//goland:noinspection GoUnusedExportedFunction
func Autovacuum(ctx context.Context, dbc db.Connection, journalID, crashID uint16) {
	go orm.NewTable(journalID, orm.BatchIndex(idxJournal)).Autovacuum(ctx, dbc)
	crash.Autovacuum(ctx, dbc, crashID)
}
