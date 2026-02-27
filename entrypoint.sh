#!/bin/bash
set -e

SOURCE_IMAGE="/usr/local/share/mz5-source"
SOURCE_DATA="/data/source"
CLAUDE_DATA="/data/.claude"
SSH_DATA="/data/.ssh"

# --- Setup symlinks for ALL users that might run Claude Code ---
# The server runs as 'app' but Claude Code runs as root.
setup_user_symlinks() {
    local user_home="$1"
    local user_name="$2"

    echo "entrypoint: setting up symlinks for $user_name (home=$user_home)"

    # Claude Code data
    if [ -d "$user_home/.claude" ] && [ ! -L "$user_home/.claude" ]; then
        echo "entrypoint: merging $user_name ephemeral .claude into persistent volume"
        cp -an "$user_home/.claude/." "$CLAUDE_DATA/" 2>/dev/null || true
        rm -rf "$user_home/.claude"
    fi
    ln -sfn "$CLAUDE_DATA" "$user_home/.claude"

    # SSH keys
    if [ -d "$SSH_DATA" ]; then
        if [ -d "$user_home/.ssh" ] && [ ! -L "$user_home/.ssh" ]; then
            rm -rf "$user_home/.ssh"
        fi
        ln -sfn "$SSH_DATA" "$user_home/.ssh"
    fi
}

# --- Source code persistence ---
if [ ! -d "$SOURCE_DATA/.git" ]; then
    echo "entrypoint: first boot — copying source to persistent volume"
    cp -a "$SOURCE_IMAGE" "$SOURCE_DATA"
fi
git config --global --add safe.directory "$SOURCE_DATA"

# --- Ensure persistent dirs exist ---
mkdir -p "$CLAUDE_DATA"
mkdir -p "$SSH_DATA"

# --- Symlinks for root (Claude Code runs as root) ---
setup_user_symlinks "/root" "root"

# --- Symlinks for app user (server runs as app) ---
setup_user_symlinks "/home/app" "app"

# --- Project-level .claude persistence ---
PROJECT_CLAUDE="$SOURCE_DATA/.claude"
CLAUDE_PROJECT_DATA="/data/.claude-project"
if [ ! -d "$CLAUDE_PROJECT_DATA" ]; then
    if [ -d "$PROJECT_CLAUDE" ] && [ ! -L "$PROJECT_CLAUDE" ]; then
        mv "$PROJECT_CLAUDE" "$CLAUDE_PROJECT_DATA"
    else
        mkdir -p "$CLAUDE_PROJECT_DATA"
    fi
fi
if [ -d "$PROJECT_CLAUDE" ] && [ ! -L "$PROJECT_CLAUDE" ]; then
    rm -rf "$PROJECT_CLAUDE"
fi
ln -sfn "$CLAUDE_PROJECT_DATA" "$PROJECT_CLAUDE"

# --- Session helper script ---
# Creates /data/bin/m — a single command to resume or start a tmux+claude session.
# Survives SSH disconnects (tmux) and machine restarts (claude --continue).
mkdir -p /data/bin
cat > /data/bin/m << 'SCRIPT'
#!/bin/bash
SESSION="mind"
WORKDIR="/data/source"

# If already inside the tmux session, just run claude
if [ "$TMUX" ] && [ "$(tmux display-message -p '#S')" = "$SESSION" ]; then
    cd "$WORKDIR"
    exec claude --continue "$@"
fi

# If tmux session exists, attach to it
if tmux has-session -t "$SESSION" 2>/dev/null; then
    echo "Attaching to existing session..."
    exec tmux attach -t "$SESSION"
fi

# No session — start a new tmux session with claude --continue
echo "Starting new session (resuming last conversation)..."
exec tmux new-session -s "$SESSION" -c "$WORKDIR" "claude --continue $*; bash"
SCRIPT
chmod +x /data/bin/m

# Add /data/bin to PATH for all users
if ! grep -q '/data/bin' /etc/profile 2>/dev/null; then
    echo 'export PATH="/data/bin:$PATH"' >> /etc/profile
fi

echo "entrypoint: source=$SOURCE_DATA, claude=$CLAUDE_DATA, ssh=$SSH_DATA"

# --- Ensure go is in PATH for mind's preflight check ---
ln -sf /usr/local/go/bin/go /usr/local/bin/go 2>/dev/null || true

# --- Build from /data/source if newer than installed binaries ---
# The persistent volume has the latest code; Docker image binaries may be stale.
if [ -f "$SOURCE_DATA/go.mod" ]; then
    SOURCE_MOD=$(stat -c %Y "$SOURCE_DATA/go.mod" 2>/dev/null || echo 0)
    SERVER_MOD=$(stat -c %Y /usr/local/bin/server 2>/dev/null || echo 0)
    if [ "$SOURCE_MOD" -gt "$SERVER_MOD" ]; then
        echo "entrypoint: source is newer than binaries, rebuilding..."
        cd "$SOURCE_DATA"
        if /usr/local/go/bin/go build -o /usr/local/bin/server ./cmd/server && \
           /usr/local/go/bin/go build -o /usr/local/bin/mind ./cmd/mind && \
           /usr/local/go/bin/go build -o /usr/local/bin/eg ./cmd/eg; then
            echo "entrypoint: rebuild complete"
        else
            echo "entrypoint: rebuild failed, using Docker image binaries"
        fi
    fi
fi

# --- Start server (mind runs in-process) ---
# Server and mind share the same process and event bus.
# No separate mind process or watchdog needed.
echo "entrypoint: starting server (mind runs in-process)"
exec "$@"
