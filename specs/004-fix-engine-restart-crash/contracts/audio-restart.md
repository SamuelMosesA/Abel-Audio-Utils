# API Contract: Audio Engine Restart

## Endpoint: `POST /api/audio/restart`

Restarts the audio engine subsystem, re-scans hardware devices, reloads configuration, and reconnects to the previously active audio interface.

### Authentication
- Session Cookie (`abel_session`) or Basic Auth credentials.

### Request
- No request body required.

### Responses

#### 200 OK
Audio engine successfully restarted and devices scanned.

```json
{
  "devices": [
    {
      "id": 0,
      "name": "HD-Audio Generic: ALC245 Analog (hw:1,0)",
      "inputs": 2
    }
  ],
  "deviceID": 0,
  "reconnected": "HD-Audio Generic: ALC245 Analog (hw:1,0)",
  "configError": ""
}
```

#### 401 Unauthorized
Session invalid or unauthenticated.

```json
{
  "error": "Unauthorized session"
}
```

#### 409 Conflict
An audio recording is active, or an engine restart is already in progress.

```json
{
  "error": "Cannot restart the engine while recording"
}
```
or
```json
{
  "error": "Audio engine restart already in progress"
}
```

#### 500 Internal Server Error
A failure occurred during hardware re-initialization.

```json
{
  "error": "reinitialize audio: <driver error details>"
}
```
