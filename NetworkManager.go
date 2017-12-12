package gonetworkmanager

import (
	"encoding/json"

	"github.com/godbus/dbus"
)

const (
	NetworkManagerInterface  = "org.freedesktop.NetworkManager"
	NetworkManagerObjectPath = "/org/freedesktop/NetworkManager"

	NetworkManagerGetDevices               = NetworkManagerInterface + ".GetDevices"
	NetworkManagerPropertyState            = NetworkManagerInterface + ".State"
	NetworkManagerAddAndActivateConnection = NetworkManagerInterface + ".AddAndActivateConnection"

	NetworkManagerWifiScan = NetworkManagerInterface + ".Device.Wireless" + ".RequestScan"
)

type NetworkManager interface {

	// GetDevices gets the list of network devices.
	GetDevices() []Device

	// GetState returns the overall networking state as determined by the
	// NetworkManager daemon, based on the state of network devices under it's
	// management.
	GetState() NmState

	Subscribe() <-chan *dbus.Signal
	Unsubscribe()

	AddConnection(name, password string)

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

func (n *networkManager) AddConnection(name, password string) {

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

	for _, dev = range n.GetDevices() {
		if dev.GetDeviceType() == NmDeviceTypeWifi {
			break
		}
	}

	wireless_dev, _ := NewWirelessDevice(dev.GetObjectPath())

	// scan wifi, get path of network you want to conenct to
	for _, conn = range wireless_dev.GetAccessPoints() {
		if conn.GetSSID() == name {
			break
		}
	}

	n.callMultipleResults(ret, NetworkManagerAddAndActivateConnection, settings, dev.GetObjectPath(), conn.GetObjectPath())
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
