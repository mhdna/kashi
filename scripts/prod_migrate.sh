#!/bin/sh
# Runs on the server (ssh host 'sh -s' < this): applies pending migrations.
# Usage: prod_migrate.sh MIGRATIONS_DIR DB_URL
# A migration that needs a table lock waits behind any long query on that
# table, and every new query then queues behind it, which freezes the site
# while the deploy sits silent. So it gives up after LOCK_TIMEOUT (default
# 20s), says which query is in the way and how to clear it.
dir=$1; db=$2
export PGOPTIONS="-c lock_timeout=${LOCK_TIMEOUT:-20s}"
out=$(migrate -path "$dir" -database "$db" up 2>&1); rc=$?
printf '%s\n' "$out"
[ "$rc" -eq 0 ] && exit 0

echo
case "$out" in
  *"Dirty database"*)
    echo "WHY: an earlier migration stopped half way, so the database is marked dirty."
    echo "     Check what it left behind, then: migrate -path $dir -database <db> force <last good version>" ;;
  *"lock timeout"*|*"lock_timeout"*)
    echo "WHY: the migration waited ${LOCK_TIMEOUT:-20s} for a table lock and gave up."
    echo "     Nothing was applied, but the database is now marked dirty at that version."
    echo "     Queries holding things up (oldest first):"
    psql "$db" -At -F '  ' -c "select pid, now()-xact_start, left(regexp_replace(query,'\s+',' ','g'),110) from pg_stat_activity where datname=current_database() and pid<>pg_backend_pid() and xact_start is not null and cardinality(pg_blocking_pids(pid))=0 order by xact_start limit 5" 2>&1
    echo "     Cancel one with: select pg_cancel_backend(<pid>);  then: migrate ... force <previous version>, and deploy again." ;;
  *) echo "WHY: see the migrate output above." ;;
esac
exit "$rc"
