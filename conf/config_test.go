package conf

import (
	"fmt"
	assert "github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	var driverConfig *Config
	configStr := "container_runtime: containerd\nset_io_limits: true\nread_iops_limit: [\"RioVolGroup=30\"]\nwrite_iops_limit: [\"RioVolGroup=20\"]\nbase_iops_limit: [\"RioVolGroup=2000\"]\nmax_iops_limit: [\"RioVolGroup=20000\"]\nread_bps_limit: [\"RioVolGroup=524288\"]\nwrite_bps_limit: [\"RioVolGroup=262144\"]\nbase_bps_limit: [\"RioVolGroup=104857600\"]\nmax_bps_limit: [\"RioVolGroup=314572800\"]\nmetrics_path: /metric\nmetrics_addr: 9099\nprobe_addr: 9098\ndisable_exporter_metrics: false\niscsi_username: rio-csi\niscsi_passwd: rio-123"
	err := yaml.Unmarshal([]byte(configStr), &driverConfig)
	assert.Nil(t, err)
	fmt.Println(driverConfig.ContainerRuntime, driverConfig.IscsiUsername)
}
