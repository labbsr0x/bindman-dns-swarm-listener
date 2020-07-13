package listener

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	bindmanManagerAddress = "manager-address"
	reverseProxyAddress   = "reverse-proxy-address"
	tags                  = "tags"
)

// AddFlags adds flags for Options.
func AddFlags(flags *pflag.FlagSet) {
	flags.String(bindmanManagerAddress, "", "The address of the DNS Manager which will manage the identified DNS updates")
	flags.String(reverseProxyAddress, "", "The address of the Reverse Proxy which will load balance requests to the Sandman managed hostnames")
	flags.StringArray(tags, nil, "A comma-separated list of dns tags enabling this listener to choose which service updates its dns manager should deal with")
}

// InitFromViper initializes Options with properties retrieved from Viper.
func (b *Builder) InitFromViper(v *viper.Viper) *Builder {
	b.BindmanManagerAddress = v.GetString(bindmanManagerAddress)
	b.ReverseProxyAddress = v.GetString(reverseProxyAddress)
	b.Tags = v.GetStringSlice(tags)
	return b
}
