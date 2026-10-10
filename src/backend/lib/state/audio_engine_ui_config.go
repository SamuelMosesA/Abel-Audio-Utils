package state

import "strings"

// AudioEngineUIConfig holds audio interface channel selection, digital gain, and sample rate.
// InterfaceConfig is an alias for AudioEngineUIConfig.
type InterfaceConfig = AudioEngineUIConfig

type AudioEngineUIConfig struct {
	deviceID   int32
	chL        int32
	chR        int32
	boost      float64
	isRunning  bool
	sampleRate int32
}

func (c AudioEngineUIConfig) DeviceID() int32   { return c.deviceID }
func (c AudioEngineUIConfig) ChL() int32        { return c.chL }
func (c AudioEngineUIConfig) ChR() int32        { return c.chR }
func (c AudioEngineUIConfig) Boost() float64    { return c.boost }
func (c AudioEngineUIConfig) IsRunning() bool   { return c.isRunning }
func (c AudioEngineUIConfig) SampleRate() int32 { return c.sampleRate }

// For internal use during updates
func (c *AudioEngineUIConfig) SetDeviceID(id int32)   { c.deviceID = id }
func (c *AudioEngineUIConfig) SetIsRunning(b bool)    { c.isRunning = b }
func (c *AudioEngineUIConfig) SetChL(ch int32)        { c.chL = ch }
func (c *AudioEngineUIConfig) SetChR(ch int32)        { c.chR = ch }
func (c *AudioEngineUIConfig) SetBoost(b float64)     { c.boost = b }
func (c *AudioEngineUIConfig) SetSampleRate(sr int32) { c.sampleRate = sr }

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
	interfaceCfg AudioEngineUIConfig
	aiCfg        AIConfig
}
