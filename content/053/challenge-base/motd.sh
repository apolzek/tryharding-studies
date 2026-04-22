#!/usr/bin/env bash
TITLE="${CHALLENGE_TITLE:-SRE Challenge}"
OBJ="${CHALLENGE_OBJECTIVE:-fix whatever is broken}"
cat <<BANNER
────────────────────────────────────────────────────────────
  $TITLE
────────────────────────────────────────────────────────────
  Objective: $OBJ
  You have passwordless sudo. Fix it, then click "Verify".
────────────────────────────────────────────────────────────
BANNER
