package portaudio

import (
	"behringerRecorder/lib/types"

	pa "github.com/gordonklaus/portaudio"
)

func GetDevices(devices []*pa.DeviceInfo) []types.AudioDevice {
	var list []types.AudioDevice
	for i, d := range devices {
		if d.MaxInputChannels > 0 {
			list = append(list, types.AudioDevice{
				ID:   i,
				Name: d.Name,
				In:   d.MaxInputChannels,
			})
		}
	}
	return list
}
