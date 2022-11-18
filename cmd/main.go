package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"qiniu.io/rio-csi/driver"
	"qiniu.io/rio-csi/manager"
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
	endpoint    string
	nodeID      string
	metricsAddr string
	probeAddr   string

	enableIdentityServer   bool
	enableControllerServer bool
	enableNodeServer       bool

	driverType DriverType
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
			logrus.Info("start ", driverType)

			switch driverType {
			case DriverTypeNode:
				go func() {
					driver.NewCSIDriver(
						name,
						Version,
						nodeID,
						endpoint,
						true,
						false,
						true,
					).Run()
				}()

				manager.StartManager(nodeID, metricsAddr, probeAddr)

			case DriverTypeControl:

				driver.NewCSIDriver(
					name,
					Version,
					nodeID,
					endpoint,
					true,
					true,
					false,
				).Run()
			}

		},
	}
)

func init() {
	cobra.OnInitialize(initLog)
	setRootCMD()
}

func main() {
	Execute()
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
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

	rootCmd.PersistentFlags().StringVar(&Version, "version", "v1.0", "CSI Driver Version")
	_ = rootCmd.PersistentFlags().MarkHidden("version")

	dt := ""
	rootCmd.PersistentFlags().StringVar(&dt, "driverType", "node", "set driver type node or control")
	_ = rootCmd.MarkPersistentFlagRequired("driverType")
	driverType = DriverType(dt)

	rootCmd.PersistentFlags().StringVar(&metricsAddr, "metricsAddr", ":9180", "set metrics addr")
	rootCmd.PersistentFlags().StringVar(&probeAddr, "probeAddr", "9181", "set probe addr")

	//rootCmd.SetVersionTemplate(fmt.Sprintf(versionTpl, name, Version, runtime.GOOS+"/"+runtime.GOARCH, BuildDate, CommitID))
}

func initLog() {
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}
