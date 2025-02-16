from bcc import BPF
from pyroute2 import IPRoute
import pyroute2
bpf_program = """
#include <uapi/linux/bpf.h>
#include <linux/ip.h>
#include <linux/in.h>
#include <linux/if_ether.h>

#define SVC_IP  0x0A604D4C // 10.96.77.76
#define BACKEND_IP 0x0AF401ED // 10.244.1.216

int nat_ingress(struct __sk_buff *skb) {
    struct ethhdr eth;
    struct iphdr ip;
    int ip_offset = sizeof(struct ethhdr);

    if (bpf_skb_load_bytes(skb, 0, &eth, sizeof(eth)) < 0)
        return 1;

    if (eth.h_proto != bpf_htons(ETH_P_IP))
        return 1;

    if (bpf_skb_load_bytes(skb, ip_offset, &ip, sizeof(ip)) < 0)
        return 1;
        
    u32 dst_ip = bpf_ntohl(ip.daddr);

    if (dst_ip == SVC_IP) {
        __be32 new_dst = bpf_htonl(BACKEND_IP);
        
        bpf_trace_printk("INGRESS: src=%x, dst=%x -> DNAT to %x\\n", ip.saddr, ip.daddr, new_dst);

        // DNAT: Change destination to BACKEND_IP
        bpf_skb_store_bytes(skb, ip_offset + offsetof(struct iphdr, daddr), &new_dst, sizeof(new_dst), 0);
        bpf_l3_csum_replace(skb, ip_offset + offsetof(struct iphdr, check), ip.daddr, new_dst, sizeof(new_dst));
    }

    return 1;
}

int nat_egress(struct __sk_buff *skb) {
    struct ethhdr eth;
    struct iphdr ip;
    int ip_offset = sizeof(struct ethhdr);

    if (bpf_skb_load_bytes(skb, 0, &eth, sizeof(eth)) < 0)
        return 1;

    if (eth.h_proto != bpf_htons(ETH_P_IP))
        return 1;

    if (bpf_skb_load_bytes(skb, ip_offset, &ip, sizeof(ip)) < 0)
        return 1;

    u32 src_ip = bpf_ntohl(ip.daddr);
    
    if (src_ip == BACKEND_IP) {
        __be32 svc_ip = bpf_htonl(SVC_IP);

        // SNAT: Change source back to SVC_IP
        bpf_skb_store_bytes(skb, ip_offset + offsetof(struct iphdr, saddr), &svc_ip, sizeof(svc_ip), 0);
        bpf_l3_csum_replace(skb, ip_offset + offsetof(struct iphdr, check), ip.saddr, svc_ip, sizeof(svc_ip));
    }

    return 1;
}
"""
ipr = IPRoute()
interface = "vethe0068b41"
# Ensure the interface exists
try:
    idx = ipr.link_lookup(ifname=interface)[0]
except IndexError:
    print(f"Error: Interface {interface} not found. Is it created?")
    exit(1)
# # Ensure cleanup of the existing ingress qdisc
# try:
#     ipr.tc("del", "ingress", idx)  # Remove existing ingress qdisc
# except pyroute2.netlink.exceptions.NetlinkError:
#     pass  # In case it doesn't exist
# Attach to veth0 using TC
try:
    b = BPF(text=bpf_program)
    ingress_fn = b.load_func("nat_ingress", BPF.SCHED_CLS)
    egress_fn = b.load_func("nat_egress", BPF.SCHED_CLS)

    # ipr.tc("add", "clsact", idx)

    ipr.tc("add-filter", "bpf", idx, ":1", fd=ingress_fn.fd, name=ingress_fn.name, parent="ffff:fff2", classid=1)
    ipr.tc("add-filter", "bpf", idx, ":2", fd=egress_fn.fd, name=egress_fn.name, parent="ffff:fff3", classid=1)

    b.trace_print()
finally:
    print("Exiting... No interface deletion performed.")