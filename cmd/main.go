package main

import (
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/conf"
	"qiniu.io/rio-csi/driver"
	"qiniu.io/rio-csi/lib/mount"
	"qiniu.io/rio-csi/logger"
	"qiniu.io/rio-csi/manager"
	"syscall"
	"time"
)

var (
	Version   string
	BuildDate string
	CommitID  string

	versionTpl = `
Name: %s
Version: %s
Arch: %s
BuildDate: %s
CommitID: %s
`
)

var (
	debug       bool
	name        string
	namespace   string
	endpoint    string
	nodeID      string
	metricsAddr string
	probeAddr   string

	//iscsiUsername string
	//iscsiPasswd   string

	driverType    DriverType
	driverTypeStr string
)

type DriverType string

const (
	DriverTypeNode    DriverType = "node"
	DriverTypeControl DriverType = "control"
)

var (
	rootCmd = &cobra.Command{
		Use:     "rio-csi",
		Short:   "CSI based rio csi driver",
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := conf.LoadConfig(namespace)
			if err != nil {
				logger.StdLog.Error(err)
				return
			}
			mount.SetIORateLimits(config)

			driverType = DriverType(driverTypeStr)
			logger.StdLog.Info("start ", driverType, nodeID, endpoint, config.IscsiUsername)

			switch driverType {
			case DriverTypeNode:

				go func() {
					driver.NewCSIDriver(
						name,
						Version,
						nodeID,
						endpoint,
						config.IscsiUsername,
						config.IscsiPasswd,
						true,
						false,
						true,
					).Run()
				}()

				manager.StartManager(nodeID, namespace, metricsAddr,
					probeAddr, config.IscsiUsername, config.IscsiPasswd, stopCh)

			case DriverTypeControl:

				driver.NewCSIDriver(
					name,
					Version,
					nodeID,
					endpoint,
					config.IscsiUsername,
					config.IscsiPasswd,
					true,
					true,
					false,
				).Run()
			}

		},
	}
)

func init() {
	setRootCMD()
}

var stopCh chan struct{}

func main() {
	client.SetupClusterConfig()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

		for {
			select {
			case <-sigs:
				logger.StdLog.Info("接受到了结束进程的信号")
				close(stopCh)
				time.Sleep(time.Second * 2)
				os.Exit(1)
			}
		}

	}()

	Execute()
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.StdLog.Fatal(err)
	}
}

func setRootCMD() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug log")

	rootCmd.PersistentFlags().StringVar(&nodeID, "nodeid", "", "CSI Node ID")
	_ = rootCmd.MarkPersistentFlagRequired("nodeid")

	rootCmd.PersistentFlags().StringVar(&endpoint, "endpoint", "unix:///csi/csi.sock", "CSI gRPC Server Endpoint")
	_ = rootCmd.MarkPersistentFlagRequired("endpoint")

	rootCmd.PersistentFlags().StringVar(&name, "name", "rio-csi", "CSI Driver Name")
	_ = rootCmd.PersistentFlags().MarkHidden("name")

	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "riocsi", "CSI Driver namespace")
	_ = rootCmd.MarkPersistentFlagRequired("namespace")
	_ = rootCmd.PersistentFlags().MarkHidden("namespace")

	rootCmd.PersistentFlags().StringVar(&Version, "version", "v1.0", "CSI Driver Version")
	_ = rootCmd.PersistentFlags().MarkHidden("version")

	rootCmd.PersistentFlags().StringVar(&driverTypeStr, "driverType", "node", "set driver type node or control")
	_ = rootCmd.MarkPersistentFlagRequired("driverType")

	rootCmd.PersistentFlags().StringVar(&metricsAddr, "metricsAddr", ":9180", "set metrics addr")
	rootCmd.PersistentFlags().StringVar(&probeAddr, "probeAddr", ":9181", "set probe addr")

	//rootCmd.SetVersionTemplate(fmt.Sprintf(versionTpl, name, Version, runtime.GOOS+"/"+runtime.GOARCH, BuildDate, CommitID))
}
