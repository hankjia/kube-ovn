package pinger

import (
	"flag"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"os"
	"strings"
	"time"
)

type Configuration struct {
	KubeConfigFile     string
	KubeClient         kubernetes.Interface
	Port               int
	DaemonSetNamespace string
	DaemonSetName      string
	Interval           int
	Mode               string
	InternalDNS        string
	ExternalDNS        string
	NodeName           string
	HostIP             string
	PodName            string
	PodIP              string
	ExternalAddress    string
	NetworkMode        string

	// Used for OVS Monitor
	PollTimeout                     int
	PollInterval                    int
	SystemRunDir                    string
	DatabaseVswitchName             string
	DatabaseVswitchSocketRemote     string
	DatabaseVswitchFileDataPath     string
	DatabaseVswitchFileLogPath      string
	DatabaseVswitchFilePidPath      string
	DatabaseVswitchFileSystemIDPath string
	ServiceVswitchdFileLogPath      string
	ServiceVswitchdFilePidPath      string
	ServiceOvnControllerFileLogPath string
	ServiceOvnControllerFilePidPath string
}

func ParseFlags() (*Configuration, error) {
	var (
		argPort               = pflag.Int("port", 8080, "metrics port")
		argKubeConfigFile     = pflag.String("kubeconfig", "", "Path to kubeconfig file with authorization and master location information. If not set use the inCluster token.")
		argDaemonSetNameSpace = pflag.String("ds-namespace", "kube-system", "kube-ovn-pinger daemonset namespace")
		argDaemonSetName      = pflag.String("ds-name", "kube-ovn-pinger", "kube-ovn-pinger daemonset name")
		argInterval           = pflag.Int("interval", 5, "interval seconds between consecutive pings")
		argMode               = pflag.String("mode", "server", "server or job Mode")
		argInternalDns        = pflag.String("internal-dns", "kubernetes.default", "check dns from pod")
		argExternalDns        = pflag.String("external-dns", "alauda.cn", "check external dns resolve from pod")
		argExternalAddress    = pflag.String("external-address", "114.114.114.114", "check ping connection to an external address, default empty that will disable external check")
		argNetworkMode        = pflag.String("network-mode", "kube-ovn", "The cni plugin current cluster used, default: kube-ovn")

		argPollTimeout                     = pflag.Int("ovs.timeout", 2, "Timeout on JSON-RPC requests to OVS.")
		argPollInterval                    = pflag.Int("ovs.poll-interval", 15, "The minimum interval (in seconds) between collections from OVS server.")
		argSystemRunDir                    = pflag.String("system.run.dir", "/var/run/openvswitch", "OVS default run directory.")
		argDatabaseVswitchName             = pflag.String("database.vswitch.name", "Open_vSwitch", "The name of OVS db.")
		argDatabaseVswitchSocketRemote     = pflag.String("database.vswitch.socket.remote", "unix:/var/run/openvswitch/db.sock", "JSON-RPC unix socket to OVS db.")
		argDatabaseVswitchFileDataPath     = pflag.String("database.vswitch.file.data.path", "/etc/openvswitch/conf.db", "OVS db file.")
		argDatabaseVswitchFileLogPath      = pflag.String("database.vswitch.file.log.path", "/var/log/openvswitch/ovsdb-server.log", "OVS db log file.")
		argDatabaseVswitchFilePidPath      = pflag.String("database.vswitch.file.pid.path", "/var/run/openvswitch/ovsdb-server.pid", "OVS db process id file.")
		argDatabaseVswitchFileSystemIDPath = pflag.String("database.vswitch.file.system.id.path", "/etc/openvswitch/system-id.conf", "OVS system id file.")

		argServiceVswitchdFileLogPath      = pflag.String("service.vswitchd.file.log.path", "/var/log/openvswitch/ovs-vswitchd.log", "OVS vswitchd daemon log file.")
		argServiceVswitchdFilePidPath      = pflag.String("service.vswitchd.file.pid.path", "/var/run/openvswitch/ovs-vswitchd.pid", "OVS vswitchd daemon process id file.")
		argServiceOvnControllerFileLogPath = pflag.String("service.ovncontroller.file.log.path", "/var/log/ovn/ovn-controller.log", "OVN controller daemon log file.")
		argServiceOvnControllerFilePidPath = pflag.String("service.ovncontroller.file.pid.path", "/var/run/ovn/ovn-controller.pid", "OVN controller daemon process id file.")
	)

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	// Sync the glog and klog flags.
	flag.CommandLine.VisitAll(func(f1 *flag.Flag) {
		f2 := klogFlags.Lookup(f1.Name)
		if f2 != nil {
			value := f1.Value.String()
			if err := f2.Value.Set(value); err != nil {
				klog.Fatalf("failed to set flag %v", err)
			}
		}
	})

	pflag.CommandLine.AddGoFlagSet(klogFlags)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	config := &Configuration{
		KubeConfigFile:     *argKubeConfigFile,
		KubeClient:         nil,
		Port:               *argPort,
		DaemonSetNamespace: *argDaemonSetNameSpace,
		DaemonSetName:      *argDaemonSetName,
		Interval:           *argInterval,
		Mode:               *argMode,
		InternalDNS:        *argInternalDns,
		ExternalDNS:        *argExternalDns,
		PodIP:              os.Getenv("POD_IP"),
		HostIP:             os.Getenv("HOST_IP"),
		NodeName:           os.Getenv("NODE_NAME"),
		PodName:            os.Getenv("POD_NAME"),
		ExternalAddress:    *argExternalAddress,
		NetworkMode:        *argNetworkMode,

		// OVS Monitor
		PollTimeout:                     *argPollTimeout,
		PollInterval:                    *argPollInterval,
		SystemRunDir:                    *argSystemRunDir,
		DatabaseVswitchName:             *argDatabaseVswitchName,
		DatabaseVswitchSocketRemote:     *argDatabaseVswitchSocketRemote,
		DatabaseVswitchFileDataPath:     *argDatabaseVswitchFileDataPath,
		DatabaseVswitchFileLogPath:      *argDatabaseVswitchFileLogPath,
		DatabaseVswitchFilePidPath:      *argDatabaseVswitchFilePidPath,
		DatabaseVswitchFileSystemIDPath: *argDatabaseVswitchFileSystemIDPath,
		ServiceVswitchdFileLogPath:      *argServiceVswitchdFileLogPath,
		ServiceVswitchdFilePidPath:      *argServiceVswitchdFilePidPath,
		ServiceOvnControllerFileLogPath: *argServiceOvnControllerFileLogPath,
		ServiceOvnControllerFilePidPath: *argServiceOvnControllerFilePidPath,
	}
	if err := config.initKubeClient(); err != nil {
		return nil, err
	}

	if config.Mode == "job" {
		ds, err := config.KubeClient.AppsV1().DaemonSets(config.DaemonSetNamespace).Get(config.DaemonSetName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		for _, arg := range ds.Spec.Template.Spec.Containers[0].Command {
			arg = strings.Trim(arg, "\"")
			if strings.HasPrefix(arg, "--external-address=") {
				config.ExternalAddress = strings.TrimPrefix(arg, "--external-address=")
			}
			if strings.HasPrefix(arg, "--external-dns=") {
				config.ExternalDNS = strings.TrimPrefix(arg, "--external-dns=")
			}
		}
	}
	klog.Infof("pinger config is %+v", config)
	return config, nil
}

func (config *Configuration) initKubeClient() error {
	var cfg *rest.Config
	var err error
	if config.KubeConfigFile == "" {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			klog.Errorf("use in cluster config failed %v", err)
			return err
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", config.KubeConfigFile)
		if err != nil {
			klog.Errorf("use --kubeconfig %s failed %v", config.KubeConfigFile, err)
			return err
		}
	}
	cfg.Timeout = 15 * time.Second
	cfg.QPS = 1000
	cfg.Burst = 2000
	cfg.ContentType = "application/vnd.kubernetes.protobuf"
	cfg.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Errorf("init kubernetes client failed %v", err)
		return err
	}
	config.KubeClient = kubeClient
	return nil
}
