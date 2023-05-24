package conf

// Config struct define how driver running
type Config struct {
	// ContainerRuntime informs the driver of the container runtime
	// used on the node, so that the driver can make assumptions
	// about the cgroup path for a pod requesting a volume mount
	ContainerRuntime string `yaml:"container_runtime"`

	// SetIOLimits if set to true, directs the driver
	// to set iops, bps limits on a pod using a volume
	// provisioned on its node. For this to work,
	// CSIDriver.Spec.podInfoOnMount must be set to 'true'
	SetIOLimits bool `yaml:"set_io_limits"`

	// ReadIopsLimit provides read iops rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	// read iops limit formula: min{(BaseIopsLimit + ReadIopsLimit * GB), MaxIopsLimit}
	ReadIopsLimit *[]string `yaml:"read_iops_limit"`

	// WriteIopsLimit provides write iops rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	WriteIopsLimit *[]string `yaml:"write_iops_limit"`

	// BaseIopsLimit provides write base iops rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	BaseIopsLimit *[]string `yaml:"base_iops_limit"`

	// MaxIopsLimit provides write max iops rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	MaxIopsLimit *[]string `yaml:"max_iops_limit"`

	// ReadBpsLimit provides read bps rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	// read bps limit formula: min{(BaseBpsLimit + ReadBpsLimit * GB), MaxBpsLimit}
	ReadBpsLimit *[]string `yaml:"read_bps_limit"`

	// WriteBpsLimit provides read bps rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	WriteBpsLimit *[]string `yaml:"write_bps_limit"`

	// BaseBpsLimit provides write base bps rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	BaseBpsLimit *[]string `yaml:"base_iops_limit"`

	// MaxBpsLimit provides write max bps rate limits per GB in specific volume group type
	// as a string slice, in the form ["vg1-prefix=100", "vg2-prefix=200"]
	MaxBpsLimit *[]string `yaml:"max_iops_limit"`

	// The HTTP path where prometheus metrics will be exposed. Default is `/metrics`.
	MetricsPath string `yaml:"metrics_path"`

	// MetricsAddr specific metric server listen addr
	MetricsAddr string `yaml:"metrics_addr"`
	// MetricsAddr specific probe server listen addr
	ProbeAddr string `yaml:"probe_addr"`

	// Exclude metrics about the exporter itself (process_*, go_*).
	DisableExporterMetrics bool `yaml:"disable_exporter_metrics"`

	// IscsiUsername specific iscsi username for iscsi auth
	IscsiUsername string `yaml:"iscsi_username"`
	// IscsiPasswd specific iscsi password for iscsi auth
	IscsiPasswd string `yaml:"iscsi_passwd"`
}
