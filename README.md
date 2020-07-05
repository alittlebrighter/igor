# Igor Home Automation
Plug and play home automation solution.  If a node fails just plug in another one and the system should continue with little to no disruption.

This is achieved through a combination of [lansrv](https://github.com/alittlebrighter/lansrv), [nats-server](https://github.com/nats-io/nats-server), and [rqlite](https://github.com/rqlite/rqlite) (untested).  When a node starts up, `nats-server` and `rqlite` use `lansrv` to find all of the other active nodes on the LAN and join the cluster (or start their own if there are no available nodes).  Events are published to the local `nats-server` and recorded in `rqlite`.  The respective clusters then distribute the data to the other nodes on the LAN.

## Architecture
Home automation turns your home into one big user interface.  The flux architecture is the best I've seen for managing user interfaces and Igor seeks to apply the concepts of flux to home automation.  Specifically, it seeks to use the flux architecture as implemented by NGRX which adds the concept of effects which are effectively non-deterministic reducers.

1. Igor starts from a base state where no default values initialize the system.
2. Actions/events are generated
   1. user actions involve changing settings (max/min temperature settings, triggering garage door, etc.)
   2. sensor actions report on how the environment has changed
3. Effects (updating settings in a file/db, toggle GPIO pin, etc.) run based on the new proposed state and emit new actions
4. Reducers are run to compute the desired state.
5. The new state is published to the system.
6. Controllers read the new state and run the necessary commands to their attached devices and report any errors as events returning to step #2.

Note: since the system communicates via NATS so update broadcasts can be sent specifically so `{a: 1, b: {c: 2, d: 3}}` where only `c` is updated can be broadcast on `state.b.c` with a message of `4` (or whatever the value was updated to).

Reducers are written in Typescript that gets transpiled to Javascript and run via [Goja](https://github.com/dop251/goja)

## Roadmap
- [] security
- [x] choose serialization format -- JSON
- [] come up with common metadata for messages (started in `capnp_models`)

## Contributing
Please do!  PRs are certainly welcome.  Note, this project is intended to be the base for a home automation system and will not house device drivers, user interfaces, etc.  Those can exist in their own repositories.
