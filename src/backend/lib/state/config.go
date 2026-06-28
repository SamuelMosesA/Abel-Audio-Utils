package state

type InterfaceConfig struct {
	deviceID   int32
	chL        int32
	chR        int32
	boost      float64
	isRunning  bool
	sampleRate int32
}

func (c InterfaceConfig) DeviceID() int32  { return c.deviceID }
func (c InterfaceConfig) ChL() int32       { return c.chL }
func (c InterfaceConfig) ChR() int32       { return c.chR }
func (c InterfaceConfig) Boost() float64   { return c.boost }
func (c InterfaceConfig) IsRunning() bool { return c.isRunning }
func (c InterfaceConfig) SampleRate() int32  { return c.sampleRate }

// For internal use during updates
func (c *InterfaceConfig) SetDeviceID(id int32) { c.deviceID = id }
func (c *InterfaceConfig) SetIsRunning(b bool) { c.isRunning = b }
func (c *InterfaceConfig) SetChL(ch int32)     { c.chL = ch }
func (c *InterfaceConfig) SetChR(ch int32)     { c.chR = ch }
func (c *InterfaceConfig) SetBoost(b float64)  { c.boost = b }
func (c *InterfaceConfig) SetSampleRate(sr int32) { c.sampleRate = sr }

type AIConfig struct {
	Enabled bool
}

func (c AIConfig) IsEnabled() bool { return c.Enabled }
func (c *AIConfig) SetEnabled(b bool) { c.Enabled = b }

type configState struct {
	interfaceCfg InterfaceConfig
	aiCfg        AIConfig
}
