// Copyright (c) 2016 Niklas Wolber
// This file is licensed under the MIT license.
// See the LICENSE file for more information.

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hpcloud/tail"
)

func monitorAuthLog(authLog string, s *eventStore) {
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

			s.storeUserEvent("login", user, msg, line.Time)
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

			s.storeUserEvent("logout", user, msg, line.Time)
		}
	}
	if err = t.Err(); err != nil {
		log.Println(err)
	}
}
