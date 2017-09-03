package main

import (
  "fmt"
  "time"
  "errors"
  "strings"
  "encoding/json"
  "io/ioutil"
  "os"
)

type Alarm struct {
  Id      string
  Name    string
  When    time.Time
  Effect  string
  disabled bool   // set to true when we're checking alarms and it fired
  countdown bool  // set to true when we're checking alarms and we signaled countdown
}

type LoaderMsg struct {
  msg   string
  alarm Alarm
  val   interface{}
}

func handledMessage(alm Alarm) LoaderMsg {
  return LoaderMsg{msg:"handled", alarm: alm}
}

func reloadMessage() LoaderMsg {
  return LoaderMsg{msg:"reload"}
}

func writeAlarms(alarms []Alarm, fname string) error {
  output, err := json.Marshal(alarms)
  logMessage(string(output))
  if err != nil {
    return err
  }
  return ioutil.WriteFile(fname, output, 0644)
}

func handledAlarm(alarm Alarm, handled map[string]Alarm) bool {
  // if the time changed, consider it "unhandled"
  v, ok := handled[alarm.Id]
  if !ok { return false}
  if v.When != alarm.When { return false }
  // everything else ignore
  return true
}

func cacheFilename(settings *Settings) string {
  return settings.GetString("alarmPath") + "/alarm.json"
}

func getAlarmsFromService(settings *Settings, handled map[string]Alarm) ([]Alarm, error) {
  alarms := make([]Alarm, 0)
  srv := GetCalenderService(settings)

  // TODO: if it wasn't available, send an Alarm message
  if srv == nil {
    return alarms, errors.New("Failed to get calendar service")
  }

  // map the calendar to an ID
  calName := settings.GetString("calendar")
  var id string
  {
    logMessage("get calendar list")
    list, err := srv.CalendarList.List().Do()
    if err != nil {
      logMessage(err.Error())
      return alarms, err
    }
    for _, i := range list.Items {
      if i.Summary == calName {
        id = i.Id
        break
      }
    }
  }

  if id == "" {
    return alarms, errors.New(fmt.Sprintf("Could not find calendar %s", calName))
  }
  // get next 10 (?) alarms
  t := time.Now().Format(time.RFC3339)
  events, err := srv.Events.List(id).
                    ShowDeleted(false).
                    SingleEvents(true).
                    TimeMin(t).
                    MaxResults(10).
                    OrderBy("startTime").
                    Do()
  if err != nil {
    return alarms, err
  }

  // remove the cached alarms if they are present
  cacheFile := cacheFilename(settings)
  if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
    err = os.Remove(cacheFile)
    // an error here is a system config issue
    if err != nil {
      // TODO: severe error effect
      logMessage(fmt.Sprintf("Error: %s", err.Error()))
      return alarms, err
    }
  }

  // calculate the alarms, write to a file
  if len(events.Items) > 0 {
    for _, i := range events.Items {
      // If the DateTime is an empty string the Event is an all-day Event.
      // So only Date is available.
      if i.Start.DateTime == "" {
        logMessage(fmt.Sprintf("Not a time based alarm: %s @ %s", i.Summary, i.Start.Date))
        continue
      }
      var when time.Time
      when, err = time.Parse(time.RFC3339, i.Start.DateTime)
      if err != nil {
        // skip bad formats
        logMessage(err.Error())
        continue
      }

      if when.Sub(time.Now()) < 0 {
        logMessage(fmt.Sprintf("Skipping old alarm: %s", i.Id))
        continue
      }

      alm := Alarm{Id: i.Id, Name: i.Summary, When: when, disabled: false}

      // look for hastags (does not work ATM, the gAPI is broken I think)
      music := strings.Contains(i.Summary, "#music")
      random := strings.Contains(i.Summary, "#random")
      file := strings.Contains(i.Summary, "#file")
      tones := strings.Contains(i.Summary, "#tone") // tone or tones

      // priority is arbitrary except for random (default)
      if music {
        alm.Effect = "music"
      } else if file {
        alm.Effect = "file" // TODO: figure out the filename
      } else if tones {
        alm.Effect = "tones" // TODO: tone options
      } else if random {
        alm.Effect = "random"
      } else {
        alm.Effect = "random"
      }

      // has this one been handled?
      if handledAlarm(alm, handled) {
        logMessage(fmt.Sprintf("Skipping handled alarm: %s", alm.Id))
      }

      alarms = append(alarms, alm)
    }

    // cache in a file for later if we go offline
    writeAlarms(alarms, cacheFile)
  }

  return alarms, nil
}

func getAlarmsFromCache(settings *Settings, handled map[string]Alarm) ([]Alarm, error) {
  alarms := make([]Alarm, 0)
  data, err := ioutil.ReadFile(cacheFilename(settings))
  if err != nil {
    return alarms, err
  }
  err = json.Unmarshal(data, &alarms)
  if err != nil {
    return alarms, err
  }
  // remove any that are in the "handled" map
  for i:=len(alarms)-1;i>=0;i-- {
    if handledAlarm(alarms[i], handled) {
      // remove is append two slices without the part we don't want
      logMessage(fmt.Sprintf("Discard handled alarm: %s", alarms[i].Id))
      alarms = append(alarms[:i], alarms[i+1:]...)
    }
  }

  return alarms, nil
}

func getAlarms(settings *Settings, cA chan Alarm, cE chan Effect, cL chan LoaderMsg) {
  defer wg.Done()

  // keep a list of things that we have done
  // TODO: GC the list occassionally
  handledAlarms := map[string]Alarm{}

  var lastRefresh time.Time

  for true {
    // read any messages alarms first
    keepReading := true;
    reload := false
    if time.Now().Sub(lastRefresh) > settings.GetDuration("alarmRefreshTime") {
      reload = true
    }

    for keepReading {
      select {
        case msg := <- cL:
          switch (msg.msg) {
          case "handled":
            handledAlarms[msg.alarm.Id] = msg.alarm
          case "reload":
            reload = true
          default:
            logMessage(fmt.Sprintf("Unknown msg id: %s", msg.msg))
          }
        default:
          keepReading = false
          logMessage("No handled alarms")
      }
    }

    if reload {
      alarms, err := getAlarmsFromService(settings, handledAlarms)
      if err != nil {
        cE <- alarmError(5 * time.Second)
        logMessage(err.Error())
        // try the backup
        alarms, err = getAlarmsFromCache(settings, handledAlarms)
        if err != nil {
          // very bad, so...delete and try again later?
          // TODO: more effects
          cE <- alarmError(5 * time.Second)
          fmt.Printf("Error reading alarm cache: %s\n", err.Error())
          time.Sleep(time.Second)
          continue
        }
      }

      lastRefresh = time.Now()

      // tell cA that we have some alarms
      cA <- Alarm{} // reset hack
      for i:=0;i<len(alarms);i++ {
        cA <- alarms[i]
      }
    } else {
      // wait a little
      time.Sleep(100 * time.Millisecond)
    }
  }
}

func checkAlarm(settings *Settings, cA chan Alarm, cE chan Effect, cL chan LoaderMsg) {
  defer wg.Done()

  alarms := make([]Alarm, 0)
  var lastLogSecond = -1

  for true {
    // try reading from our channel
    keepReading := true
    alarmsRead := 0
    for keepReading {
      select {
        case alm := <- cA :
          alarmsRead++
          if alm.Name == "" {
            // reset the list
            logMessage("Reset alarm list")
            alarms = make([]Alarm, 0)
          } else {
            logMessage(fmt.Sprintf("Alarm: %+v", alm))
            alarms = append(alarms, alm)
          }
        default:
          keepReading = false
      }
    }

    // alarms come in sorted with soonest first
    for index:=0;index<len(alarms);index++ {
      if alarms[index].disabled {
        continue // skip processed alarms
      }

      now := time.Now()
      duration := alarms[index].When.Sub(now)
      if lastLogSecond != now.Second() && now.Second() % 30 == 0 {
        lastLogSecond = now.Second()
        logMessage(fmt.Sprintf("Time to next alarm: %ds (%ds to countdown)", duration / time.Second, (duration - settings.GetDuration("countdownTime"))/time.Second))
      }

      if (duration > 0) {
        // start a countdown?
        countdown := settings.GetDuration("countdownTime")
        if (duration < countdown) {
          cE <- setCountdownMode(alarms[0])
          alarms[index].countdown = true
        }
      } else {
        // Set alarm mode
        cE <- setAlarmMode(alarms[index])
        // let someone know we handled it
        cL <- handledMessage(alarms[index])
        alarms[index].disabled = true
      }
      break
    }
    // take some time off
    time.Sleep(100 * time.Millisecond)
  }
}
