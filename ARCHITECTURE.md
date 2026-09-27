```
flowchart TB
  subgraph Init ["1. Process Initialization"]
    START([os/exec → main.go]) --> HOME[Resolve ~/.config/abel/config.yaml]
    HOME -->|missing| DIE([os.Exit 1])
    HOME --> LOAD[config.LoadConfig]
    LOAD --> OTEL[telemetry.InitTelemetry]
    OTEL --> PA[portaudio.Initialize]
    PA --> STATE[state.NewAppState]
    STATE --> SEED["Seed InterfaceConfig (chL, chR, boost, sampleRate)"]
    SEED --> AIINIT{OpenAI API key?}
    AIINIT -->|yes| TM[openai.NewOpenAIManager]
    AIINIT -->|no| WARN[AI disabled]
    TM --> DEVS[Enumerate PortAudio input devices]
    WARN --> DEVS
    DEVS --> WORKERS[Start workers]
    WORKERS --> BCAST["StartAudioBroadcaster (from PlaybackChan)"]
    WORKERS --> STOR["StartStorageWorker (from RecordChan)"]
    BCAST --> ROUTER[web.NewRouter + embed static]
    STOR --> ROUTER
    ROUTER --> LISTEN([Gin listen 0.0.0.0:port])
  end

  subgraph API ["2. HTTP / WS Entry Points"]
    LISTEN --> EP{{Incoming request}}
    EP -->|POST /api/auth/session| AUTH["LoginHandler (set abel_session cookie)"]
    EP -->|GET static pages| UI[Serve embedded SvelteKit HTML]
    EP -->|GET /ws + cookie| WS["Upgrade → WSClient in Clients map"]
    EP -->|public REST| PUB["devices / config / stream / subtitles"]
    EP -->|admin REST + SessionAuth| ADM["config PATCH / recordings / AI / files"]
  end

  subgraph Actions ["3. Admin Control Actions"]
    ADM -->|PATCH /api/audio/config| ENG[StartAudioEngine goroutine]
    ADM -->|POST /api/recordings start| REC_ON["Open WAV file (SetRecording true)"]
    ADM -->|POST /api/ai/streams| AI_ON[Translator.SetEnabled]
  end

  subgraph Pipeline ["4. Audio Engine & Fan-out"]
    ENG --> LOOP[PortAudio Read loop]
    LOOP --> ROUTE["Select chL/chR + boost + clamp → stereo float32"]
    ROUTE -->|non-blocking| RC[(RecordChan)]
    ROUTE -->|non-blocking| PC[(PlaybackChan)]

    RC --> STOR2[StorageWorker]
    STOR2 -->|if recording + file| WAV[(disk WAV int16 LE)]

    PC --> BC[AudioBroadcaster]
    BC --> PKT["Build binary packet (peakL + peakR + PCM)"]
    PKT --> WS_OUT[WriteMessage → admin WS clients]
    BC --> HTTP_FAN[Fan-out StreamChannels]
    BC -->|if AI enabled| PUSH[Translator.PushAudio]
  end

  subgraph Streams ["5. AI & Streaming Consumption"]
    PUB -->|GET /api/audio/stream| STREAM_DEF["Subscribe StreamChannels or translated channel"]
    HTTP_FAN --> STREAM_DEF
    STREAM_DEF --> WAV_HTTP[Write forever WAV header + PCM]

    PUSH --> OAI{lang == original?}
    OAI -->|yes| TX["TranscriptionManager (OpenAI Realtime WS)"]
    OAI -->|no| TR["TranslationManager (OpenAI Realtime WS)"]
    TX --> SUB_CH[(subtitle chan)]
    TR --> SUB_CH
    TR --> AUD_CH[(translated audio chan)]
    TX --> AUD_CH

    PUB -->|GET /api/ai/subtitles/lang| SSE[SSE write subtitle text]
    SUB_CH --> SSE
    AUD_CH --> STREAM_DEF

    ADM --> UPD[state.Update Section]
    UPD --> HUB[(BroadcastHub)]
    HUB -->|GET /api/system/changelog| SYNC[Admin UI re-sync stores]
  end

```
