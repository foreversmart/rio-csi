package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"qiniu.io/rio-csi/driver"
	"qiniu.io/rio-csi/iscsi"
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

	iscsiUsername string
	iscsiPasswd   string

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
			driverType = DriverType(driverTypeStr)
			logrus.Info("start ", driverType, nodeID, endpoint, iscsiUsername)

			switch driverType {
			case DriverTypeNode:
				// init iscsi server
				target := ""
				targets, err := iscsi.ListTarget()
				if err != nil {
					logrus.Error(err)
					return
				}

				// pick default targets
				if len(targets) > 0 {
					target = targets[0]
					// TODO check acl rules
				} else {
					// create a target and set up
					target, err = iscsi.SetUpTarget("rio-csi", nodeID)
					if err != nil {
						logrus.Error(err)
						return
					}

					_, err = iscsi.SetUpTargetAcl(target, iscsiUsername, iscsiPasswd)
					if err != nil {
						logrus.Error(err)
						return
					}
				}

				logrus.Info("iscsi target:", target)

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

				manager.StartManager(nodeID, target, metricsAddr, probeAddr)

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

	rootCmd.PersistentFlags().StringVar(&driverTypeStr, "driverType", "node", "set driver type node or control")
	_ = rootCmd.MarkPersistentFlagRequired("driverType")

	rootCmd.PersistentFlags().StringVar(&iscsiUsername, "iscsiUsername", "", "set iscsi portal username")

	rootCmd.PersistentFlags().StringVar(&iscsiPasswd, "iscsiPasswd", "", "set iscsi portal password")

	rootCmd.PersistentFlags().StringVar(&metricsAddr, "metricsAddr", ":9180", "set metrics addr")
	rootCmd.PersistentFlags().StringVar(&probeAddr, "probeAddr", ":9181", "set probe addr")

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
