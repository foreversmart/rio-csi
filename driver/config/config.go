package config

// Config struct define how driver running
type Config struct {
	// Namespace string
	Namespace string `yaml:"namespace"`
	// SetIOLimits if set to true, directs the driver
	// to set iops, bps limits on a pod using a volume
	// provisioned on its node. For this to work,
	// CSIDriver.Spec.podInfoOnMount must be set to 'true'
	SetIOLimits bool `yaml:"set_io_limits"`
	// ContainerRuntime informs the driver of the container runtime
	// used on the node, so that the driver can make assumptions
	// about the cgroup path for a pod requesting a volume mount
	ContainerRuntime string `yaml:"container_runtime"`

	// ReadIopsLimit provides read iops rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	ReadIopsLimit *[]string `yaml:"read_iops_limit"`

	// WriteIopsLimit provides write iops rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	WriteIopsLimit *[]string

	// ReadBpsLimit provides read bps rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	ReadBpsLimit *[]string

	// WriteBpsLimit provides read bps rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	WriteBpsLimit *[]string

	// The HTTP path where prometheus metrics will be exposed. Default is `/metrics`.
	MetricsPath string

	// MetricsAddr specific metric server listen addr
	MetricsAddr string
	// MetricsAddr specific probe server listen addr
	ProbeAddr string

	// Exclude metrics about the exporter itself (process_*, go_*).
	DisableExporterMetrics bool

	// IscsiUsername specific iscsi username for iscsi auth
	IscsiUsername string
	// IscsiPasswd specific iscsi password for iscsi auth
	IscsiPasswd string
}
