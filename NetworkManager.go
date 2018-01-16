package gonetworkmanager

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	"github.com/godbus/dbus"
	"github.com/satori/go.uuid"
)

const (
	NetworkManagerInterface  = "org.freedesktop.NetworkManager"
	NetworkManagerObjectPath = "/org/freedesktop/NetworkManager"

	NetworkManagerGetDevices               = NetworkManagerInterface + ".GetDevices"
	NetworkManagerPropertyState            = NetworkManagerInterface + ".State"
	NetworkManagerAddAndActivateConnection = NetworkManagerInterface + ".AddAndActivateConnection"

	NetworkManagerWifiScan = NetworkManagerInterface + ".Device.Wireless" + ".RequestScan"
)

type IpProxyConfig struct {
	Ip          string
	Prefix      string
	Gateway     string
	Dns_server  string
	Http_proxy  string
	Https_proxy string
}

type NetworkManager interface {

	// GetDevices gets the list of network devices.
	GetDevices() []Device

	// GetState returns the overall networking state as determined by the
	// NetworkManager daemon, based on the state of network devices under it's
	// management.
	GetState() NmState

	Subscribe() <-chan *dbus.Signal
	Unsubscribe()

	AddWirelessConnection(name, password string) dbus.ObjectPath
	AddWiredConnection(manual bool, config IpProxyConfig) dbus.ObjectPath

	MarshalJSON() ([]byte, error)
}

func NewNetworkManager() (NetworkManager, error) {
	var nm networkManager
	return &nm, nm.init(NetworkManagerInterface, NetworkManagerObjectPath)
}

type networkManager struct {
	dbusBase

	sigChan chan *dbus.Signal
}

func (n *networkManager) AddWiredConnection(manual bool, config IpProxyConfig) dbus.ObjectPath {

	var ret1 dbus.ObjectPath
	var ret2 dbus.ObjectPath
	var ret []interface{}

	ret = append(ret, &ret1)
	ret = append(ret, &ret2)

	var dev Device

	devFound := false

	for _, dev = range n.GetDevices() {
		if dev.GetDeviceType() == NmDeviceTypeEthernet {
			devFound = true
			fmt.Println("Found eth device ", dev)
			break
		}
	}

	if !devFound {
		return *(ret[0].(*dbus.ObjectPath))
	}

	settings := make(ConnectionSettings)

	ip := net.ParseIP(config.Ip)
	prefix, _ := strconv.ParseUint(config.Prefix, 10, 32)
	gateway := net.ParseIP(config.Gateway)
	dns := net.ParseIP(config.Dns_server)

	settings["802-3-ethernet"] = make(map[string]interface{})
	settings["ipv4"] = make(map[string]interface{})
	settings["connection"] = make(map[string]interface{})

	settings["802-3-ethernet"]["duplex"] = "full"

	id := uuid.Must(uuid.NewV4())
	settings["connection"]["id"] = "MyWiredConnection"
	settings["connection"]["uuid"] = id.String()
	settings["connection"]["type"] = "802-3-ethernet"

	var addrs [][]uint32
	addrs = append(addrs, []uint32{ip2int(ip), uint32(prefix), ip2int(gateway)})

	var dns_addrs []uint32
	dns_addrs = append(dns_addrs, ip2int(dns))

	settings["ipv4"]["addresses"] = addrs
	//settings["ipv4"]["gateway"] = gateway.String()
	if manual {
		settings["ipv4"]["method"] = "manual"
	} else {
		settings["ipv4"]["method"] = "auto"
	}
	settings["ipv4"]["dns"] = dns_addrs
	settings["ipv4"]["may-fail"] = true

	// ignored for wired connections https://people.freedesktop.org/~lkundrak/nm-docs/gdbus-org.freedesktop.NetworkManager.html#gdbus-method-org-freedesktop-NetworkManager.AddAndActivateConnection
	conn := "/"

	n.callMultipleResults(ret, NetworkManagerAddAndActivateConnection, settings, dev.GetObjectPath(), dbus.ObjectPath(conn))
	return *(ret[0].(*dbus.ObjectPath))
}

func (n *networkManager) AddWirelessConnection(name, password string) dbus.ObjectPath {

	var ret1 dbus.ObjectPath
	var ret2 dbus.ObjectPath

	var ret []interface{}

	ret = append(ret, &ret1)
	ret = append(ret, &ret2)

	settings := make(ConnectionSettings)
	settings["802-11-wireless"] = make(map[string]interface{})
	settings["802-11-wireless-security"] = make(map[string]interface{})
	settings["connection"] = make(map[string]interface{})

	settings["802-11-wireless"]["ssid"] = []byte(name)
	//settings["802-11-wireless"]["security"] = "802-11-wireless-security"
	settings["802-11-wireless-security"]["psk"] = password

	settings["connection"]["id"] = name
	settings["connection"]["type"] = "802-11-wireless"

	var dev Device
	var conn AccessPoint

	devFound := false
	connFound := false

	for _, dev = range n.GetDevices() {
		if dev.GetDeviceType() == NmDeviceTypeWifi {
			devFound = true
			break
		}
	}

	if !devFound {
		return *(ret[0].(*dbus.ObjectPath))
	}

	wireless_dev, _ := NewWirelessDevice(dev.GetObjectPath())

	// scan wifi, get path of network you want to conenct to
	for _, conn = range wireless_dev.GetAccessPoints() {
		if conn.GetSSID() == name {
			connFound = true
			break
		}
	}

	if !connFound {
		return *(ret[0].(*dbus.ObjectPath))
	}

	n.callMultipleResults(ret, NetworkManagerAddAndActivateConnection, settings, dev.GetObjectPath(), conn.GetObjectPath())
	return *(ret[0].(*dbus.ObjectPath))
}

func (n *networkManager) GetDevices() []Device {
	var devicePaths []dbus.ObjectPath

	n.call(&devicePaths, NetworkManagerGetDevices)
	devices := make([]Device, len(devicePaths))

	var err error
	for i, path := range devicePaths {
		devices[i], err = DeviceFactory(path)
		if err != nil {
			panic(err)
		}
	}

	return devices
}

func (n *networkManager) GetState() NmState {
	return NmState(n.getUint32Property(NetworkManagerPropertyState))
}

func (n *networkManager) Subscribe() <-chan *dbus.Signal {
	if n.sigChan != nil {
		return n.sigChan
	}

	n.subscribeNamespace(NetworkManagerObjectPath)
	n.sigChan = make(chan *dbus.Signal, 10)
	n.conn.Signal(n.sigChan)

	return n.sigChan
}

func (n *networkManager) Unsubscribe() {
	n.conn.RemoveSignal(n.sigChan)
	n.sigChan = nil
}

func (n *networkManager) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"NetworkState": n.GetState().String(),
		"Devices":      n.GetDevices(),
	})
}
