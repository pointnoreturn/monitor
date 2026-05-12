package meshtastic

import (
	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
)

// firmware consts for want_config_id:
// #define SPECIAL_NONCE_ONLY_CONFIG 69420
// #define SPECIAL_NONCE_ONLY_NODES 69421 // ( ͡° ͜ʖ ͡°)
const ConfigId_OnlyNodes = 69421
const ConfigId_ConfigOnly = 69420

var corePortNames = map[pb.PortNum]string{
	0:                                      "UNKNOWN_APP", // deprecated pb.PortNum_UNKNOWN_APP
	pb.PortNum_TEXT_MESSAGE_APP:            "TEXT_MESSAGE_APP",
	pb.PortNum_REMOTE_HARDWARE_APP:         "REMOTE_HARDWARE_APP",
	pb.PortNum_POSITION_APP:                "POSITION_APP",
	pb.PortNum_NODEINFO_APP:                "NODEINFO_APP",
	pb.PortNum_ROUTING_APP:                 "ROUTING_APP",
	pb.PortNum_ADMIN_APP:                   "ADMIN_APP",
	pb.PortNum_TEXT_MESSAGE_COMPRESSED_APP: "TEXT_MESSAGE_COMPRESSED_APP",
	pb.PortNum_WAYPOINT_APP:                "WAYPOINT_APP",
	pb.PortNum_AUDIO_APP:                   "AUDIO_APP",
	pb.PortNum_DETECTION_SENSOR_APP:        "DETECTION_SENSOR_APP",
	pb.PortNum_ALERT_APP:                   "ALERT_APP",
	pb.PortNum_KEY_VERIFICATION_APP:        "KEY_VERIFICATION_APP",
	pb.PortNum_REMOTE_SHELL_APP:            "REMOTE_SHELL_APP",
	pb.PortNum_REPLY_APP:                   "REPLY_APP",
	pb.PortNum_IP_TUNNEL_APP:               "IP_TUNNEL_APP",
	pb.PortNum_PAXCOUNTER_APP:              "PAXCOUNTER_APP",
	pb.PortNum_STORE_FORWARD_PLUSPLUS_APP:  "STORE_FORWARD_PLUSPLUS_APP",
	pb.PortNum_NODE_STATUS_APP:             "NODE_STATUS_APP",
	pb.PortNum_SERIAL_APP:                  "SERIAL_APP",
	pb.PortNum_STORE_FORWARD_APP:           "STORE_FORWARD_APP",
	pb.PortNum_RANGE_TEST_APP:              "RANGE_TEST_APP",
	pb.PortNum_TELEMETRY_APP:               "TELEMETRY_APP",
	pb.PortNum_ZPS_APP:                     "ZPS_APP",
	pb.PortNum_SIMULATOR_APP:               "SIMULATOR_APP",
	pb.PortNum_TRACEROUTE_APP:              "TRACEROUTE_APP",
	pb.PortNum_NEIGHBORINFO_APP:            "NEIGHBORINFO_APP",
	pb.PortNum_ATAK_PLUGIN:                 "ATAK_PLUGIN",
	pb.PortNum_MAP_REPORT_APP:              "MAP_REPORT_APP",
	pb.PortNum_POWERSTRESS_APP:             "POWERSTRESS_APP",
	pb.PortNum_LORAWAN_BRIDGE:              "LORAWAN_BRIDGE",
	pb.PortNum_RETICULUM_TUNNEL_APP:        "RETICULUM_TUNNEL_APP",
	pb.PortNum_CAYENNE_APP:                 "CAYENNE_APP",
	pb.PortNum_ATAK_PLUGIN_V2:              "ATAK_PLUGIN_V2",
	pb.PortNum_GROUPALARM_APP:              "GROUPALARM_APP",
	pb.PortNum_PRIVATE_APP:                 "PRIVATE_APP",
	pb.PortNum_ATAK_FORWARDER:              "ATAK_FORWARDER",
	// MAX
}

// maps port number to common name (core ports only)
func GetCorePortName(portnum pb.PortNum) (string, bool) {
	v, ok := corePortNames[portnum]
	return v, ok
}
