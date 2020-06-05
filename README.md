# Igor Home Automation
Plug and play home automation solution.  If a node fails just plug in another one and the system should continue with no disruption.

This is achieved through the use of rqlite which is a distributed, sqlite-like storage engine.  All services record their actions and observed data in the local instance and rely on rqlite to distribute the data so if a node fails or is stopped for some reason all of the other nodes should have a copy of the data from the stopped node.

Communication is performed via `nats-server` 