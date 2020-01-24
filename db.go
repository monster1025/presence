package main

import (
	"bytes"
	"encoding/gob"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
)

func checkOrCreateDb() {
	// create bucket if not exist
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(world)
		if err != nil {
			return err
		}
		return nil
	})
}

func loadDataToGlobalVariables() {
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(world)
		if bucket == nil {
			return err
		}

		key := []byte("beacons_list")
		val := bucket.Get(key)
		if val != nil {
			buf := bytes.NewBuffer(val)
			dec := gob.NewDecoder(buf)
			err = dec.Decode(&BEACONS)
			if err != nil {
				logrus.Fatal("decode error:", err)
			}
		}

		key = []byte("buttons_list")
		val = bucket.Get(key)
		if val != nil {
			buf := bytes.NewBuffer(val)
			dec := gob.NewDecoder(buf)
			err = dec.Decode(&Buttons_list)
			if err != nil {
				logrus.Fatal("decode error:", err)
			}
		}

		key = []byte("settings")
		val = bucket.Get(key)
		if val != nil {
			buf := bytes.NewBuffer(val)
			dec := gob.NewDecoder(buf)
			err = dec.Decode(&settings)
			if err != nil {
				logrus.Fatal("decode error:", err)
			}
		}

		return nil
	})
	if err != nil {
		logrus.Fatal(err)
	}

	//debug list them out
	debug("Database beacons:")
	for _, beacon := range BEACONS.Beacons {
		debug("Database has known beacon: " + beacon.Beacon_id + " " + beacon.Name + "\n")
	}
	debugf("Settings has %#v\n", settings)
}

func persistBeacons() error {
	// gob it first
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(BEACONS); err != nil {
		return err
	}

	key := []byte("beacons_list")
	// store some data
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(world)
		if err != nil {
			return err
		}

		err = bucket.Put(key, []byte(buf.String()))
		if err != nil {
			return err
		}
		return nil
	})
	return nil
}

func persistButtons() error {
	// gob it first
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(Buttons_list); err != nil {
		return err
	}

	key := []byte("buttons_list")
	// store some data
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(world)
		if err != nil {
			return err
		}

		err = bucket.Put(key, []byte(buf.String()))
		if err != nil {
			return err
		}
		return nil
	})
	return nil
}

func persistSettings() error {
	// gob it first
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(settings); err != nil {
		return err
	}

	key := []byte("settings")
	// store some data
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(world)
		if err != nil {
			return err
		}

		err = bucket.Put(key, []byte(buf.String()))
		if err != nil {
			return err
		}
		return nil
	})
	return nil
}
