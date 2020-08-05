package igor

const PluginInitSymbol = "Init"

// PluginInit initializes the plugin with a function to emit events from and the path through the global state
// that contains the parts this plugin should get it's data from
type PluginInit func(func(Event), []string) IgorPlugin

type IgorPlugin interface {
	// UpdateState is called whenever the state the plugin is listening to changes
	// the state will be in a hierarchical structure like JSON and the path dictates
	// which part of the state the update argument should replace.  An empty path
	// indicates that the entire state for the plugin should be updated
	UpdateState(path []string, update []byte) error
}
