package audioengine

import (
	pa "github.com/gordonklaus/portaudio"
)

type AudioDevice struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	In   int    `json:"inputs"`
}

// InputDevices keeps only the devices that can capture audio.
func InputDevices(all []*pa.DeviceInfo) []*pa.DeviceInfo {
	var inputs []*pa.DeviceInfo
	for _, d := range all {
		if d.MaxInputChannels > 0 {
			inputs = append(inputs, d)
		}
	}
	return inputs
}

func GetDevices(devices []*pa.DeviceInfo) []AudioDevice {
	list := []AudioDevice{}
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
