package services

import (
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"

	"go-backend/server"
)

// sqliteTestSeq makes every freshSQLiteDB call produce a uniquely-named
// in-memory database, so bespoke driver tests never collide — neither with
// each other nor with the shared :memory: DB that tests.Setup() builds when
// the rest of the services suite runs in the same `go test ./services/`
// process.
var sqliteTestSeq uint64

// freshSQLiteDB opens a PRIVATE in-memory SQLite DB (D4 pragmas applied via
// OpenSQLite) for a single test, and registers a t.Cleanup to close it.
//
// server.OpenSQLite(":memory:") uses SQLite shared cache
// (file::memory:?cache=shared), which means every caller in the process gets
// the SAME in-memory database. That is desirable for tests.Setup() — the whole
// suite shares one schema + fixtures via tx-per-test rollback — but bespoke
// driver tests that apply their own schema would then hit "table already
// exists" on the second schema load. freshSQLiteDB sidesteps this by minting a
// unique database name (file:<name>?mode=memory&cache=shared): modernc treats
// each distinct name as its own shared-cache in-memory DB, isolated from every
// other name and from the bare :memory: DB.
func freshSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	n := atomic.AddUint64(&sqliteTestSeq, 1)
	name := fmt.Sprintf("svc_sqlite_test_%d", n)
	db, err := server.OpenSQLite("file:" + name + "?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("freshSQLiteDB open %s: %v", name, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
