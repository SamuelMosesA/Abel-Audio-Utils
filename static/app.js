let ws = null;
let audioCtx = null;
let isRecording = false;

// --- PRECISE TIMING STATE ---
let nextStartTime = 0;
const LATENCY_BUFFER = 0.1; // 100ms for smooth play

// Visual Physics
let targetL = 0, targetR = 0, currentL = 0, currentR = 0;
const DECAY = 0.25;

// 1. Fetch Devices
window.onload = async () => {
    try {
        const res = await fetch("/api/devices");
        const devices = await res.json();
        const sel = document.getElementById("deviceSelect");
        sel.innerHTML = "";
        devices.forEach(d => {
            const opt = document.createElement("option");
            opt.value = d.id;
            opt.innerText = `[${d.id}] ${d.name} (${d.inputs} in)`;
            sel.appendChild(opt);
        });
    } catch (e) {
        document.getElementById("deviceSelect").innerHTML = "<option>Error loading devices</option>";
    }
    connectWebSocket();
};

// 2. Connect Device (Backend)
async function connectDevice() {
    const id = parseInt(document.getElementById("deviceSelect").value);
    const btn = document.getElementById("btnConnect");
    btn.innerText = "Connecting...";
    try {
        await fetch("/api", { method: "POST", body: JSON.stringify({ action: "connect", DeviceID: id }) });
        btn.innerText = "Connected!";
        btn.style.background = "#4caf50";
        btn.style.color = "white";
        document.getElementById("btnRec").disabled = false;
        document.getElementById("monitorToggle").disabled = false;
        document.getElementById("status").innerText = "Status: Engine Running";
    } catch (e) {
        btn.innerText = "Failed";
        alert("Error starting engine");
    }
}

// 3. WebSocket (Audio + Meters)
function connectWebSocket() {
    const protocol = location.protocol === 'https:' ? 'wss://' : 'ws://';
    ws = new WebSocket(protocol + location.host + "/ws");
    ws.binaryType = "arraybuffer";

    ws.onmessage = (event) => {
        const dv = new DataView(event.data);

        // A. Meters (Bytes 0-7)
        const rL = dv.getFloat32(0, true);
        const rR = dv.getFloat32(4, true);
        targetL = Math.min(Math.sqrt(rL) * 100, 100);
        targetR = Math.min(Math.sqrt(rR) * 100, 100);

        // B. Audio (Bytes 8+)
        if (document.getElementById("monitorToggle").checked && audioCtx) {
            scheduleAudio(dv, 8);
        } else {
            nextStartTime = 0;
        }
    };

    ws.onclose = () => setTimeout(connectWebSocket, 2000);
}

// --- THE SCHEDULER (The Fix) ---
function scheduleAudio(dataView, offset) {
    if (audioCtx.state === 'suspended') audioCtx.resume();

    // Copy data to avoid memory leaks
    const floatData = new Float32Array(dataView.buffer.slice(offset));

    // Create Audio Buffer
    const buffer = audioCtx.createBuffer(2, floatData.length / 2, 48000);
    const chL = buffer.getChannelData(0);
    const chR = buffer.getChannelData(1);

    // De-Interleave
    for (let i = 0; i < floatData.length / 2; i++) {
        chL[i] = floatData[i * 2];
        chR[i] = floatData[i * 2 + 1];
    }

    const now = audioCtx.currentTime;

    // If lagging or drifting, snap to now
    if (nextStartTime < now || nextStartTime > now + 1.0) {
        nextStartTime = now + LATENCY_BUFFER;
    }

    const source = audioCtx.createBufferSource();
    source.buffer = buffer;
    source.connect(audioCtx.destination);
    source.start(nextStartTime);
    nextStartTime += buffer.duration;
}

// 4. Toggle Monitor
document.getElementById("monitorToggle").addEventListener('change', async (e) => {
    if (e.target.checked) {
        if (!audioCtx) audioCtx = new (window.AudioContext || window.webkitAudioContext)({ latencyHint: 'interactive', sampleRate: 48000 });
        await audioCtx.resume();
    } else {
        if (audioCtx) audioCtx.suspend();
    }
});

// 5. Visual Loop
function draw() {
    currentL -= (currentL - targetL) * DECAY;
    currentR -= (currentR - targetR) * DECAY;
    if (currentL < 0) currentL = 0; if (currentR < 0) currentR = 0;

    const bL = document.getElementById("meterL");
    const bR = document.getElementById("meterR");
    if (bL && bR) {
        bL.style.width = currentL + "%";
        bR.style.width = currentR + "%";
        bL.style.background = currentL > 95 ? "#ff5252" : "#4caf50";
        bR.style.background = currentR > 95 ? "#ff5252" : "#4caf50";
    }
    requestAnimationFrame(draw);
}
requestAnimationFrame(draw);

// 6. Config Controls
async function toggleRec() {
    const action = isRecording ? "stop" : "start";
    const boost = parseFloat(document.getElementById("boost").value);

    await fetch("/api", { method: "POST", body: JSON.stringify({ action, Boost: boost }) });
    isRecording = !isRecording;
    updateUI();
}

async function updateConfig() {
    const l = parseInt(document.getElementById("chL").value);
    const r = parseInt(document.getElementById("chR").value);
    const boost = parseFloat(document.getElementById("boost").value);
    await fetch("/api", { method: "POST", body: JSON.stringify({ action: "update", chL: l, chR: r, Boost: boost }) });
}

function updateUI() {
    const btn = document.getElementById("btnRec");
    const stat = document.getElementById("status");

    // Disable inputs during recording
    document.getElementById("btnUpdateConfig").disabled = isRecording;
    document.getElementById("deviceSelect").disabled = isRecording;
    document.getElementById("btnConnect").disabled = isRecording;

    if (isRecording) {
        btn.innerText = "Stop Recording";
        btn.className = "btn-stop";
        stat.innerText = "● RECORDING";
        stat.classList.add("recording-active");
    } else {
        btn.innerText = "Start Recording";
        btn.className = "btn-rec";
        stat.innerText = "Engine Running";
        stat.classList.remove("recording-active");
    }
}