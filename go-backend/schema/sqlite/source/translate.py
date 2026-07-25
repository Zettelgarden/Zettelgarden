#!/usr/bin/env python3
"""
One-shot translator: Postgres `pg_dump --schema-only` -> consolidated SQLite schema.

Input:  <repo-root>/dev_schema.sql   (pg_dump --schema-only --no-owner --no-privileges)
Output: go-backend/schema/sqlite/schema.sqlite.sql

Translation rules (proven on the cards sub-graph by
services/testdata/spike_cards.sqlite.sql; see migration design doc Phase 2 / D5):

  SERIAL (nextval-backed single-col PK) -> INTEGER PRIMARY KEY AUTOINCREMENT
  composite PK                          -> table-level PRIMARY KEY (...)
  TIMESTAMP[TZ] without/with time zone  -> DATETIME   (NOT TEXT -- modernc needs the
                                                      declared type to return time.Time)
  DATE                                  -> DATE
  BOOLEAN                               -> BOOLEAN    (SQLite NUMERIC affinity 0/1)
  JSONB                                  -> TEXT
  UUID                                   -> TEXT
  TEXT[] / any array                     -> TEXT       (JSON)
  double precision / real                -> REAL
  bigint / smallint / integer            -> INTEGER
  numeric                                -> NUMERIC
  bytea                                  -> BLOB

  DEFAULT now()                          -> DEFAULT (datetime('now'))
  DEFAULT CURRENT_TIMESTAMP              -> DEFAULT (datetime('now'))
  DEFAULT '<x>'::text / ::jsonb / ::character varying -> DEFAULT '<x>'
  DEFAULT gen_random_uuid()              -> SQLite UUIDv4 expression default
  DEFAULT false / true                   -> kept (SQLite accepts; stored 0/1)

  CHECK (col = ANY (ARRAY['a','b']))     -> CHECK (col IN ('a','b'))
  CHECK (... = true/false)               -> ... = 1 / = 0
  CHECK (length(TRIM(BOTH FROM x)) > 0)  -> CHECK (length(trim(x)) > 0)

  FK: REFERENCES public.t(id) [ON DELETE ...] -> REFERENCES t(id) [ON DELETE ...]
  UNIQUE (cols)                          -> CREATE UNIQUE INDEX
  CREATE INDEX ... USING btree (cols)    -> CREATE INDEX ... ON t (cols)
  partial index WHERE (... = false)      -> WHERE (... = 0)

Dropped (handled elsewhere):
  CREATE FUNCTION, CREATE TRIGGER, COMMENT ON, CREATE SEQUENCE,
  ALTER SEQUENCE, ALTER COLUMN ... SET DEFAULT nextval(...),
  the 3 GIN indexes (FTS + JSONB + array containment -- verified dead),
  ALTER TABLE ADD CONSTRAINT PRIMARY KEY/UNIQUE (folded into CREATE TABLE / unique index).

This script is itself a throwaway build tool; the produced schema.sqlite.sql is
the artifact kept under version control.
"""
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))            # .../schema/sqlite/source
OUTDIR = os.path.dirname(HERE)                                # .../schema/sqlite
SRC = os.path.join(HERE, "dev_schema.sql")                    # the pg_dump snapshot
OUT = os.path.join(OUTDIR, "schema.sqlite.sql")               # generated artifact
#
# NOTE on layout: the runtime migration runner (server.RunMigrations) scans
# the TOP LEVEL of schema/sqlite/ for loadable .sql files. Only
# schema.sqlite.sql belongs there; this source/ subdir (the pg_dump input and
# this translator) is excluded because ReadDir is non-recursive, and because
# dev_schema.sql is raw Postgres that would not load on SQLite.

with open(SRC, encoding="utf-8") as f:
    dump = f.read()

# ---------------------------------------------------------------------------
# 1. Slice the dump into sections using pg_dump's `-- Name: ...; Type: X;` markers.
# ---------------------------------------------------------------------------
section_pat = re.compile(
    r"^-- Name: (?P<name>[^;]*); Type: (?P<type>[A-Z _]+); Schema: public;",
    re.MULTILINE,
)
sections = []
for m in section_pat.finditer(dump):
    sections.append((m.group("name").strip(), m.group("type").strip(), m.start()))
bodies: dict[str, list[tuple[str, str]]] = {}
for i, (name, typ, start) in enumerate(sections):
    end = sections[i + 1][2] if i + 1 < len(sections) else len(dump)
    bodies.setdefault(typ, []).append((name, dump[start:end]))

table_order = [n for n, _ in bodies.get("TABLE", [])]
tables_raw = {n: b for n, b in bodies.get("TABLE", [])}

# ---------------------------------------------------------------------------
# 2. Collect PK / UNIQUE / FK constraints.
# ---------------------------------------------------------------------------
pk_cols: dict[str, list[str]] = {}
unique_constraints: dict[str, list[list[str]]] = {}
fks: dict[str, list[str]] = {}

for name, body in bodies.get("CONSTRAINT", []):
    m = re.search(
        r"ALTER TABLE ONLY public\.(\w+)\s+ADD CONSTRAINT \w+ (PRIMARY KEY|UNIQUE) \(([^)]+)\)",
        body,
    )
    if not m:
        continue
    tbl, kind, cols = m.group(1), m.group(2), [c.strip() for c in m.group(3).split(",")]
    if kind == "PRIMARY KEY":
        pk_cols[tbl] = cols
    else:
        unique_constraints.setdefault(tbl, []).append(cols)

for name, body in bodies.get("FK CONSTRAINT", []):
    m = re.search(
        r"ALTER TABLE ONLY public\.(\w+)\s+ADD CONSTRAINT \w+ FOREIGN KEY \(([^)]+)\) "
        r"REFERENCES public\.(\w+)\(([^)]+)\)(.*?)\s*;",
        body,
        re.DOTALL,
    )
    if not m:
        print(f"!! could not parse FK: {name!r}", file=sys.stderr)
        continue
    child, ccols, parent, pcols, tail = m.groups()
    fks.setdefault(child, []).append(
        f"FOREIGN KEY ({ccols.strip()}) REFERENCES {parent}({pcols}){tail.rstrip()}"
    )

# ---------------------------------------------------------------------------
# 3. nextval-backed serial columns.
# ---------------------------------------------------------------------------
serial_cols: dict[str, str] = {}
for name, body in bodies.get("DEFAULT", []):
    m = re.search(
        r"ALTER TABLE ONLY public\.(\w+)\s+ALTER COLUMN (\w+) SET DEFAULT nextval\(", body
    )
    if m:
        serial_cols[m.group(1)] = m.group(2)

# ---------------------------------------------------------------------------
# 4. Default / type / CHECK translation.
# ---------------------------------------------------------------------------
def translate_default(expr: str) -> str:
    e = expr.strip()
    low = e.lower()
    if low in ("now()", "current_timestamp"):
        return "DEFAULT (datetime('now'))"
    if low == "false":
        return "DEFAULT false"
    if low == "true":
        return "DEFAULT true"
    if low.startswith("gen_random_uuid"):
        return ("DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) "
                "|| '-' || lower(hex(randomblob(2))) || '-' || lower(hex(randomblob(2))) "
                "|| '-' || lower(hex(randomblob(6))))")
    m = re.match(r"^'(?P<val>([^'\\]|\\.|'')*)'(?P<rest>::[\w ]+)?$", e)
    if m:
        return f"DEFAULT '{m.group('val')}'"
    return f"DEFAULT {e}"


def translate_type(t: str) -> str:
    low = t.strip().lower()
    if "timestamp" in low:
        return "DATETIME"
    if low == "date":
        return "DATE"
    if low == "boolean":
        return "BOOLEAN"
    if low in ("jsonb", "json"):
        return "TEXT"
    if low in ("uuid", "character varying", "character", "char", "varchar"):
        return "TEXT"
    if low.endswith("[]"):
        return "TEXT"
    if low in ("double precision", "real"):
        return "REAL"
    if low in ("bigint", "smallint", "integer", "int", "int4", "int8", "int2"):
        return "INTEGER"
    if low.startswith("numeric") or low.startswith("decimal"):
        return "NUMERIC"
    if low == "bytea":
        return "BLOB"
    if low == "interval":
        return "TEXT"
    return "TEXT"


def translate_pg_expr(e: str) -> str:
    """Normalize a Postgres expression (CHECK body, index expr/predicate) to SQLite."""
    # 1. strip all ::<type> casts (text, character varying, double precision,
    #    text[], etc.) that appear outside string literals.
    e = re.sub(r"::[\w ]+(\[\])?", "", e)
    # 2. char_length(...) -> length(...)
    e = re.sub(r"\bchar_length\s*\(", "length(", e)
    # 3. TRIM(BOTH FROM x) -> trim(x)
    e = re.sub(r"TRIM\s*\(\s*BOTH\s+FROM\s+([^)]+)\)", r"trim(\1)", e)
    # 4. AT TIME ZONE 'tz' -> (removed; SQLite stores UTC ISO-8601 strings)
    e = re.sub(r"\s+AT\s+TIME\s+ZONE\s+'[^']*'", "", e)
    # 5. col = ANY ( ( ARRAY [ ... ] ) ) -> col IN (...)  (handle nested parens
    #    that pg_dump emits, e.g. ANY ((ARRAY[...]::text[]))) ).
    e = re.sub(r"\(\s*([\w.]+)\s*\)\s*=\s*ANY\s*\(\s*\(\s*ARRAY\s*\[([^\]]*)\]\s*\)\s*\)",
               r"\1 IN (\2)", e)
    e = re.sub(r"([\w.]+)\s*=\s*ANY\s*\(\s*\(\s*ARRAY\s*\[([^\]]*)\]\s*\)\s*\)",
               r"\1 IN (\2)", e)
    e = re.sub(r"\(\s*([\w.]+)\s*\)\s*=\s*ANY\s*\(\s*ARRAY\s*\[([^\]]*)\]\s*\)",
               r"\1 IN (\2)", e)
    e = re.sub(r"([\w.]+)\s*=\s*ANY\s*\(\s*ARRAY\s*\[([^\]]*)\]\s*\)",
               r"\1 IN (\2)", e)
    # 6. boolean literal comparisons
    e = re.sub(r"=\s*true\b", "= 1", e)
    e = re.sub(r"=\s*false\b", "= 0", e)
    return e


# translate_check is now an alias for the unified translate_pg_expr so CHECK
# bodies, index expressions, and partial-index predicates all share one
# normalizer.
def translate_check(expr: str) -> str:
    return translate_pg_expr(expr.strip())


# ---------------------------------------------------------------------------
# 5. Column-line translation.
# ---------------------------------------------------------------------------
def translate_column_line(raw: str, serial_col):
    line = raw.rstrip()
    trailing_comma = line.endswith(",")
    body = line[:-1].strip() if trailing_comma else line.strip()
    if not body:
        return None

    up = body.upper()
    if up.startswith("CONSTRAINT ") or up.startswith("CHECK ") or up.startswith("UNIQUE "):
        if re.search(r"\bCHECK\s*\(", body, re.IGNORECASE):
            m = re.search(r"\bCHECK\s*\((.*)\)\s*$", body, re.IGNORECASE | re.DOTALL)
            if m:
                name_m = re.match(r"(CONSTRAINT\s+\w+\s+)", body, re.IGNORECASE)
                prefix = name_m.group(1) if name_m else ""
                return f"  {prefix}CHECK ({translate_check(m.group(1))})"
        return f"  {body}"

    parts = body.split(None, 1)
    if len(parts) == 1:
        return f"  {body}"
    colname, rest = parts
    is_serial = bool(serial_col and colname == serial_col)

    type_str = ""
    rem = rest
    for mw in ("timestamp without time zone", "timestamp with time zone",
               "double precision", "character varying", "bit varying"):
        if rem.lower().startswith(mw):
            type_str, rem = mw, rem[len(mw):].lstrip()
            break
    if not type_str:
        m = re.match(r"(\w+)(\[\])?", rem)
        if m:
            type_str, rem = m.group(0), rem[m.end():].lstrip()
    # consume a type length/precision specifier like (20) or (10,2) -- SQLite
    # ignores them and they must not leak as stray tokens.
    m = re.match(r"\(\s*\d+(?:\s*,\s*\d+)?\s*\)", rem)
    if m:
        rem = rem[m.end():].lstrip()

    sqlite_type = "INTEGER" if is_serial else translate_type(type_str)
    mods = rem.strip()
    if mods:
        mods = re.sub(
            r"DEFAULT\s+(.*?)(?=\s+NOT NULL\s*$|\s*$)",
            lambda mm: translate_default(mm.group(1).strip()),
            mods,
        )
    out = f"  {colname} {sqlite_type}".rstrip()
    if mods:
        out += f" {mods}"
    return out


def build_table(tname: str) -> str:
    raw = tables_raw[tname]
    m = re.search(r"CREATE TABLE public\.\w+\s*\((.*)\)\s*;", raw, re.DOTALL)
    if not m:
        return f"-- !! FAILED to parse table {tname}"
    inner = m.group(1)
    serial_col = serial_cols.get(tname)
    raw_lines = [ln for ln in inner.split("\n") if ln.strip()]

    out_lines = []
    autoinc_emitted = False
    for ln in raw_lines:
        translated = translate_column_line(ln, serial_col)
        if translated is None:
            continue
        if serial_col and re.match(rf"  {re.escape(serial_col)}\s+INTEGER", translated):
            translated = translated.rstrip()
            had_comma = translated.endswith(",")
            if had_comma:
                translated = translated[:-1]
            translated = re.sub(r"\s+NOT NULL\s*$", "", translated)
            if "PRIMARY KEY" not in translated:
                translated += " PRIMARY KEY AUTOINCREMENT"
            autoinc_emitted = True
            translated = translated + ("," if had_comma else "")
        out_lines.append(translated)

    if pk_cols.get(tname) and not autoinc_emitted:
        out_lines.append(f"  PRIMARY KEY ({', '.join(pk_cols[tname])})")
    for fk in fks.get(tname, []):
        out_lines.append(f"  {fk},")

    cleaned = []
    n = len(out_lines)
    for i, ln in enumerate(out_lines):
        ln = ln.rstrip()
        if ln.endswith(","):
            ln = ln[:-1]
        cleaned.append(ln + ("," if i < n - 1 else ""))
    return f"CREATE TABLE {tname} (\n" + "\n".join(cleaned) + "\n);"


# ---------------------------------------------------------------------------
# 6. Indexes.
# ---------------------------------------------------------------------------
def translate_indexes():
    out, dropped = [], []
    for name, body in bodies.get("INDEX", []):
        m = re.search(
            r"CREATE (UNIQUE )?INDEX (\w+) ON public\.(\w+) USING (\w+) \((.*?)\)(\s+WHERE \(.*?\))?\s*;",
            body,
            re.DOTALL,
        )
        if not m:
            m2 = re.search(r"CREATE (UNIQUE )?INDEX (\w+) ON public\.(\w+)\s+(.*?)\s*;", body, re.DOTALL)
            if not m2:
                print(f"!! could not parse index {name!r}", file=sys.stderr)
                continue
            dropped.append(name)
            continue
        uniq, idxname, tbl, method, cols, where = m.groups()
        if method.lower() == "gin":
            dropped.append(f"{idxname} (GIN)")
            continue
        cols = translate_pg_expr(cols).strip()
        where = translate_pg_expr(where) if where else ""
        out.append(f"CREATE {uniq or ''}INDEX {idxname} ON {tbl} ({cols}){where};")
    if dropped:
        print(f"# dropped indexes: {', '.join(dropped)}", file=sys.stderr)
    return out, len(out)


def translate_unique_indexes():
    out = []
    for tbl, colgroups in unique_constraints.items():
        for cols in colgroups:
            clean = [c.strip().strip('"') for c in cols]
            idxname = "uq_" + tbl + "_" + "_".join(clean)
            out.append(f"CREATE UNIQUE INDEX {idxname} ON {tbl} ({', '.join(cols)});")
    return out


# ---------------------------------------------------------------------------
# 7. Emit consolidated schema (preserving pg_dump table order -- FK-safe).
# ---------------------------------------------------------------------------
header = """\
-- Consolidated SQLite schema for the Zettelgarden backend.
--
-- Source: pg_dump --schema-only of the live dev Postgres DB, translated to
-- SQLite by schema/sqlite/translate.py (see migration design doc Phase 2).
--
-- This single file represents the *current final-state* schema. New installs
-- build the DB from this file (loaded via server/sqlsplit.go). The historical
-- Postgres migrations under schema/*.sql are retained only for the one-time
-- ETL (Phase 6b).
--
-- NOT here by design:
--   * CREATE FUNCTION / TRIGGER / COMMENT ON -- trigger logic is ported to Go
--     in Phase 5 (see design doc); nothing in Go reads the GIN indexes.
--   * The 3 GIN indexes (FTS to_tsvector on files.extracted_text, JSONB
--     containment on chat_messages.referenced_cards, array containment on
--     notifications.filter_tags) -- verified dead, dropped.
--   * Sequences -- replaced by INTEGER PRIMARY KEY AUTOINCREMENT.
--
-- Timestamp columns are declared DATETIME (not TEXT): modernc.org/sqlite needs
-- the declared type to return time.Time (migration design doc decision D5).
-- SQLite's loose affinity still stores the value as text either way.
--
-- Generated by translate.py; re-run it against a fresh dump to regenerate.
-- Do not hand-edit.

PRAGMA foreign_keys = ON;

"""

chunks = [header]
for tname in table_order:
    chunks.append(build_table(tname) + "\n\n")

chunks.append("\n-- Ordinary indexes (btree -> SQLite; partial-index predicates translated)\n")
idx_kept, n_idx = translate_indexes()
for line in idx_kept:
    chunks.append(line + "\n")

uq = translate_unique_indexes()
chunks.append("\n-- Unique constraints (from ALTER TABLE ... ADD CONSTRAINT UNIQUE)\n")
for line in uq:
    chunks.append(line + "\n")

with open(OUT, "w", encoding="utf-8") as f:
    f.write("".join(chunks))

print(f"wrote {OUT}")
print(f"tables={len(table_order)}  serial_pks={len(serial_cols)}  "
      f"composite_pks={sum(1 for t in table_order if pk_cols.get(t) and t not in serial_cols)}  "
      f"fks={sum(len(v) for v in fks.values())}  btree_indexes={n_idx}  "
      f"unique_indexes={len(uq)}")
