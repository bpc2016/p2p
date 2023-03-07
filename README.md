# p2p chat app with libp2p relay

This program demonstrates a simple p2p chat application using a relay. It can work between two peers even if neither has a public IP. The relay, of course, must be publicly accessible for ths to work behind NATs.

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
   
## Sample 
On the server console ... you will need access to port 8919 (set in the relay code)
```
# ./relay
	2023/03/07 07:17:08 Relay is: /ip4/111.222.333.444/tcp/8919/p2p/QmQoAyYiDawoTgbkhUpevLBRtrmwUQ6rhPMffhKKnxGX7K
	Use this address in setting up relay services
```
At site 'A', use the relay multiaddress above:
```
$ ./chat -r /ip4/111.222.333.444/tcp/8919/p2p/QmQoAyYiDawoTgbkhUpevLBRtrmwUQ6rhPMffhKKnxGX7K
	2023/03/07 09:26:52 I am host: /p2p/12D3KooWQZEZDx5q26iGwb69Pz289Qo5cnQtjtwWC1w1n727iVJw
```
Note the multiaddress in the console above. At site 'B' we use both addresses:
```
$ ./chat -r /ip4/111.222.333.444/tcp/8919/p2p/QmQoAyYiDawoTgbkhUpevLBRtrmwUQ6rhPMffhKKnxGX7K  -t /p2p/12D3KooWQZEZDx5q26iGwb69Pz289Qo5cnQtjtwWC1w1n727iVJw
	2023/03/07 09:28:37 I am host: /p2p/12D3KooWHKeiWwJFKXYgJjuaASYvAtqiGqS3bFy3mQEHqB6skxpM	
```
After a brief lapse, both terminals show a text input prompt.
## Notes

This was developed from the [excellent circuitv2 example](https://github.com/libp2p/go-libp2p/tree/master/examples/relay) on the go-libp2p site. In particular, the clients do not provide ports! 

## Todo
* Add peer discovery
* Add pubsub
* Where is the hole-punching example to build on ??

## Author
Busiso Chisala



