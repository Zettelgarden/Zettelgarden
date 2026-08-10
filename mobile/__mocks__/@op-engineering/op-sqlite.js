/**
 * Jest mock for @op-engineering/op-sqlite: the native module cannot load in
 * jest. open() returns a fake DB with jest.fn()s; every created instance is
 * recorded on __instances and results are queued via __pushResult so tests
 * can script executor responses before a command opens the DB. Real SQL
 * semantics are exercised in sync-engine's adapter matrix (a better-sqlite3
 * loopback stands in for the RN executor).
 */

const __instances = [];
const __nextResults = [];

function makeDb() {
  const db = {
    execute: jest.fn(async () => {
      if (__nextResults.length > 0) {
        return __nextResults.shift();
      }
      return { rowsAffected: 0, rows: [] };
    }),
    close: jest.fn(),
    delete: jest.fn(),
  };
  __instances.push(db);
  return db;
}

const open = jest.fn(makeDb);

module.exports = {
  open,
  __instances,
  __pushResult: result => {
    __nextResults.push(result);
  },
};
