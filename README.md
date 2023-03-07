# p2p chat app with libp2p relay

This program demonstrates a simple p2p chat application using a relay. It can work between two peers even if neither has a public IP.

## Build
There are two aspects to it: 
1. a `relay` application (in directory `relayserver`) and
2. a `chat` application in `chatclient`

Create the binaries `relay` and `chat` in these two directories with `go build -o <binary name>`. Let's Assume these are called `relay` and `chat`, respectively. 

## Usage  
1. Install the relay in a location with a public ip address.  Have the client binaries in locations A and B, (possibly different terminals on the same machine). 
2. Run `./relay` at the relay site.
3. At client site A, run `./chat -r <RELAY>` where this is the multiaddress presented by the relay.
4. At client site B, run  `./chat -r <RELAY> -t <SITE_A>` ,  where <SITE_A> is the multiaddress published at the first client site console. Both consoles will show a `>` prompt and allow chat.
   
## Notes

This was developed from the [excellent circuitv2 example](https://github.com/libp2p/go-libp2p/tree/master/examples/relay) on the go-libp2p site.

## Todo
* Add peer discovery
* Remove limits on the relay (looks easy - use an opt) 
* Add pubsub

## Author
Busiso Chisala



