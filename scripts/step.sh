#!/bin/sh
# step.sh LABEL DONE_LABEL -- COMMAND [ARGS...]
# Runs one deploy step. Quiet by default: the command's output goes to a log
# under .deploy-logs/ and the terminal shows "RUN label…" then "OK done label in 4s".
# On failure the end of the log is shown, with where the whole log is.
# VERBOSE=1 shows the command's own output instead (make deploy VERBOSE=1).
# STEP_NOSPIN=1 skips the spinner (for steps that run in parallel).
label=$1; done_label=$2; shift 3
. "$(dirname "$0")/pretty.sh"
dir=${STEP_LOG_DIR:-.deploy-logs}; mkdir -p "$dir"
log="$dir/$(printf '%s' "$label" | tr -c 'A-Za-z0-9' '-' | tr -s '-' | sed 's/^-//;s/-$//' | tr 'A-Z' 'a-z').log"
start=$(date +%s)

if [ -n "$VERBOSE" ]; then
  say_step "$label…"
  "$@"; rc=$?
  secs=$(( $(date +%s) - start ))
  if [ "$rc" -eq 0 ]; then say_ok "$done_label in ${secs}s"; else say_fail "$label failed (after ${secs}s)"; fi
  exit "$rc"
fi

"$@" >"$log" 2>&1 &
pid=$!
if [ -t 1 ] && [ -z "$NO_COLOR" ] && [ -z "$STEP_NOSPIN" ]; then
  i=0
  while kill -0 "$pid" 2>/dev/null; do
    case $((i % 4)) in 0) f='⠋';; 1) f='⠙';; 2) f='⠹';; *) f='⠸';; esac
    printf '\r%s %s%s%s %s… %s%ss%s ' "$B_STEP" "$C_CYAN" "$f" "$C_RESET" "$label" "$C_DIM" "$(( $(date +%s) - start ))" "$C_RESET"
    i=$((i + 1)); sleep 0.15
  done
  printf '\r\033[K'
else
  say_step "$label…"
fi
wait "$pid"; rc=$?
secs=$(( $(date +%s) - start ))
if [ "$rc" -eq 0 ]; then
  say_ok "$done_label in ${secs}s"
else
  say_fail "$label failed (after ${secs}s)"
  tail -n 30 "$log"
  say_dim "Full log: $log   (or rerun with VERBOSE=1)"
fi
exit "$rc"
