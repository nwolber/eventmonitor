// Copyright (c) 2016 Niklas Wolber
// This file is licensed under the MIT license.
// See the LICENSE file for more information.

package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
)

func main() {
	var err error
	host, influxHost, username, password, db, measurement, authLog := config()

	if host == "" {
		host, err = os.Hostname()
		if err != nil {
			log.Fatalln("Hostname not provided and failed to get hostname from system", err)
		}
		log.Println("Using hostname", host)
	}

	client, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:      influxHost,
		Username:  username,
		Password:  password,
		UserAgent: "loginmonitor",
	})
	if err != nil {
		log.Fatalln("Failed to create InfluxDB client", err)
	}

	c := make(chan struct{})

	go func() {
		if _, _, err = client.Ping(time.Second); err != nil {
			log.Fatalln("Error connecting to InfluxDB", err)
		}
		c <- struct{}{}
	}()

	select {
	case <-c:
	case <-time.Tick(5 * time.Second):
		log.Fatalln("InfluxDB failed to respond in time")
	}

	log.Println("Connected to InfluxDB at", influxHost)

	s := &eventStore{
		c:           client,
		hostname:    host,
		db:          db,
		measurement: measurement,
	}

	closer := make(chan struct{})

	go func() {
		monitorAuthLog(authLog, s)
		closer <- struct{}{}
	}()

	go func() {
		monitorDocker(s)
		closer <- struct{}{}
	}()

	<-closer
}

func config() (host, influx, username, password, db, measurement, authLog string) {
	const (
		defaultHost        = ""
		defaultInfluxDb    = "http://localhost:8086"
		defaultUsername    = ""
		defaultPassword    = ""
		defaultDb          = ""
		defaultMeasurement = "events"
		defaultAuthLog     = "/var/log/auth.log"
	)

	flag.StringVar(&host, "host", defaultHost, "String to use in the 'hostname' tag, if empty the system will be queried")
	flag.StringVar(&influx, "influxdb", defaultInfluxDb, "InfluxDB HTTP endpoint")
	flag.StringVar(&username, "username", defaultUsername, "Username for InfluxDB")
	flag.StringVar(&password, "password", defaultPassword, "Password for InfluxDB")
	flag.StringVar(&db, "db", defaultDb, "Database where events are written to")
	flag.StringVar(&measurement, "measurement", defaultMeasurement, "Measurement where events are written to")
	flag.StringVar(&authLog, "authlog", defaultAuthLog, "The PAM authentication log to watch for login/logout messages")

	help := flag.Bool("help", false, "Print this help message")
	c := flag.Bool("config", false, "Print config")

	flag.Parse()

	if *help {
		flag.PrintDefaults()
		os.Exit(1)
		// Not reached
	}

	if *c {
		log.Println("host        =", host)
		log.Println("influxdb    =", influx)
		log.Println("username    =", username)
		log.Println("password    =", password)
		log.Println("db          =", db)
		log.Println("measurement =", measurement)
		log.Println("authLog     =", authLog)
	}

	return
}

type eventStore struct {
	c                         influx.Client
	hostname, db, measurement string
}

func (s *eventStore) storeUserEvent(typ, user, msg string, timestamp time.Time) {
	s.store("auth", typ, msg, map[string]string{
		"user": user,
	}, timestamp)
}

func (s *eventStore) storeDockerEvent(typ, service, container, image, msg string, timestamp time.Time) {
	tags := make(map[string]string)

	tags["container"] = container
	tags["image"] = image

	if service != "" {
		tags["service"] = service
	}

	s.store("docker", typ, msg, tags, timestamp)
}

func (s *eventStore) store(provider, event, msg string, tags map[string]string, timestamp time.Time) {
	t := make(map[string]string)
	t["hostname"] = s.hostname
	t["event"] = event

	for k, v := range tags {
		if _, ok := t[k]; ok {
			log.Println(k, "is a reserved tag. Won't store", event, "event.")
			return
		}

		t[k] = v
	}

	fields := make(map[string]interface{})
	fields["description"] = msg

	measurement := provider + strings.Title(s.measurement)
	p, err := influx.NewPoint(measurement, t, fields, timestamp)
	if err != nil {
		log.Println("Error creating event", msg, err)
		return
	}

	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database: s.db,
	})
	if err != nil {
		log.Println("Error creating batch points", err)
		return
	}

	bp.AddPoint(p)

	if err := s.c.Write(bp); err != nil {
		log.Println("Error writing batch point", err)
		return
	}
	log.Println("Message written")
}
