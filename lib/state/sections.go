package state

import "fmt"

type Section int

const (
	SectionInterface Section = iota
	SectionGemini
	SectionRecording
	SectionLocations
)

func (s Section) String() string {
	switch s {
	case SectionInterface:
		return "interface"
	case SectionGemini:
		return "gemini"
	case SectionRecording:
		return "recording"
	case SectionLocations:
		return "locations"
	default:
		return "unknown"
	}
}

// Update is a generic function to update a specific section of the state in a thread-safe manner.
func Update[T any](s *AppState, sec Section, fn func(*T)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var sectionPtr interface{}

	switch sec {
	case SectionInterface:
		sectionPtr = &s.config.interfaceCfg
	case SectionGemini:
		sectionPtr = &s.config.geminiCfg
	case SectionRecording:
		sectionPtr = &s.intent
	case SectionLocations:
		sectionPtr = &s.static
	default:
		return fmt.Errorf("unknown section: %v", sec)
	}

	typedPtr, ok := sectionPtr.(*T)
	if !ok {
		return fmt.Errorf("type mismatch for section %s", sec.String())
	}

	fn(typedPtr)

	// Broadcast the change
	s.Broadcast(sec)

	return nil
}
