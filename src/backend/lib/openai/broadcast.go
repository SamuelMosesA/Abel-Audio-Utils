package openai

import (
	"encoding/json"

	sync_map "github.com/zolstein/sync-map"
)

// SubtitleMessage is the standard JSON payload streamed to subtitle subscribers.
type SubtitleMessage struct {
	Text   string `json:"text"`
	Tokens int    `json:"tokens"`
}

// BroadcastSubtitle encodes subtitle text once and non-blockingly delivers it
// to all active subscriber channels for the given language.
func BroadcastSubtitle(subscribers *sync_map.Map[string, []chan string], language, text string) {
	if subscribers == nil || language == "" || text == "" {
		return
	}
	subs, ok := subscribers.Load(language)
	if !ok || len(subs) == 0 {
		return
	}
	payload, err := json.Marshal(SubtitleMessage{Text: text, Tokens: 0})
	if err != nil {
		return
	}
	msg := string(payload)
	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
		}
	}
}
