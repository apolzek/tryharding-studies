FROM ubuntu:24.04

ARG K0S_VERSION=v1.31.2+k0s.0
ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      ca-certificates curl iproute2 iptables kmod \
      conntrack ethtool socat jq tini tzdata \
 && apt-get purge -y --auto-remove \
 && rm -rf /var/lib/apt/lists/* /var/cache/apt/* /usr/share/doc/* /usr/share/man/*

COPY scripts/install-k0s.sh /usr/local/bin/install-k0s.sh
RUN chmod 0755 /usr/local/bin/install-k0s.sh \
 && K0S_VERSION="${K0S_VERSION}" /usr/local/bin/install-k0s.sh

COPY config/k0s.yaml          /etc/k0s/k0s.yaml
COPY config/audit-policy.yaml /etc/k0s/audit-policy.yaml
COPY scripts/entrypoint.sh    /usr/local/bin/entrypoint.sh
RUN printf 'nameserver 1.1.1.1\nnameserver 8.8.8.8\noptions edns0\n' > /etc/resolv.conf.k0s \
 && chmod 0755 /usr/local/bin/entrypoint.sh \
 && chmod 0640 /etc/k0s/k0s.yaml /etc/k0s/audit-policy.yaml /etc/resolv.conf.k0s

ENV PATH="/usr/local/bin:${PATH}"
EXPOSE 6443 8132 9443
ENTRYPOINT ["/usr/bin/tini","--","/usr/local/bin/entrypoint.sh"]
