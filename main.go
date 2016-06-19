// Copyright (c) 2016 Niklas Wolber
// This file is licensed under the MIT license.
// See the LICENSE file for more information.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hpcloud/tail"
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

	t, err := tail.TailFile(authLog, tail.Config{
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: os.SEEK_END,
		},
		Follow: true,
		ReOpen: true,
	})

	if err != nil {
		log.Fatalln("Error opening", authLog, err)
	}
	log.Println("Tailing", authLog)

	const (
		loginFinterprint  = "session opened for user"
		logoutFingerprint = "session closed for user"
	)

	for line := range t.Lines {
		if strings.Contains(line.Text, loginFinterprint) {
			index := strings.Index(line.Text, loginFinterprint)
			text := line.Text[index:]

			parts := strings.Split(text, " ")
			if len(parts) < 4 {
				log.Println("Unexpected number of parts", line.Text)
				continue
			}

			user := parts[4]
			msg := fmt.Sprint("User ", user, " logged in")
			log.Println(msg)

			s.store("login", user, msg, line.Time)
			continue
		}

		if strings.Contains(line.Text, logoutFingerprint) {
			index := strings.Index(line.Text, logoutFingerprint)
			text := line.Text[index:]

			parts := strings.Split(text, " ")
			if len(parts) < 4 {
				log.Println("Unexpected number of parts", parts)
				continue
			}

			user := parts[4]
			msg := fmt.Sprint("User ", user, " logged out")
			log.Println(msg)

			s.store("logout", user, msg, line.Time)
		}
	}
	if err = t.Err(); err != nil {
		log.Println(err)
	}
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

func (s *eventStore) store(typ, user, msg string, t time.Time) {
	tags := make(map[string]string)
	tags["hostname"] = s.hostname
	tags["type"] = typ
	tags["user"] = user
	fields := make(map[string]interface{})
	fields["description"] = msg

	p, err := influx.NewPoint(s.measurement, tags, fields, t)
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
