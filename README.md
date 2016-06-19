#Loginmonitor
Loginmonitor monitors a Linux authentication log for PAM messages indicating user login and logout.
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
Login and logout events are written to the measurement given by the **-measurements** parameter (by default 'events'). There are two tags and one field.

- Tag **type**: The type of event, either *login* or *logout*.
- Tag **user**: The name of the user that caused the event.
- Field **description**: A textual description of what happend.

##Grafana Annotations
### Annotate all logins
```
SELECT description FROM events WHERE type='logout' AND $timeFilter
```
###Annotate all *root* logins and logouts
```
SELECT description FROM events WHERE "user"='root' AND $timeFilter
```

##License
MIT. See LICENSE file.