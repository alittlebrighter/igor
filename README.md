# Igor Home Automation
Plug and play home automation solution.  If a node fails just plug in another one and the system should continue with little to no disruption.

This is achieved through a combination of [lansrv](https://github.com/alittlebrighter/lansrv), [nats-server](https://github.com/nats-io/nats-server), and [rqlite (untested)](https://github.com/rqlite/rqlite).  When a node starts up, `nats-server` and `rqlite` use `lansrv` to find all of the other active nodes on the LAN and join the cluster (or start their own if there are no available nodes).  Events are published to the local `nats-server` and recorded in `rqlite`.  The respective clusters then distribute the data to the other nodes on the LAN.

## Roadmap
- [] security
- [] choose serialization format ([capnp](https://capnproto.org/) models started)
- [] come up with common metadata for messages (started in `capnp_models`)

## Contributing
Please do!  PRs are certainly welcome.  Note, this project is intended to be the base for a home automation system and will not house device drivers, user interfaces, etc.  Those can exist in their own repositories.
