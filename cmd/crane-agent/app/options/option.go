package options

import (
	"github.com/spf13/pflag"
)

// Options hold the command-line options about crane manager
type Options struct {
	// HostnameOverride is the name of k8s node
	HostnameOverride string
	// RuntimeEndpoint is the endpoint of runtime
	RuntimeEndpoint string
	// Ifaces is the network devices to collect metric
	Ifaces []string
}

// NewOptions builds an empty options.
func NewOptions() *Options {
	return &Options{}
}

// Complete completes all the required options.
func (o *Options) Complete() error {
	return nil
}

// Validate all required options.
func (o *Options) Validate() error {
	return nil
}

// AddFlags adds flags to the specified FlagSet.
func (o *Options) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.HostnameOverride, "hostname-override", "", "Which is the name of k8s node be used to filtered.")
	flags.StringVar(&o.RuntimeEndpoint, "runtime-endpoint", "unix:///var/run/dockershim.sock", "The runtime endpoint, default: unix:///var/run/dockershim.sock.")
	flags.StringArrayVar(&o.Ifaces, "ifaces", []string{"eth0"}, "The network devices to collect metric, use comma to separated, default: eth0")
}
