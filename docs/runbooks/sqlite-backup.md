# SQLite Backup & Restore

Zettelgarden stores all data in a single SQLite database file (default
`./data/zettelgarden.db`, WAL mode). Because the database is a live file under
Write-Ahead Logging, **do not back it up with a plain `cp`** — `cp` can capture
the file mid-write and produce a torn/inconsistent copy, and it will miss or
partially copy the `-wal` / `-shm` sidecars.

## Recommended: `VACUUM INTO` (atomic online snapshot)

`VACUUM INTO` writes a consistent, defragmented single-file copy of the database
while the app keeps running. It is the right tool for scheduled backups.

```sh
sqlite3 /usr/src/app/data/zettelgarden.db \
  "VACUUM INTO '/backups/zettelgarden-$(date +%Y%m%d-%H%M%S).db'"
```

The output file is a standalone database (no `-wal` / `-shm` needed) and is safe
to copy off-host, compress, or restore directly.

### Daily cron example

```sh
# /etc/cron.d/zettelgarden-backup — daily 04:00, keep 14 days
0 4 * * * root sqlite3 /usr/src/app/data/zettelgarden.db \
  "VACUUM INTO '/backups/zettelgarden-$(date +\%Y\%m\%d).db'" \
  && find /backups -name 'zettelgarden-*.db' -mtime +14 -delete
```

## Alternative: the SQLite `.backup` command

`.backup` uses the online backup API (also consistent; app stays up) and is a
fine substitute:

```sh
sqlite3 /usr/src/app/data/zettelgarden.db ".backup '/backups/zettelgarden.db'"
```

## If you must use `cp` (stop-the-world)

Only safe with the writer quiesced and the WAL checkpointed first:

```sh
# 1. checkpoint the WAL into the main db, truncating the -wal file
sqlite3 /usr/src/app/data/zettelgarden.db "PRAGMA wal_checkpoint(TRUNCATE);"
# 2. (ideally) stop the app so nothing writes during the copy
# 3. copy the main file only — the -wal/-shm are no longer needed post-checkpoint
cp /usr/src/app/data/zettelgarden.db /backups/zettelgarden-$(date +%Y%m%d).db
```

Prefer `VACUUM INTO` for online backups; reserve `cp` for maintenance windows.

## Restore

1. Stop the backend (`docker compose stop go_backend`, or equivalent).
2. Move the live file aside: `mv data/zettelgarden.db data/zettelgarden.db.broken`.
3. Copy the snapshot into place:
   `cp /backups/zettelgarden-YYYYMMDD.db data/zettelgarden.db`.
4. Restart the backend. (The app recreates the `-wal` / `-shm` on first write.)

A `VACUUM INTO` snapshot restores cleanly because it is a single, complete
database file.

## Notes

- WAL mode means the on-disk `.db` may lag the `-wal` file between checkpoints —
  another reason to prefer `VACUUM INTO` / `.backup` over `cp`.
- Post-cutover baseline: run one `PRAGMA wal_checkpoint(TRUNCATE)` then a
  `VACUUM INTO` to establish the ongoing snapshot cadence.
- The paths above target the in-container location (`/usr/src/app/data/...` when
  running under Docker — see the `./data` volume mount in `docker-zettel-run.yml`
  / `docker-compose.yml`). Adjust for non-Docker installs.
