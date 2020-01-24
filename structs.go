package main

import "sync"

// data structures
type Settings struct {
	Location_confidence    int64 `json:"location_confidence"`
	Last_seen_threshold    int64 `json:"last_seen_threshold"`
	Beacon_metrics_size    int   `json:"beacon_metrics_size"`
	HA_send_interval       int64 `json:"ha_send_interval"`
	HA_send_changes_only   bool  `json:"ha_send_changes_only"`
	RSSI_min_threshold     int64 `json:"rssi_min_threshold"`
	RSSI_enforce_threshold bool  `json:"enforce_rssi_threshold"`
	Debug                  bool  `json:"debug"`
}

type Incoming_json struct {
	Name             string `json:"name"`
	Hostname         string `json:"hostname"`
	MAC              string `json:"mac"`
	RSSI             int64  `json:"rssi"`
	Is_scan_response string `json:"is_scan_response"`
	Ttype            string `json:"type"`
	Data             string `json:"data"`
	Beacon_type      string `json:"beacon_type"`
	UUID             string `json:"uuid"`
	Major            string `json:"major"`
	Minor            string `json:"minor"`
	TX_power         string `json:"tx_power"`
	Namespace        string `json:"namespace"`
	Instance_id      string `json:"instance_id"`
	// button stuff
	HB_ButtonCounter int64  `json:"hb_button_counter"`
	HB_Battery       int64  `json:"hb_button_battery"`
	HB_RandomNonce   string `json:"hb_button_random"`
	HB_ButtonMode    string `json:"hb_button_mode"`
}

type Advertisement struct {
	ttype   string
	content string
	seen    int64
}

type beacon_metric struct {
	location  string
	distance  float64
	rssi      int64
	timestamp int64
}

type Location struct {
	name string
	lock sync.RWMutex
}

type Best_location struct {
	distance  float64
	name      string
	last_seen int64
}

type HTTP_location struct {
	Distance       float64 `json:"distance"`
	Name           string  `json:"name"`
	Beacon_name    string  `json:"beacon_name"`
	Beacon_id      string  `json:"beacon_id"`
	Beacon_type    string  `json:"beacon_type"`
	HB_Battery     int64   `json:"hb_button_battery"`
	HB_ButtonMode  string  `json:"hb_button_mode"`
	Beacon_Enabled bool    `json:"beacon_enabled"`
	Location       string  `json:"location"`
	Last_seen      int64   `json:"last_seen"`
}

type Location_change struct {
	Beacon_ref        Beacon `json:"beacon_info"`
	Name              string `json:"name"`
	Beacon_name       string `json:"beacon_name"`
	Previous_location string `json:"previous_location"`
	New_location      string `json:"new_location"`
	Timestamp         int64  `json:"timestamp"`
}

type HA_message struct {
	Beacon_id   string  `json:"id"`
	Beacon_name string  `json:"name"`
	Distance    float64 `json:"distance"`
}

type HTTP_locations_list struct {
	Beacons []HTTP_location `json:"beacons"`
	Buttons []Button        `json:"buttons"`
}

type Beacon struct {
	Name                        string        `json:"name"`
	Beacon_id                   string        `json:"beacon_id"`
	Beacon_type                 string        `json:"beacon_type"`
	Beacon_location             string        `json:"beacon_location"`
	Last_seen                   int64         `json:"last_seen"`
	Incoming_JSON               Incoming_json `json:"incoming_json"`
	Distance                    float64       `json:"distance"`
	Beacon_Enabled              bool          `json:"beacon_enabled"`
	Previous_location           string
	Previous_confident_location string
	Location_confidence         int64
	beacon_metrics              []beacon_metric

	HB_ButtonCounter int64  `json:"hb_button_counter"`
	HB_Battery       int64  `json:"hb_button_battery"`
	HB_RandomNonce   string `json:"hb_button_random"`
	HB_ButtonMode    string `json:"hb_button_mode"`
}

type Button struct {
	Name            string        `json:"name"`
	Button_id       string        `json:"button_id"`
	Button_type     string        `json:"button_type"`
	Button_location string        `json:"button_location"`
	Incoming_JSON   Incoming_json `json:"incoming_json"`
	Distance        float64       `json:"distance"`
	Last_seen       int64         `json:"last_seen"`

	HB_ButtonCounter int64  `json:"hb_button_counter"`
	HB_Battery       int64  `json:"hb_button_battery"`
	HB_RandomNonce   string `json:"hb_button_random"`
	HB_ButtonMode    string `json:"hb_button_mode"`
}

type Beacons_list struct {
	Beacons map[string]Beacon `json:"beacons"`
	lock    sync.RWMutex
}

type Locations_list struct {
	locations map[string]Location
	lock      sync.RWMutex
}
