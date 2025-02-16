from bcc import BPF
from pyroute2 import IPRoute
import pyroute2
bpf_program = """
#include <uapi/linux/bpf.h>
#include <linux/ip.h>
#include <linux/in.h>
#include <linux/if_ether.h>
#define SRC_IP  0x0AF400A4 // 10.244.0.6 (hex representation)
#define PODIP 0x0AF40109  // 10.244.1.8 (hex representation)
#define SVCIP 0x0A604D4C  // 10.96.77.76 (hex representation)
#define NEW_DST_IP 0x0AF401EF  // 10.244.1.8 (hex representation)
int redirect_service(struct __sk_buff *skb) {
    struct ethhdr eth;
    struct iphdr ip;
    int ip_offset = 14;
    if (bpf_skb_load_bytes(skb, 0, &eth, sizeof(eth)) < 0)
        return 1;
    if (eth.h_proto != bpf_htons(ETH_P_IP))
        return 1;
    if (bpf_skb_load_bytes(skb, ip_offset, &ip, sizeof(ip)) < 0)
        return 1;
    u32 dst_ip = bpf_ntohl(ip.daddr);
    u32 src_ip = bpf_ntohl(ip.saddr);
    
    if (src_ip == NEW_DST_IP) {
        bpf_trace_printk("Captured return packet to destination IP: %d.%d.%d\\n",
            (dst_ip >> 24) & 0xFF,
            (dst_ip >> 16) & 0xFF,
            (dst_ip >> 8) & 0xFF);
    }
    
    if (src_ip == SRC_IP) {
        if (dst_ip == SVCIP) {
        bpf_trace_printk("From tctry vethaf37f675\\n");
        bpf_trace_printk("Captured packet to destination IP: %d.%d.%d\\n",
                    (dst_ip >> 24) & 0xFF,
                    (dst_ip >> 16) & 0xFF,
                    (dst_ip >> 8) & 0xFF);
        bpf_trace_printk("Captured packet to destination IP: %d.\\n", dst_ip & 0xFF);
        bpf_trace_printk("Captured packet from source IP: %d.%d.%d\\n",
                        (src_ip >> 24) & 0xFF,
                        (src_ip >> 16) & 0xFF,
                        (src_ip >> 8) & 0xFF);
        bpf_trace_printk("Captured packet from source IP: %d.\\n", src_ip & 0xFF);
        u32 new_dst_ip = bpf_htonl(NEW_DST_IP);

        // Replace destination IP in the packet
        bpf_skb_store_bytes(skb, ip_offset + offsetof(struct iphdr, daddr), &new_dst_ip, sizeof(new_dst_ip), 0);

        // Fix the IP checksum
        bpf_l3_csum_replace(skb, ip_offset + offsetof(struct iphdr, check), ip.daddr, new_dst_ip, sizeof(new_dst_ip));
        bpf_trace_printk("New destination IP: %d.%d.%d\\n",
                        (bpf_ntohl(ip.daddr) >> 24) & 0xFF,
                        (bpf_ntohl(ip.daddr) >> 16) & 0xFF,
                        (bpf_ntohl(ip.daddr) >> 8) & 0xFF);
        bpf_trace_printk("New destination IP: %d.\\n", bpf_ntohl(ip.daddr) & 0xFF);
        }
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
    fn = b.load_func("redirect_service", BPF.SCHED_CLS)
    # ipr.tc("add", "clsact", idx)
    ipr.tc("add-filter", "bpf", idx, ":1", fd=fn.fd, name=fn.name, parent="ffff:fff2", classid=1)
    print(f"BPF attached to {interface} - SCHED_CLS: OK")
    print("Waiting for packets... Press Ctrl+C to stop.")
    b.trace_print()
finally:
    print("Exiting... No interface deletion performed.")