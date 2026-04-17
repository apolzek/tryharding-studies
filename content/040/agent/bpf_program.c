// eBPF program for user activity monitoring.
//
// Attach points:
//   - kprobe:input_handle_event  -> keyboard + mouse events (button + motion)
//   - kprobe:tcp_sendmsg         -> outbound TCP bytes (per UID)
//   - kprobe:tcp_cleanup_rbuf    -> inbound TCP bytes (per UID)
//
// Design: all counters live in kernel BPF maps; userspace polls them every
// few seconds. No perf events / ring buffers are used on the hot path, so
// overhead is O(1) per event. This program intentionally avoids including
// heavy kernel headers (net/sock.h, linux/input.h) because BCC bundles
// pre-built headers that can mismatch the running kernel; we only need the
// argument values (passed via register-mapped kprobe arguments) and treat
// struct sock*/input_dev* as opaque pointers.

#include <uapi/linux/ptrace.h>

// ---- Global counters (single-slot BPF_ARRAY) ----
// Indexes are stable; see AGG_* constants in the Python side.
BPF_ARRAY(agg, u64, 8);

#define AGG_KEYS      0  // key press events (EV_KEY with value==1)
#define AGG_CLICKS    1  // mouse button press (EV_KEY on BTN_* with value==1)
#define AGG_MOVES     2  // mouse relative motion events (EV_REL)
#define AGG_SCROLL    3  // scroll wheel events
#define AGG_KEY_REL   4  // key release events (useful to derive hold time)
#define AGG_RX_BYTES  5  // TCP bytes received (aggregate, all users)
#define AGG_TX_BYTES  6  // TCP bytes sent (aggregate, all users)
#define AGG_NET_CALLS 7  // total socket calls (send+recv)

// ---- Per-UID network counters ----
struct net_stats_t {
    u64 rx_bytes;
    u64 tx_bytes;
    u64 rx_calls;
    u64 tx_calls;
};
BPF_HASH(net_by_uid, u32, struct net_stats_t, 4096);

// ---- Input event hook ----
// void input_handle_event(struct input_dev *dev, unsigned int type,
//                         unsigned int code, int value)
//
// EV_KEY = 1 ; EV_REL = 2
// EV_KEY value: 0=release 1=press 2=autorepeat
// EV_REL code : REL_WHEEL=8, REL_HWHEEL=6
// BTN_MOUSE..BTN_TASK : 0x110..0x117
int kprobe__input_handle_event(struct pt_regs *ctx, void *dev,
                               unsigned int type, unsigned int code,
                               int value)
{
    u32 k;
    u64 one = 1;

    if (type == 1 /* EV_KEY */) {
        if (code >= 0x110 && code <= 0x117) {
            if (value == 1) {
                k = AGG_CLICKS;
                agg.atomic_increment(k, one);
            }
        } else {
            if (value == 1) {
                k = AGG_KEYS;
                agg.atomic_increment(k, one);
            } else if (value == 0) {
                k = AGG_KEY_REL;
                agg.atomic_increment(k, one);
            }
        }
    } else if (type == 2 /* EV_REL */) {
        if (code == 8 || code == 6) {
            k = AGG_SCROLL;
            agg.atomic_increment(k, one);
        } else {
            k = AGG_MOVES;
            agg.atomic_increment(k, one);
        }
    }
    return 0;
}

// ---- TCP send ----
// int tcp_sendmsg(struct sock *sk, struct msghdr *msg, size_t size)
int kprobe__tcp_sendmsg(struct pt_regs *ctx, void *sk, void *msg, u64 size)
{
    u32 uid = (u32)bpf_get_current_uid_gid();
    u64 sz = size;

    u32 k = AGG_TX_BYTES;
    agg.atomic_increment(k, sz);
    k = AGG_NET_CALLS;
    u64 one = 1;
    agg.atomic_increment(k, one);

    struct net_stats_t zero = {};
    struct net_stats_t *st = net_by_uid.lookup_or_try_init(&uid, &zero);
    if (st) {
        __sync_fetch_and_add(&st->tx_bytes, sz);
        __sync_fetch_and_add(&st->tx_calls, 1);
    }
    return 0;
}

// ---- TCP recv accounting ----
// void tcp_cleanup_rbuf(struct sock *sk, int copied)
int kprobe__tcp_cleanup_rbuf(struct pt_regs *ctx, void *sk, int copied)
{
    if (copied <= 0)
        return 0;

    u32 uid = (u32)bpf_get_current_uid_gid();
    u64 sz = (u64)copied;

    u32 k = AGG_RX_BYTES;
    agg.atomic_increment(k, sz);
    k = AGG_NET_CALLS;
    u64 one = 1;
    agg.atomic_increment(k, one);

    struct net_stats_t zero = {};
    struct net_stats_t *st = net_by_uid.lookup_or_try_init(&uid, &zero);
    if (st) {
        __sync_fetch_and_add(&st->rx_bytes, sz);
        __sync_fetch_and_add(&st->rx_calls, 1);
    }
    return 0;
}
