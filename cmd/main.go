package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/darkmuggle/arptracker/pkg/arping"
	"github.com/darkmuggle/arptracker/pkg/metrics"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootArgs holds the programs values
var rootArgs = struct {
	Interface string
	CIDR      string
	Port      int
}{}

var (
	cmdRoot = &cobra.Command{
		Use:   "watch",
		Short: "watch subnet",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if rootArgs.Interface == "" {
				logrus.Fatal("--interface must be defined")
			}
			if rootArgs.CIDR == "" {
				logrus.Fatal("--cidrs must be defined")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGTERM)

			hname, _ := os.Hostname()
			l := logrus.WithField("hostname", hname)

			a, err := arping.NewApringer(l, rootArgs.CIDR, rootArgs.Interface)
			if err != nil {
				l.WithError(err).Fatal("failed to allocate an arp client")
			}
			results, err := a.Run()
			if err != nil {
				l.WithError(err).Fatal("failed start arpinger")
			}

			termChan := make(chan struct{}, 1)
			if err := metrics.Watch(l, rootArgs.Port, results, termChan); err != nil {
				l.WithError(err).Fatal("failed to start watcher")
			}

			l.Info("No news is good news...")

			<-sigs
			termChan <- struct{}{}
		},
	}
)

func init() {
	cmdRoot.Flags().StringVar(&rootArgs.Interface, "interface", "", "interface to ping on")
	cmdRoot.Flags().StringVar(&rootArgs.CIDR, "cidrs", "", "cidrs to monitor")
	cmdRoot.Flags().IntVar(&rootArgs.Port, "port", 2113, "port to serve metrics on")
}

func main() {
	_ = cmdRoot.Execute()
}
