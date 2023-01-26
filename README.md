## arpwatcher

arpwatcher is a daemon to detect for duplicate IP addresses on a network.

### why not just like arping or nmap or tool?

- arpwatch, the linux daemon, is good but you have to look at the logs
- arping is a CLI only tool

For my use case, I needed the ability to _visualize_ the results via Grafana.
Also, I wanted to have AlertManager signal if we had duplicate IP's. 

### why not just configure your network properly

People. In all seriousness, this was written to monitor an R&D lab where
well-meaning engineers might make a mistake.

### what does it do?

At its core, it:
- does 10 arp requests to a taget
- outputs the results if a duplicate is seen
- exports prometheus metrics on port 2112 

```
$ sudo ./arpwatcher.linux  --interface br0 --cidrs 10.25.50.0/23
INFO[0000] host IP                                       cidr=10.25.50.0/23 func=pinger hostname=d01 interface=br0 ip addresses="[10.25.51.4/23]"
INFO[0000] starting pinger                               cidr=10.25.50.0/23 count=510 func=pinger hostname=devops-services-01 interface=br0
WARN[0095] duplicate mac addresss responded              cidr=10.25.50.0/23 duplicates=2 func=pinger hostname=devops-services-01 interface=br0 ip=10.25.50.91
WARN[0095]                                               cidr=10.25.50.0/23 count=4 func=pinger hostname=d01 hw addr="00:00:00:01:00:00" interface=br0 ip=10.25.50.91
WARN[0095]                                               cidr=10.25.50.0/23 count=7 func=pinger hostname=d01 hw addr="00:00:00:02:aa:aa" interface=br0 ip=10.25.50.91
```