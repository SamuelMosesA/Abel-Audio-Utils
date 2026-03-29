package audioengine

import (
	pa "github.com/gordonklaus/portaudio"
)

type AudioDevice struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	In   int    `json:"inputs"`
}

func GetDevices(devices []*pa.DeviceInfo) []AudioDevice {
	var list []AudioDevice
	for i, d := range devices {
		if d.MaxInputChannels > 0 {
			list = append(list, AudioDevice{
				ID:   i,
				Name: d.Name,
				In:   d.MaxInputChannels,
			})
		}
	}
	return list
}
