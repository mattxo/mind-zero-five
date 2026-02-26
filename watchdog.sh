#!/bin/sh
# watchdog.sh — External process monitor for the mind.
#
# Does NOT depend on Go, Claude, Postgres, or the mind itself.
# Uses only: pgrep, kill, stat, git, go build, cp, date, sleep.
#
# Responsibilities:
# 1. Restart mind if the process dies
# 2. Kill and restart mind if it's stuck (no heartbeat for 2 minutes)
# 3. Revert last commit and rebuild if mind keeps crashing (3 crashes in 5 minutes)
# 4. Backup the mind binary before any restart

MIND_BIN="/usr/local/bin/mind"
MIND_BAK="/usr/local/bin/mind.bak"
HEARTBEAT="/tmp/mind-heartbeat"
CRASH_LOG="/tmp/mind-crashes"
WATCHDOG_LOG="/tmp/watchdog.log"
REPO="/data/source"
CHECK_INTERVAL=30
HEARTBEAT_MAX_AGE=120  # seconds before considering mind stuck
CRASH_WINDOW=300       # 5 minutes
MAX_CRASHES=3

log() {
    echo "$(date -u '+%Y-%m-%dT%H:%M:%SZ') watchdog: $*" | tee -a "$WATCHDOG_LOG"
}

record_crash() {
    echo "$(date +%s)" >> "$CRASH_LOG"
}

recent_crash_count() {
    if [ ! -f "$CRASH_LOG" ]; then
        echo 0
        return
    fi
    cutoff=$(( $(date +%s) - CRASH_WINDOW ))
    count=0
    while read -r ts; do
        if [ "$ts" -gt "$cutoff" ] 2>/dev/null; then
            count=$(( count + 1 ))
        fi
    done < "$CRASH_LOG"
    echo "$count"
}

backup_binary() {
    if [ -f "$MIND_BIN" ]; then
        cp "$MIND_BIN" "$MIND_BAK"
    fi
}

restore_binary() {
    if [ -f "$MIND_BAK" ]; then
        log "restoring previous mind binary from backup"
        cp "$MIND_BAK" "$MIND_BIN"
        return 0
    fi
    return 1
}

revert_and_rebuild() {
    log "CRASH LOOP DETECTED — reverting last commit and rebuilding"

    cd "$REPO" || return 1

    # Try to revert the last commit
    if git revert --no-commit HEAD 2>/dev/null; then
        /usr/local/go/bin/go build -o "$MIND_BIN" ./cmd/mind 2>>"$WATCHDOG_LOG"
        if [ $? -eq 0 ]; then
            git checkout -- . 2>/dev/null  # clean up the revert, keep the old binary
            log "rebuilt mind with reverted code"
            return 0
        fi
        git checkout -- . 2>/dev/null
    fi

    # Revert failed or rebuild failed — try restoring backup binary
    if restore_binary; then
        return 0
    fi

    log "CRITICAL: cannot recover mind — revert failed, no backup"
    return 1
}

start_mind() {
    backup_binary
    "$MIND_BIN" &
    log "started mind (pid=$!)"
}

# --- Main loop ---
log "started (check_interval=${CHECK_INTERVAL}s, heartbeat_max=${HEARTBEAT_MAX_AGE}s)"

# Clear old crash log on fresh start
: > "$CRASH_LOG"

while true; do
    sleep "$CHECK_INTERVAL"

    mind_pid=$(pgrep -x mind)

    if [ -z "$mind_pid" ]; then
        # Mind is dead
        log "mind process not found"
        record_crash

        crashes=$(recent_crash_count)
        if [ "$crashes" -ge "$MAX_CRASHES" ]; then
            revert_and_rebuild
            : > "$CRASH_LOG"  # reset crash counter after recovery attempt
        fi

        start_mind
        continue
    fi

    # Mind is alive — check heartbeat
    if [ -f "$HEARTBEAT" ]; then
        heartbeat_age=$(( $(date +%s) - $(stat -c %Y "$HEARTBEAT" 2>/dev/null || echo 0) ))

        if [ "$heartbeat_age" -gt "$HEARTBEAT_MAX_AGE" ]; then
            log "mind stuck — no heartbeat for ${heartbeat_age}s (pid=$mind_pid), killing"
            kill "$mind_pid" 2>/dev/null
            sleep 2
            kill -9 "$mind_pid" 2>/dev/null
            record_crash

            crashes=$(recent_crash_count)
            if [ "$crashes" -ge "$MAX_CRASHES" ]; then
                revert_and_rebuild
                : > "$CRASH_LOG"
            fi

            start_mind
        fi
    fi
done
