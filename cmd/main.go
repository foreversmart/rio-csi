package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"qiniu.io/rio-csi/driver"
	"runtime"
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
	debug    bool
	name     string
	endpoint string
	nodeID   string

	enableIdentityServer   bool
	enableControllerServer bool
	enableNodeServer       bool
)

var (
	rootCmd = &cobra.Command{
		Use:     "rio-csi",
		Short:   "CSI based rio csi driver",
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			logrus.Info("start ")
			driver.NewCSIDriver(
				name,
				Version,
				nodeID,
				endpoint,
				enableIdentityServer,
				enableControllerServer,
				enableNodeServer,
			).Run()
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

	rootCmd.PersistentFlags().BoolVar(&enableIdentityServer, "enable-identity-server", false, "Enable Identity gRPC Server")
	rootCmd.PersistentFlags().BoolVar(&enableControllerServer, "enable-controller-server", false, "Enable Controller gRPC Server")
	rootCmd.PersistentFlags().BoolVar(&enableNodeServer, "enable-node-server", false, "Enable Node gRPC Server")

	rootCmd.PersistentFlags().StringVar(&nodeID, "nodeid", "", "CSI Node ID")
	_ = rootCmd.MarkPersistentFlagRequired("nodeid")

	rootCmd.PersistentFlags().StringVar(&endpoint, "endpoint", "unix:///csi/csi.sock", "CSI gRPC Server Endpoint")
	_ = rootCmd.MarkPersistentFlagRequired("endpoint")

	rootCmd.PersistentFlags().StringVar(&name, "name", "rio-csi", "CSI Driver Name")
	_ = rootCmd.PersistentFlags().MarkHidden("name")

	rootCmd.SetVersionTemplate(fmt.Sprintf(versionTpl, name, Version, runtime.GOOS+"/"+runtime.GOARCH, BuildDate, CommitID))
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
