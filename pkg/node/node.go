package node

type Node struct {
	Name             string
	IPv4             string
	Port             int
	SchedulerAddress string
}

/*
Proxy:

- proxy pass to the upstream, should filter our every request that meets a certain regex
- node client/core should have:
	methods to
		register()
		notifyLayer()
		deregister()
		removeLayer()
		addConnection()
		removeConnection()
		getPeer()
		download()
		garbageCollector() // spin up in separate go-routine
- fileserver
	if fileserver is requested trigger addConnection()

*/
