package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"
)

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	beaconPeriod = 2 * time.Second
)

// GLOBALS

var BEACONS Beacons_list

var Buttons_list map[string]Button

var cli *client.Client

var http_results HTTP_locations_list
var http_results_lock sync.RWMutex

var Latest_beacons_list map[string]Beacon
var latest_list_lock sync.RWMutex

var db *bolt.DB
var err error

var world = []byte("presence")

var settings = Settings{
	Location_confidence:    8,
	Last_seen_threshold:    45,
	Beacon_metrics_size:    30,
	HA_send_interval:       5,
	HA_send_changes_only:   false,
	RSSI_enforce_threshold: false,
	RSSI_min_threshold:     -120,
}

func getBeaconID(incoming Incoming_json) string {
	unique_id := fmt.Sprintf("%s", incoming.MAC)
	if incoming.Beacon_type == "ibeacon" {
		unique_id = fmt.Sprintf("%s_%s_%s", incoming.UUID, incoming.Major, incoming.Minor)
	} else if incoming.Beacon_type == "eddystone" {
		unique_id = fmt.Sprintf("%s_%s", incoming.Namespace, incoming.Instance_id)
	} else if incoming.Beacon_type == "hb_button" {
		unique_id = fmt.Sprintf("%s_%s", incoming.Namespace, incoming.Instance_id)
	}
	return unique_id
}

func incomingBeaconFilter(incoming Incoming_json) Incoming_json {
	out_json := incoming
	if incoming.Beacon_type != "ibeacon" && incoming.Beacon_type != "eddystone" && incoming.Beacon_type != "hb_button" {
		//do additional checks here to detect if a Habby Bubbles Button
		// looks like 020104020a0011ff045600012d3859db59e1000b9453

		raw_data := incoming.Data
		company_id := []byte{0x04, 0x56}
		product_id := []byte{0x00, 0x01}
		hb_button_prefix_str := fmt.Sprintf("020104020a0011ff%02x%02x%02x%02x", company_id[0], company_id[1], product_id[0], product_id[1])
		if strings.HasPrefix(raw_data, hb_button_prefix_str) {
			out_json.Namespace = "ddddeeeeeeffff5544ff"
			out_json.Instance_id = raw_data[24:36]

			//TODO: do MUCH better error handling here

			counter_str := fmt.Sprintf("0x%s", raw_data[36:38])
			counter, _ := strconv.ParseInt(counter_str, 0, 64)
			out_json.HB_ButtonCounter = counter

			battery_str := fmt.Sprintf("0x%s", raw_data[38:42])
			battery, _ := strconv.ParseInt(battery_str, 0, 64)
			out_json.HB_Battery = battery

			out_json.HB_RandomNonce = raw_data[42:44]

			mode := raw_data[44:46]
			if mode == "00" {
				out_json.HB_ButtonMode = "presence_button"
			} else {
				out_json.HB_ButtonMode = "button_only"
			}
			out_json.TX_power = fmt.Sprintf("0x%s", raw_data[46:48])

			out_json.Beacon_type = "hb_button"

			debugf("Button adv has %#v\n", out_json)
		}
	} //else if incoming.Beacon_type == "eddystone" && incoming.Namespace == "ddddeeeeeeffff5544ff" {
	//out_json.Beacon_type = "hb_button"
	//}
	return out_json
}

func processButton(bbeacon Beacon, cl *client.Client) {
	btn := Button{Name: bbeacon.Name}
	btn.Button_id = bbeacon.Beacon_id
	btn.Button_type = bbeacon.Beacon_type
	btn.Button_location = bbeacon.Previous_location
	btn.Incoming_JSON = bbeacon.Incoming_JSON
	btn.Distance = bbeacon.Distance
	btn.Last_seen = bbeacon.Last_seen
	btn.HB_ButtonCounter = bbeacon.HB_ButtonCounter
	btn.HB_Battery = bbeacon.HB_Battery
	btn.HB_RandomNonce = bbeacon.HB_RandomNonce
	btn.HB_ButtonMode = bbeacon.HB_ButtonMode

	nonce, ok := Buttons_list[btn.Button_id]
	if !ok || nonce.HB_RandomNonce != btn.HB_RandomNonce {
		// send the button message to MQTT
		sendButtonMessage(btn, cl)
	}
	Buttons_list[btn.Button_id] = btn
}

func getiBeaconDistance(rssi int64, power string) float64 {
	ratio := float64(rssi) * (1.0 / float64(twos_comp(power)))
	distance := 100.0
	if ratio < 1.0 {
		distance = math.Pow(ratio, 10)
	} else {
		distance = (0.89976)*math.Pow(ratio, 7.7095) + 0.111
	}
	return distance
}

func getBeaconDistance(incoming Incoming_json) float64 {
	distance := 1000.0
	if incoming.Beacon_type == "ibeacon" {
		distance = getiBeaconDistance(incoming.RSSI, incoming.TX_power)
	} else if incoming.Beacon_type == "eddystone" {
		//TODO: fix this, probably not the way to do this calc with eddystone
		distance = getiBeaconDistance(incoming.RSSI, incoming.TX_power)
	} else if incoming.Beacon_type == "hb_button" {
		//TODO: fix this, probably not the way to do this calc with eddystone
		distance = getiBeaconDistance(incoming.RSSI, incoming.TX_power)
	} else {
		//return the absolute value of RSSI. this is fine since should always be below 0 and the closer to 0 the better, so smaller is better just like ibeacon distance
		distance = math.Abs(float64(incoming.RSSI))
	}
	return distance
}

func getAverageDistance(beacon_metrics []beacon_metric) float64 {
	total := 0.0

	for _, v := range beacon_metrics {
		total += v.distance
	}
	return (total / float64(len(beacon_metrics)))
}

func sendHARoomMessage(beacon_id string, beacon_name string, distance float64, location string, cl *client.Client) {
	//first make the json
	ha_msg, err := json.Marshal(HA_message{Beacon_id: beacon_id, Beacon_name: beacon_name, Distance: distance})
	if err != nil {
		panic(err)
	}

	//send the message to HA
	err = cl.Publish(&client.PublishOptions{
		QoS:       mqtt.QoS1,
		TopicName: []byte("happy-bubbles/presence/ha/" + location),
		Message:   ha_msg,
	})
	if err != nil {
		panic(err)
	}
}

func sendButtonMessage(btn Button, cl *client.Client) {
	//first make the json
	btn_msg, err := json.Marshal(btn)
	if err != nil {
		panic(err)
	}

	//send the message to HA
	err = cl.Publish(&client.PublishOptions{
		QoS:       mqtt.QoS1,
		TopicName: []byte("happy-bubbles/presence/button/" + btn.Button_id),
		Message:   btn_msg,
	})
	if err != nil {
		panic(err)
	}
}

func getLikelyLocations(settings Settings, locations_list Locations_list, cl *client.Client) {
	// create the http results structure
	http_results_lock.Lock()
	http_results = HTTP_locations_list{}
	http_results.Beacons = make([]HTTP_location, 0)
	http_results.Buttons = make([]Button, 0)
	http_results_lock.Unlock()

	should_persist := false

	// iterate through the beacons we want to search for
	for _, beacon := range BEACONS.Beacons {

		r := HTTP_location{}
		r.Distance = 999
		r.Name = beacon.Name
		r.Beacon_name = beacon.Name
		r.Beacon_id = beacon.Beacon_id
		r.Beacon_type = beacon.Beacon_type
		r.HB_Battery = beacon.HB_Battery
		r.HB_ButtonMode = beacon.HB_ButtonMode
		r.Location = ""
		r.Last_seen = 999
		r.Beacon_Enabled = beacon.Beacon_Enabled

		if len(beacon.beacon_metrics) == 0 {
			http_results_lock.Lock()
			http_results.Beacons = append(http_results.Beacons, r)
			http_results_lock.Unlock()
			continue
		}

		if (int64(time.Now().Unix()) - (beacon.beacon_metrics[len(beacon.beacon_metrics)-1].timestamp)) > settings.Last_seen_threshold {
			http_results_lock.Lock()
			http_results.Beacons = append(http_results.Beacons, r)
			http_results_lock.Unlock()
		}

		best_location := Best_location{}

		// go through its beacon metrics and pick out the location that appears most often
		loc_list := make(map[string]float64)
		seen_weight := 1.5
		rssi_weight := 0.75
		for _, metric := range beacon.beacon_metrics {
			loc, ok := loc_list[metric.location]
			if !ok {
				loc = seen_weight + (rssi_weight * (1.0 - (float64(metric.rssi) / -100.0)))
			} else {
				loc = loc + seen_weight + (rssi_weight * (1.0 - (float64(metric.rssi) / -100.0)))
			}
			loc_list[metric.location] = loc
		}
		debugf("beacon: %s list: %#v\n", beacon.Name, loc_list)
		// now go through the list and find the largest, that's the location
		best_name := ""
		ts := 0.0
		for name, times_seen := range loc_list {
			if times_seen > ts {
				best_name = name
				ts = times_seen
			}
		}
		debugf("BEST LOCATION FOR %s IS: %s with score: %f\n", beacon.Name, best_name, ts)
		best_location = Best_location{name: best_name, distance: beacon.beacon_metrics[len(beacon.beacon_metrics)-1].distance, last_seen: beacon.beacon_metrics[len(beacon.beacon_metrics)-1].timestamp}

		//filter, only let this location become best if it was X times in a row
		if best_location.name == beacon.Previous_location {
			beacon.Location_confidence = beacon.Location_confidence + 1
		} else {
			beacon.Location_confidence = 0
		}

		//create an http result from this
		r = HTTP_location{}
		r.Distance = best_location.distance
		r.Name = beacon.Name
		r.Beacon_name = beacon.Name
		r.Beacon_id = beacon.Beacon_id
		r.Beacon_type = beacon.Beacon_type
		r.HB_Battery = beacon.HB_Battery
		r.HB_ButtonMode = beacon.HB_ButtonMode
		r.Location = best_location.name
		r.Last_seen = best_location.last_seen

		if beacon.Location_confidence == settings.Location_confidence && beacon.Previous_confident_location != best_location.name {
			// location has changed, send an mqtt message

			should_persist = true
			debugf("detected a change!!! %#v\n\n", beacon)

			beacon.Location_confidence = 0

			//first make the json
			js, err := json.Marshal(Location_change{Beacon_ref: beacon, Name: beacon.Name, Beacon_name: beacon.Name, Previous_location: beacon.Previous_confident_location, New_location: best_location.name, Timestamp: time.Now().Unix()})
			if err != nil {
				continue
			}

			//send the message
			err = cl.Publish(&client.PublishOptions{
				QoS:       mqtt.QoS1,
				TopicName: []byte("happy-bubbles/presence/changes"),
				Message:   js,
			})
			if err != nil {
				panic(err)
			}

			if settings.HA_send_changes_only && beacon.Beacon_Enabled {
				sendHARoomMessage(beacon.Beacon_id, beacon.Name, best_location.distance, best_location.name, cl)
			}

			beacon.Previous_confident_location = best_location.name

		}

		beacon.Previous_location = best_location.name

		BEACONS.Beacons[beacon.Beacon_id] = beacon

		http_results_lock.Lock()
		http_results.Beacons = append(http_results.Beacons, r)
		http_results_lock.Unlock()

		if best_location.name != "" {
			if !settings.HA_send_changes_only {
				secs := int64(time.Now().Unix())
				if secs%settings.HA_send_interval == 0 {
					sendHARoomMessage(beacon.Beacon_id, beacon.Name, best_location.distance, best_location.name, cl)
				}
			}
		}

		debugf("\n\n%s is most likely in %s with average distance %f \n\n", beacon.Name, best_location.name, best_location.distance)
		// publish this to a topic
		// Publish a message.
		err := cl.Publish(&client.PublishOptions{
			QoS:       mqtt.QoS0,
			TopicName: []byte("happy-bubbles/presence"),
			Message:   []byte(fmt.Sprintf("%s is most likely in %s with average distance %f", beacon.Name, best_location.name, best_location.distance)),
		})
		if err != nil {
			panic(err)
		}
	}

	for _, button := range Buttons_list {
		http_results.Buttons = append(http_results.Buttons, button)
	}

	if should_persist {
		persistBeacons()
	}
}

func IncomingMQTTProcessor(updateInterval time.Duration, cl *client.Client, db *bolt.DB) chan<- Incoming_json {

	incoming_msgs_chan := make(chan Incoming_json, 10)

	// load initial BEACONS
	BEACONS.Beacons = make(map[string]Beacon)
	// retrieve the data

	checkOrCreateDb()

	loadDataToGlobalVariables()

	Latest_beacons_list = make(map[string]Beacon)

	Buttons_list = make(map[string]Button)

	//create a map of locations, looked up by hostnames
	locations_list := Locations_list{}
	ls := make(map[string]Location)
	locations_list.locations = ls

	ticker := time.NewTicker(updateInterval)

	go func() {
		for {
			select {

			case <-ticker.C:
				getLikelyLocations(settings, locations_list, cl)
			case incoming := <-incoming_msgs_chan:
				func() {
					defer func() {
						if err := recover(); err != nil {
							debugf("work failed: %s", err)
						}
					}()

					incoming = incomingBeaconFilter(incoming)
					this_beacon_id := getBeaconID(incoming)

					now := time.Now().Unix()

					debug("saw " + this_beacon_id + " at " + incoming.Hostname)

					//if this beacon isn't in our search list, add it to the latest_beacons pile.
					beacon, ok := BEACONS.Beacons[this_beacon_id]
					if !ok {
						//should be unique
						//if it's already in list, forget it.
						latest_list_lock.Lock()
						x, ok := Latest_beacons_list[this_beacon_id]
						if ok {
							//update its timestamp
							x.Last_seen = now
							x.Incoming_JSON = incoming
							x.Distance = getBeaconDistance(incoming)
							x.Name = incoming.Name

							Latest_beacons_list[this_beacon_id] = x
						} else {
							Latest_beacons_list[this_beacon_id] = Beacon{Beacon_id: this_beacon_id, Beacon_type: incoming.Beacon_type, Last_seen: now, Incoming_JSON: incoming, Beacon_location: incoming.Hostname, Distance: getBeaconDistance(incoming), Name: incoming.Name}
						}
						for k, v := range Latest_beacons_list {
							if (now - v.Last_seen) > 10 { // 10 seconds
								delete(Latest_beacons_list, k)
							}
						}
						latest_list_lock.Unlock()
						//continue
						return
					}

					// ignore this beacon if it falls below RSSI setting
					// threshold

					if settings.RSSI_enforce_threshold && (int64(incoming.RSSI) < settings.RSSI_min_threshold) {
						//debugf("rejecting rssi incoming %d < %d\n", int64(incoming.RSSI), settings.RSSI_min_threshold)
						return
					}

					beacon.Incoming_JSON = incoming
					beacon.Last_seen = now
					beacon.Beacon_type = incoming.Beacon_type
					beacon.HB_ButtonCounter = incoming.HB_ButtonCounter
					beacon.HB_Battery = incoming.HB_Battery
					beacon.HB_RandomNonce = incoming.HB_RandomNonce
					beacon.HB_ButtonMode = incoming.HB_ButtonMode

					if beacon.beacon_metrics == nil {
						beacon.beacon_metrics = make([]beacon_metric, settings.Beacon_metrics_size)
					}
					//create metric for this beacon
					this_metric := beacon_metric{}
					this_metric.distance = getBeaconDistance(incoming)
					this_metric.timestamp = now
					this_metric.rssi = int64(incoming.RSSI)
					this_metric.location = incoming.Hostname
					beacon.beacon_metrics = append(beacon.beacon_metrics, this_metric)
					//debugf("APPENDING a metric from %s len %d\n", beacon.Name, len(beacon.beacon_metrics))
					if len(beacon.beacon_metrics) > settings.Beacon_metrics_size {
						//debugf("deleting a metric from %s len %d\n", beacon.Name, len(beacon.beacon_metrics))
						beacon.beacon_metrics = append(beacon.beacon_metrics[:0], beacon.beacon_metrics[0+1:]...)
					}
					//debugf("%#v\n", beacon.beacon_metrics)

					BEACONS.Beacons[beacon.Beacon_id] = beacon

					if beacon.Beacon_type == "hb_button" {
						processButton(beacon, cl)
					}

					//lookup location by hostname in locations
					location, ok := locations_list.locations[incoming.Hostname]
					if !ok {
						//create the location
						locations_list.locations[incoming.Hostname] = Location{}
						location, ok = locations_list.locations[incoming.Hostname]
						location.name = incoming.Hostname
					}
					locations_list.locations[incoming.Hostname] = location
				}()
			}
		}
	}()

	return incoming_msgs_chan
}

var http_host_path_ptr *string

func main() {
	http_host_path_ptr = flag.String("http_host_path", "localhost:5555", "The host:port that the HTTP server should listen on")

	mqtt_host_ptr := flag.String("mqtt_host", "localhost:1883", "The host:port of the MQTT server to listen for Happy Bubbles beacons on")
	mqtt_username_ptr := flag.String("mqtt_username", "none", "The username needed to connect to the MQTT server, 'none' if it doesn't need one")
	mqtt_password_ptr := flag.String("mqtt_password", "none", "The password needed to connect to the MQTT server, 'none' if it doesn't need one")
	mqtt_client_id_ptr := flag.String("mqtt_client_id", "happy-bubbles-presence-detector", "The client ID for the MQTT server")
	db_file_ptr := flag.String("db_file", "presence.db", "The location of the database file")

	flag.Parse()

	// Set up channel on which to send signal notifications.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	// Create an MQTT Client.
	cli := client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})
	// Terminate the Client.
	defer cli.Terminate()

	//open the database
	db, err = bolt.Open(*db_file_ptr, 0644, nil)
	if err != nil {
		logrus.Fatal(err)
	}
	defer db.Close()

	// Connect to the MQTT Server.
	err = cli.Connect(&client.ConnectOptions{
		Network:  "tcp",
		Address:  *mqtt_host_ptr,
		ClientID: []byte(*mqtt_client_id_ptr),
		UserName: []byte(*mqtt_username_ptr),
		Password: []byte(*mqtt_password_ptr),
	})
	if err != nil {
		panic(err)
	}

	incoming_updates_chan := IncomingMQTTProcessor(1*time.Second, cli, db)

	// Subscribe to topics.
	err = cli.Subscribe(&client.SubscribeOptions{
		SubReqs: []*client.SubReq{
			&client.SubReq{
				//TopicFilter: []byte("happy-bubbles/ble/+/ibeacon/+"),
				TopicFilter: []byte("happy-bubbles/ble/#"),
				QoS:         mqtt.QoS0,
				// Define the processing of the message handler.
				Handler: func(topicName, message []byte) {
					incoming := Incoming_json{}
					json.Unmarshal(message, &incoming)

					//pass this to the state monitor
					incoming_updates_chan <- incoming
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(" _   _    _    ____  ______   __  ____  _   _ ____  ____  _     _____ ____\n| | | |  / \\  |  _ \\|  _ \\ \\ / / | __ )| | | | __ )| __ )| |   | ____/ ___|\n| |_| | / _ \\ | |_) | |_) \\ V /  |  _ \\| | | |  _ \\|  _ \\| |   |  _| \\___ \\\n|  _  |/ ___ \\|  __/|  __/ | |   | |_) | |_| | |_) | |_) | |___| |___ ___) |\n|_| |_/_/   \\_\\_|   |_|    |_|   |____/ \\___/|____/|____/|_____|_____|____/")
	fmt.Println("\n ")
	fmt.Println("CONNECTED TO MQTT")
	fmt.Println("\n ")
	fmt.Println("Visit http://" + *http_host_path_ptr + " on your browser to see the web interface")
	fmt.Println("\n ")

	go startServer()

	// Wait for receiving a signal.
	<-sigc

	// Disconnect the Network Connection.
	if err := cli.Disconnect(); err != nil {
		panic(err)
	}
}
