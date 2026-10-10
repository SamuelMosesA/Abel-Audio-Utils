package state

import "strings"

// InterfaceConfig holds audio interface channel selection, digital gain, and sample rate.
type InterfaceConfig struct {
	deviceID   int32
	chL        int32
	chR        int32
	boost      float64
	isRunning  bool
	sampleRate int32
}

func (c InterfaceConfig) DeviceID() int32   { return c.deviceID }
func (c InterfaceConfig) ChL() int32        { return c.chL }
func (c InterfaceConfig) ChR() int32        { return c.chR }
func (c InterfaceConfig) Boost() float64    { return c.boost }
func (c InterfaceConfig) IsRunning() bool   { return c.isRunning }
func (c InterfaceConfig) SampleRate() int32 { return c.sampleRate }

// For internal use during updates
func (c *InterfaceConfig) SetDeviceID(id int32)   { c.deviceID = id }
func (c *InterfaceConfig) SetIsRunning(b bool)    { c.isRunning = b }
func (c *InterfaceConfig) SetChL(ch int32)        { c.chL = ch }
func (c *InterfaceConfig) SetChR(ch int32)        { c.chR = ch }
func (c *InterfaceConfig) SetBoost(b float64)     { c.boost = b }
func (c *InterfaceConfig) SetSampleRate(sr int32) { c.sampleRate = sr }

type AIConfig struct {
	Enabled          bool
	blockedLanguages map[string]bool
}

func (c AIConfig) IsEnabled() bool    { return c.Enabled }
func (c *AIConfig) SetEnabled(b bool) { c.Enabled = b }

func (c AIConfig) IsBlocked(lang string) bool {
	if c.blockedLanguages == nil || lang == "" {
		return false
	}
	return c.blockedLanguages[strings.ToLower(lang)]
}

func (c *AIConfig) SetBlocked(lang string, blocked bool) {
	if lang == "" {
		return
	}
	if c.blockedLanguages == nil {
		c.blockedLanguages = make(map[string]bool)
	}
	code := strings.ToLower(lang)
	if blocked {
		c.blockedLanguages[code] = true
	} else {
		delete(c.blockedLanguages, code)
	}
}

func (c AIConfig) BlockedLanguages() map[string]bool {
	if c.blockedLanguages == nil {
		return make(map[string]bool)
	}
	cloned := make(map[string]bool, len(c.blockedLanguages))
	for k, v := range c.blockedLanguages {
		cloned[k] = v
	}
	return cloned
}

type configState struct {
	interfaceCfg InterfaceConfig
	aiCfg        AIConfig
}
