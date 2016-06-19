// Copyright (c) 2016 Niklas Wolber
// This file is licensed under the MIT license.
// See the LICENSE file for more information.

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fsouza/go-dockerclient"
)

func monitorDocker(s *eventStore) {
	host, err := docker.DefaultDockerHost()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Docker endpoint:", host)

	client, err := docker.NewClient(host)
	if err != nil {
		log.Fatalln(err)
	}
	client.SkipServerVersionCheck = true

	events := make(chan *docker.APIEvents)
	err = client.AddEventListener(events)
	if err != nil {
		log.Fatalln(err)
	}

	for apiEvent := range events {
		if apiEvent.Type != "container" {
			log.Println("Received", apiEvent.Type, "event. Skipping.")
			continue
		}

		service, container, image := getDockerAttributes(apiEvent)
		var (
			typ, msg string
		)

		if apiEvent.Action == "start" {
			typ = "containerStart"
			msg = fmt.Sprint("Container ", container, " started.")
		} else if apiEvent.Action == "die" {
			typ = "containerDie"
			msg = fmt.Sprint("Container ", container, " died.")
		} else {
			continue
		}
		log.Println(msg)

		s.storeDockerEvent(typ, service, container, image, msg, time.Unix(0, apiEvent.TimeNano))
	}
}

func getDockerAttributes(event *docker.APIEvents) (service, container, image string) {
	const (
		serviceKey   = "com.docker.compose.service"
		containerKey = "name"
		imageKey     = "image"
	)

	service = event.Actor.Attributes[serviceKey]
	container = event.Actor.Attributes[containerKey]
	image = event.Actor.Attributes[imageKey]
	return
}
