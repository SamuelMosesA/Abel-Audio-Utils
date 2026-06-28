package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	// Histograms
	RecordingLatency    metric.Float64Histogram
	AudioLoopLatency    metric.Float64Histogram

	// Counters
	DroppedConnections  metric.Int64Counter
	AITokensConsumed    metric.Int64Counter
	ProcessedChunks     metric.Int64Counter
	SubtitlesSent       metric.Int64Counter
	AIEventsReceived     metric.Int64Counter
	AIAudioDeltasReceived metric.Int64Counter
)

// InitMetrics registers standard application metrics.
func InitMetrics() error {
	meter := otel.Meter("abel")
	var err error

	RecordingLatency, err = meter.Float64Histogram("recording_latency_ms",
		metric.WithDescription("Duration of audio file writes in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	AudioLoopLatency, err = meter.Float64Histogram("audio_loop_latency_ms",
		metric.WithDescription("Duration of the audio processing loop iteration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	DroppedConnections, err = meter.Int64Counter("dropped_connections_total",
		metric.WithDescription("Total number of dropped client connections (e.g. websocket failures)"),
		metric.WithUnit("{connections}"),
	)
	if err != nil {
		return err
	}

	AITokensConsumed, err = meter.Int64Counter("ai_tokens_consumed_total",
		metric.WithDescription("Total number of AI tokens consumed by the translator and transcriber sessions"),
		metric.WithUnit("{tokens}"),
	)
	if err != nil {
		return err
	}

	ProcessedChunks, err = meter.Int64Counter("processed_chunks_total",
		metric.WithDescription("Total number of processed audio chunks"),
		metric.WithUnit("{chunks}"),
	)
	if err != nil {
		return err
	}

	SubtitlesSent, err = meter.Int64Counter("subtitles_sent_total",
		metric.WithDescription("Total number of subtitles/segments sent to listeners"),
		metric.WithUnit("{subtitles}"),
	)
	if err != nil {
		return err
	}

	AIEventsReceived, err = meter.Int64Counter("ai_events_received_total",
		metric.WithDescription("Total number of events received from the AI provider websocket"),
		metric.WithUnit("{events}"),
	)
	if err != nil {
		return err
	}

	AIAudioDeltasReceived, err = meter.Int64Counter("ai_audio_deltas_received_total",
		metric.WithDescription("Total number of audio deltas received from the translation session"),
		metric.WithUnit("{deltas}"),
	)
	if err != nil {
		return err
	}

	return nil
}
