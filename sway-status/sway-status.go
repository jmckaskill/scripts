package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/mdlayher/wifi"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func findWiFiInterface(w *wifi.Client, name string) (*wifi.Interface, error) {
	ifaces, err := w.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Name == name {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("could not find interface %v", name)
}

func main() {
	wakeup := make(chan os.Signal)
	signal.Notify(wakeup, syscall.SIGALRM)
	tk := time.NewTicker(2 * time.Second)

	d, err := dbus.ConnectSystemBus()
	must(err)
	defer d.Close()

	w, err := wifi.New()
	must(err)
	defer w.Close()

	iface, err := findWiFiInterface(w, "wlp6s0")
	must(err)

	for {

		res := map[string]any{}
		err = d.Object("org.freedesktop.UPower", "/org/freedesktop/UPower/devices/battery_BAT0").
			Call("org.freedesktop.DBus.Properties.GetAll", 0, "org.freedesktop.UPower.Device").
			Store(&res)
		must(err)

		powerRate := res["EnergyRate"].(float64)
		batState := ""
		batTimeRemaining := time.Duration(0)
		switch res["State"].(uint32) {
		case 1:
			batState = "Charging"
			batTimeRemaining = time.Duration(res["TimeToFull"].(int64)) * time.Second
		case 2:
			batState = "Discharging"
			batTimeRemaining = time.Duration(res["TimeToEmpty"].(int64)) * time.Second
		case 3:
			batState = "Empty"
		case 4:
			batState = "Fully Charged"
		case 5:
			batState = "Pending Charge"
		case 6:
			batState = "Pending Discharge"
		default:
			batState = "Unknown"
		}
		batPercent := res["Percentage"]

		wifiText := ""
		bss, err := w.BSS(iface)
		if err == nil {
			ssid := bss.SSID

			stations, err := w.StationInfo(iface)
			if err == nil && len(stations) > 0 {
				rssi := stations[0].Signal // signal value in dBm
				quality := 0
				if rssi > -50 {
					quality = 100
				} else if rssi > -100 {
					quality = 2 * (rssi + 100)
				}

				wifiText = fmt.Sprintf("WiFi:(%s|%v%%)", ssid, quality)
			}
		}

		vol, err := exec.Command("ponymix", "get-volume").Output()
		must(err)
		volstr := strings.TrimSpace(string(vol))

		if err := exec.Command("ponymix", "is-muted").Run(); err == nil {
			volstr = "muted"
		}

		bright, err := exec.Command("brillo").Output()
		must(err)
		brightnum, err := strconv.ParseFloat(strings.TrimSpace(string(bright)), 32)
		must(err)

		tm := time.Now()
		tmstr := tm.Format("2006-01-02 15:04")
		_, week := tm.ISOWeek()
		day := tm.Weekday()
		if day == 0 {
			day = 7
		}
		fmt.Printf("%s Brightness:%.0f Volume:%s Battery:(%s|%.0f%%|%s|%.1fW) %02d.%d %s\n",
			wifiText,
			brightnum,
			volstr,
			batState, batPercent, batTimeRemaining, powerRate,
			week, day,
			tmstr)

		select {
		case <-wakeup:
		case <-tk.C:
		}
	}
}
