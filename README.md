#Eventmonitor
Eventmonitor monitors diffrent event sources.
These events are written to an [InfluxDB](https://influxdata.com/time-series-platform/influxdb/) and can be used by [Grafana](http://grafana.org/) for [annotations](http://docs.grafana.org/reference/annotations/#influxdb-annotations).

##Installation
```
go get -u github.com/nwolber/loginmonitor
```

##Usage
```
loginmonitor -help
  -authlog string
        The PAM authentication log to watch for login/logout messages (default "/var/log/auth.log")
  -config
        Print config
  -db string
        Database where events are written to
  -help
        Print this help message
  -host string 
        String to use in the 'hostname' tag, if empty the system will be queried
  -influxdb string
        InfluxDB HTTP endpoint (default "http://localhost:8086")
  -measurement string
        Measurement where events are written to (default "events")
  -password string
        Password for InfluxDB
  -username string
        Username for InfluxDB
```

##InfluxDB schema
Events are written to a measurement per provider. Currently there are two providers, **auth** and **docker**.

Shared tags:
- **hostname**: The host the event occured on. This is either provided through the *-host* command line flag or automatically retrieved from the operation system.
- **event**: The type of event.

Shared fields:
- **description**: A textual description of what happend.

###Auth log
The Linux authentication log is monitored for PAM user login and logout messages. Authentication events are stored in the `authEvents` measurement. **Event** values therefore are `login` and `logout`.

Additional tags:
- **user**: The name of the user that caused the event.

###Docker
Docker is monitored for events inicating the start and stop of a container. **Event** values are `containerStart` and `containerStop`.

Additional tags:
- **container**: The name of the container that caused the event.
- **image**: The image the container was running.
- **service**: The Docker Compose service the container belonged to, if available.

##Grafana Annotations
###Add annotations for logouts, regardless of the user that logged out.
```
SELECT description FROM events WHERE type='logout' AND $timeFilter
```
###Add annotations for logins and logouts of the *root* user.
```
SELECT description FROM events WHERE "user"='root' AND $timeFilter
```

##License
MIT. See LICENSE file.