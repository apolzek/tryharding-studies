#!/usr/bin/env bash
# Entrypoint for every SRE challenge container.
# 1. Runs /challenge/setup.sh as root (if present) to break the system on purpose.
# 2. Launches ttyd so the candidate gets a bash in the browser.
set -u

CHALLENGE_DIR=/challenge
LOG=/var/log/challenge-setup.log

mkdir -p "$CHALLENGE_DIR"

if [ -x "$CHALLENGE_DIR/setup.sh" ]; then
  echo "[entrypoint] running setup.sh" | tee -a "$LOG"
  bash "$CHALLENGE_DIR/setup.sh" >>"$LOG" 2>&1 || {
    echo "[entrypoint] setup.sh failed (exit=$?) — continuing anyway" | tee -a "$LOG"
  }
fi

chown -R sre:sre /home/sre || true

# MOTD shown every time the shell starts.
cat >/etc/profile.d/00-challenge.sh <<'EOSH'
/usr/local/bin/motd.sh
EOSH
chmod +x /etc/profile.d/00-challenge.sh

# ttyd options:
#  -W  writable  (stdin accepted)
#  -p  port
#  -t  titleFixed
#  --writable already default; we want no auth because the port is protected
#  by the platform itself (session URL is ephemeral).
exec /usr/local/bin/ttyd \
  -p 7681 \
  -W \
  -t titleFixed="SRE Challenge" \
  -t fontSize=14 \
  -t 'theme={"background":"#0d1117","foreground":"#c9d1d9"}' \
  sudo -u sre -i bash
