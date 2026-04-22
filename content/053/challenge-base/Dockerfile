FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
ENV TERM=xterm-256color

RUN apt-get update && apt-get install -y --no-install-recommends \
      bash coreutils util-linux procps psmisc iproute2 iputils-ping \
      net-tools dnsutils tcpdump curl wget jq vim nano less tree \
      htop sysstat strace lsof file git \
      nginx openssh-client openssl ca-certificates \
      postgresql-client redis-tools \
      cron rsyslog logrotate supervisor \
      python3 python3-pip sudo \
      ncat netcat-openbsd \
      build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install ttyd (static binary from upstream releases)
ARG TTYD_VERSION=1.7.7
RUN ARCH=$(dpkg --print-architecture) && \
    case "$ARCH" in \
      amd64) TTYD_ARCH=x86_64 ;; \
      arm64) TTYD_ARCH=aarch64 ;; \
      *) echo "Unsupported arch $ARCH" && exit 1 ;; \
    esac && \
    curl -fsSL -o /usr/local/bin/ttyd \
      "https://github.com/tsl0922/ttyd/releases/download/${TTYD_VERSION}/ttyd.${TTYD_ARCH}" && \
    chmod +x /usr/local/bin/ttyd

# Non-root user the terminal runs as (with passwordless sudo, since challenges
# often require root to fix OS-level problems).
RUN useradd -m -s /bin/bash sre && \
    echo "sre ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/sre && \
    chmod 0440 /etc/sudoers.d/sre

WORKDIR /home/sre
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
COPY motd.sh /usr/local/bin/motd.sh
RUN chmod +x /usr/local/bin/entrypoint.sh /usr/local/bin/motd.sh

# setup.sh and verify.sh are copied in by the backend at container creation.
EXPOSE 7681
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
