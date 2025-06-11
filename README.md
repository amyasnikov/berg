# BERG - **B**GP **E**VPN **R**edistribution (written in **G**o)

## The problem

Have you ever wondered how to connect Virtual Machines to a EVPN/VXLAN DC fabric via BGP? If yes, you know that there is almost no clean way to do this.
Here are the pitfalls of the common approaches:

1. Do BGP with a pair of leaves the hypervisor is connected with, use leaf VRF loopback addresses to establish BGP. Although this works nice for BM servers, this approach has one big problem for VMs: it does not support live migration to another hypervisor (usually connected to another pair of leaves).
2. Do BGP with a pair of border leaves/routers (always the same for a particluar DC fabric). This approach causes traffic tromboning through border leaves.


## The solution

BERG serves as a BGP route server (which means it is out of the traffic forwarding path).

BERG accepts BGP IPv4 announcements from Virtual Machines, *redistributes**  them into EVPN Type-5 routes and sets **Overlay Gateway Address** NLRI attribute equal to BGP IPv4 NextHop. Effectively this allows to forward traffic directly towards the next-hop which is usually behind the anycast gateway. Reverse-side redistribution (EVPN -> IPv4) is supported as well.

**Since there is no common term for BGP route transition from one family to another, the word **redistribution** is used.*


## Configuration

BERG is built atop of [GoBGP](https://github.com/osrg/gobgp) and preserves its configuration structure.

Example configuration:

```toml
[global.config]
  as = 100
  router-id = "10.5.0.100"

[[vrfs]]
    [vrfs.config]
        name = "vrf_10"
        id = 10           # This ID means VNI of the VRF which will be inserted into EVPN routes
        rd = "100:10"
        both-rt-list = ["100:10"]

[[neighbors]]
  [neighbors.config]
    neighbor-address = "10.5.0.1"
    peer-as = 100
  [neighbors.route-reflector.config]
    route-reflector-client = true
    route-reflector-cluster-id = "10.5.0.100"
  [[neighbors.afi-safis]]
    [neighbors.afi-safis.config]
      afi-safi-name = "l2vpn-evpn"


[[neighbors]]
  [neighbors.config]
    neighbor-address = "192.168.0.10"
    peer-as = 10
    vrf = "vrf_10"
  [[neighbors.afi-safis]]
    [neighbors.afi-safis.config]
      afi-safi-name = "ipv4-unicast"

```

## FAQ


**I don't get how this works**

BERG is the same GoBGP, but with one additional feature: it "redistributes" the routes between IPv4 family inside VRFs and BGP EVPN. When redistributing the route from VRF IPv4 to EVPN Type-5 Overaly Gateway IP is filled out to achieve optimal forwarding. You may search for what `set evpn gateway-ip use-nexthop` Cisco NX OS command does. Berg implements the same functionality, so DC fabrics built with non-Cisco equipment are able to set up BGP IPv4 sessions with VMs and achieve optimal traffic formwarding path.

More info on how Overlay GW IP in Type-5 routes works can be found in [RFC9136](https://datatracker.ietf.org/doc/rfc9136/)


**How to run BERG?**

`./berg -f config.toml`


**How to update config without breaking existing BGP sessions?**

Config file live reloading is supported. Just update the file and save it, after that BERG re-applies the configuration from the file.


**How to get operational state info?**

The easiest way is to use the default `gobgp` CLI tool which is able to communicate with BERG via gRPC. BERG listens on the `127.0.0.1:50051` by default.
