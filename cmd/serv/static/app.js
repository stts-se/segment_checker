'use strict';

const baseURL = window.location.protocol + '//' + window.location.host + window.location.pathname.replace(/\/$/g, "");
const wsBase = baseURL.replace(/^http/, "ws");
const clientID = LIB.uuidv4();
let ws;

let gloptions = {
    boundaryMovementShort: 5,
    boundaryMovementLong: 100,
    requestStatus: "Unchecked",
    autoplay: "None",
    //context: -1,
    //userName: "nizze",
}

let enabled = false;
let waveform;
let cachedSegment;

let debugVar;

function logWarning(msg) {
    let div = logMessage(msg);
    div.style.color = "orange";
}

function logError(msg) {
    let div = logMessage(msg);
    div.style.color = "red";
}

function logMessage(msg) {
    let div = document.createElement("div");
    div.innerText = LIB.timestampHHMMSS() + " " + msg;
    // if (msg.toLowerCase().includes("error"))
    // 	div.style.color = "red";
    messages.prepend(div);
    return div;
}

function lockGUI() {
    setEnabled(false);
    enableStart(false);
}

function enableStart(enable) {
    if (enable) {
	document.getElementById("start").disabled = false;
	document.getElementById("start").classList.remove("disabled");
    } else {
	document.getElementById("start").disabled = true;
	document.getElementById("start").classList.add("disabled");
    }
}

function setEnabled(enable) {
    document.getElementById("unlock-all").disabled = false;
    document.getElementById("unlock-all").classList.remove("disabled");

    enabled = enable;
    let buttons = [
        document.getElementById("save-badsample"),
        document.getElementById("save-skip"),
        document.getElementById("save-ok"),
        document.getElementById("save-badsample-next"),
        document.getElementById("save-skip-next"),
        document.getElementById("save-ok-next"),
        document.getElementById("play-all"),
        document.getElementById("play-label"),
        document.getElementById("play-right"),
        document.getElementById("play-left"),
        document.getElementById("reset"),
        document.getElementById("quit"),
        document.getElementById("next"),
        document.getElementById("prev"),
        document.getElementById("first"),
        document.getElementById("last"),
        document.getElementById("next_any"),
        document.getElementById("prev_any"),
    ];
    if (enable) {
        for (let i = 0; i < buttons.length; i++) {
            let btn = buttons[i];
            if (btn) {
                btn.classList.remove("disabled");
                btn.removeAttribute("disabled");
                btn.disabled = false;
            }
        }
	// document.getElementById("start").disabled = true;
        // document.getElementById("start").classList.add("disabled");
        document.getElementById("comment").removeAttribute("readonly");
    } else {
        document.getElementById("comment").setAttribute("readonly", "readonly");
        for (let i = 0; i < buttons.length; i++) {
            let btn = buttons[i];
            if (btn) {
                btn.classList.add("disabled");
                btn.disabled = true;
            }
        }
	// document.getElementById("start").disabled = false;
        // document.getElementById("start").classList.remove("disabled");
    }
    enableStart(!enable);
}

function autoplay() {
    console.log("autoplay called");
    if (document.getElementById("autoplay-none").checked)
        return;
    if (document.getElementById("autoplay-right").checked)
        document.getElementById("play-right").click();
    else if (document.getElementById("autoplay-left").checked)
        document.getElementById("play-left").click();
    else if (document.getElementById("autoplay-all").checked)
        document.getElementById("play-all").click();
    else if (document.getElementById("autoplay-label").checked)
        document.getElementById("play-label").click();
}

async function loadAudioBlob(url, chunk) {
    waveform.loadAudioBlob(url, [chunk]);
    // waveform.wavesurfer.on("region-created", function (region) {
    //     autoplay();
    // });
}

function loadAudioURL(url, chunk) {
    waveform.loadAudioURL(url, [chunk]);
    // waveform.wavesurfer.on("region-created", function (region) {
    //     autoplay();
    // });
}

document.getElementById("play-all").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        waveform.play(0.0);
});
document.getElementById("play-label").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        waveform.playRegionIndex(0);
});
document.getElementById("play-left").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        waveform.playLeftOfRegionIndex(0);
});
document.getElementById("play-right").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        waveform.playRightOfRegionIndex(0);
});

// document.getElementById("save-badsample").addEventListener("click", function (evt) {
//     if (!evt.target.disabled)
//         save({ status: "skip", label: "bad sample" });
// });
// document.getElementById("save-skip").addEventListener("click", function (evt) {
//     if (!evt.target.disabled)
//         save({ status: "skip" });
// });
// document.getElementById("save-ok").addEventListener("click", function (evt) {
//     if (!evt.target.disabled)
//         save({ status: "ok" });
// });
document.getElementById("save-badsample-next").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        saveUnlockAndNext({ status: "skip", label: "bad sample", stepSize: 1 });
});
document.getElementById("save-skip-next").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        saveUnlockAndNext({ status: "skip", stepSize: 1 });
});
document.getElementById("save-ok-next").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        saveUnlockAndNext({ status: "ok", stepSize: 1 });
});

if (document.getElementById("first")) {
    document.getElementById("first").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ requestIndex: "first" });
    });
}
if (document.getElementById("last")) {
    document.getElementById("last").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ requestIndex: "last" });
    });
}
if (document.getElementById("start")) {
    document.getElementById("start").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ stepSize: 1 });
    });
}
if (document.getElementById("next")) {
    document.getElementById("next").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ stepSize: 1 });
    });
}
if (document.getElementById("prev")) {
    document.getElementById("prev").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ stepSize: -1 });
    });
}

if (document.getElementById("prev_any")) {
    document.getElementById("prev_any").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ stepSize: -1, requestStatus: "any" });
    });
}
if (document.getElementById("next_any")) {
    document.getElementById("next_any").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveUnlockAndNext({ stepSize: 1, requestStatus: "any" });
    });
}

function clear() {
    if (waveform)
        waveform.clear();
    document.getElementById("comment").value = "";
    //document.getElementById("labels").innerText = "";
    document.getElementById("current_status").innerText = "";
    document.getElementById("current_status_div").style.backgroundColor = "";
    document.getElementById("current_status_div").style.borderColor = "";
    document.getElementById("segment_id").innerHTML = "&nbsp;";
}

document.getElementById("reset").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.updateRegion(0, cachedSegment.chunk.start, cachedSegment.chunk.end);
        if (cachedSegment.comment)
            document.getElementById("comment").value = cachedSegment.comment;
        else
            document.getElementById("comment").value = "";
    }
});
document.getElementById("quit").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        unlockCurrentSegment();
        setEnabled(false);
        clear();
    }
});

document.getElementById("unlock-all").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        unlockAll();
        setEnabled(false);
        clear();
    }
});

document.getElementById("load_stats").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        let request = {
            'client_id': clientID,
            'message_type': 'stats',
        };
        ws.send(JSON.stringify(request));
    }
});

document.getElementById("move-left2left-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(0, -gloptions.boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-left2right-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(0, gloptions.boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2left-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(0, -gloptions.boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2right-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(0, gloptions.boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});

document.getElementById("move-left2left-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(0, -gloptions.boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-left2right-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(0, gloptions.boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2left-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(0, -gloptions.boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2right-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(0, gloptions.boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});

function displayAudioChunk(chunk) {
    clear();
    lockGUI();
    // https://stackoverflow.com/questions/16245767/creating-a-blob-from-a-base64-string-in-javascript#16245768
    let byteCharacters = atob(chunk.audio);
    let byteNumbers = new Array(byteCharacters.length);
    for (let i = 0; i < byteCharacters.length; i++) {
        byteNumbers[i] = byteCharacters.charCodeAt(i);
    }
    let byteArray = new Uint8Array(byteNumbers);

    cachedSegment = chunk;
    cachedSegment.audio = null; // no need to cache the audio blob
    console.log("res => cache", JSON.stringify(cachedSegment));

    let blob = new Blob([byteArray], { 'type': chunk.file_type });
    loadAudioBlob(blob, chunk.chunk);
    document.getElementById("segment_id").innerText = chunk.index + " | " + chunk.id;

    // status info + color code
    let status = chunk.current_status.name;
    let statusDiv = document.getElementById("current_status_div");
    if (chunk.labels && chunk.labels.includes("bad sample"))
        status = "bad sample";
    if (status === "ok")
        statusDiv.style.borderColor = "lightgreen";
    else if (status === "bad sample")
        statusDiv.style.borderColor = "#ff5757";
    else if (status === "skip")
        statusDiv.style.borderColor = "orange";
    else if (status === "unchecked")
        statusDiv.style.borderColor = "lightgrey";
    else
        statusDiv.style.borderColor = "none";

    if (chunk.current_status.source)
        status += " (" + chunk.current_status.source + ")";
    if (chunk.current_status.timestamp)
        status += " | " + chunk.current_status.timestamp;
    document.getElementById("current_status").innerText = status;

    // comment
    if (chunk.comment)
        document.getElementById("comment").value = chunk.comment;

    // labels => integrated as status
    // if (chunk.labels && chunk.labels.length > 0)
    //     document.getElementById("labels").innerText = chunk.labels;
    // else
    //     document.getElementById("labels").innerText = "none";
    setEnabled(true);
    logMessage("Loaded segment " + chunk.id + " from server");
}

function displayStats(stats) {
    logMessage("Received stats from server");
    let ele = document.getElementById("stats");
    ele.innerText = "";
    let keys = Object.keys(stats)
    keys.sort();
    keys.forEach(function (key) {
        let tr = document.createElement("tr");
        let td1 = document.createElement("td");
        let td2 = document.createElement("td");
        td2.style["text-align"] = "right";
        td1.innerHTML = key;
        td2.innerHTML = stats[key];
        tr.appendChild(td1);
        tr.appendChild(td2);
        ele.appendChild(tr);
    });
    let timestamp = new Date().toLocaleTimeString("sv-SE");
    document.getElementById("stats_timestamp").innerText = timestamp;
}

function unlockCurrentSegment() {
    console.log("unlockCurrentSegment called")
    if (cachedSegment === undefined || cachedSegment === null)
        return;

    let request = {
        'client_id': clientID,
        'message_type': 'unlock',
        'payload': JSON.stringify({
            'segment_id': cachedSegment.id,
            'user_name': document.getElementById("username").innerText,
        }),
    };
    ws.send(JSON.stringify(request));
}

function unlockAll() {
    console.log("unlockAll called")
    let request = {
        'client_id': clientID,
        'message_type': 'unlock_all',
        'payload': JSON.stringify({
            'user_name': document.getElementById("username").innerText,
        }),
    };
    ws.send(JSON.stringify(request));
}

document.getElementById("clear_messages").addEventListener("click", function (evt) {
    document.getElementById("messages").innerHTML = "";
});


function createQuery(stepSize, requestIndex, requestStatus) {
    let query = {
        user_name: document.getElementById("username").innerText,
    }
    if (stepSize)
        query.step_size = stepSize;
    if (requestIndex)
        query.request_index = requestIndex;
    if (gloptions.context && gloptions.context >= 0)
        query.context = parseInt(gloptions.context);
    if (cachedSegment && cachedSegment !== null)
        query.curr_id = cachedSegment.id;

    // search for status
    if (requestStatus)
	query.request_status = requestStatus;
    else
	query.request_status = document.querySelector('input[name="requeststatus"]:checked').value;
    return query;
}

// function next(stepSize) {
//     console.log("next called")
//     let request = {
//     	'client_id': clientID,
//     	'message_type': 'next',
//     	'payload': JSON.stringify(createQuery(stepSize)),
//     };
//     ws.send(JSON.stringify(request));
// }


function saveUnlockAndNext(options) {
    lockGUI();
    console.log("saveUnlockAndNext called with options", options);
    let user = document.getElementById("username").innerText;
    if ((!user) || user === "") {
        let msg = "Username unset!";
        alert(msg);
	setEnabled(false);
        logError(msg);
        return;
    }
    if (options.status && (!cachedSegment || !cachedSegment.id)) {
        let msg = "No cached segment -- cannot save!";
        alert(msg);
	setEnabled(false);
        logError(msg);
        return;
    }
    let unlock = {}
    if (cachedSegment && cachedSegment.id)
        unlock = { user_name: user, segment_id: cachedSegment.id };

    let annotation = {};
    if (options.status) { // create annotation to save
        let status = {
            source: user,
            name: options.status,
            timestamp: new Date().toLocaleString("sv-SE"),
        }
        let labels = [];
        if (options.label) {
            labels.push(options.label);
        }
        let statusHistory = cachedSegment.status_history;
        if (!statusHistory)
            statusHistory = [];
        if (cachedSegment.current_status.name !== "unchecked")
            statusHistory.push(cachedSegment.current_status);
        let region = waveform.getRegion(0)
        annotation = {
            id: cachedSegment.id,
            url: cachedSegment.url,
            segment_type: cachedSegment.segment_type,
            chunk: {
                start: region.start + cachedSegment.offset,
                end: region.end + cachedSegment.offset,
            },
            current_status: status,
            status_history: statusHistory,
            labels: labels,
            comment: document.getElementById("comment").value,
            index: cachedSegment.index,
        }
    }
    let query = createQuery(options.stepSize, options.requestIndex, options.requestStatus);

    let payload = {
        annotation: annotation,
        unlock: unlock,
        query: query,
    };
    //console.log("payload", JSON.stringify(payload));

    let request = {
        'client_id': clientID,
        'message_type': 'saveunlockandnext',
        'payload': JSON.stringify(payload),
    };
    ws.send(JSON.stringify(request));
}


onload = function () {

    setEnabled(false);
    lockGUI();
    clear();
    document.getElementById("unlock-all").disabled = true;
    document.getElementById("unlock-all").classList.add("disabled");

    let params = new URLSearchParams(window.location.search);
    if (params.get('context')) {
        gloptions.context = params.get('context');
        document.getElementById("context").innerText = `${context} ms`;
        document.getElementById("context-view").classList.remove("hidden");
    }
    if (params.get('request_status')) {
        gloptions.requestStatus = params.get('request_status').toLowerCase();
        if (document.getElementById(`requeststatus-${gloptions.requestStatus}`))
            document.getElementById(`requeststatus-${gloptions.requestStatus}`).checked = true;
        else {
            logError(`Invalid search mode: ${gloptions.requestStatus}`);
            gloptions.requestStatus = "Unchecked";
        }
    }
    if (params.get('autoplay')) {
        gloptions.autoplay = params.get('autoplay').toLowerCase();
        if (document.getElementById(`autoplay-${gloptions.autoplay}`))
            document.getElementById(`autoplay-${gloptions.autoplay}`).checked = true;
        else {
            logError(`Invalid search mode: ${gloptions.autoplay}`);
            gloptions.autoplay = "None";
        }
    }

    let requestIndex;
    if (params.get('request_index')) {
        requestIndex = parseInt(params.get('request_index').toLowerCase())-1;
	requestIndex = requestIndex + "";
    }

    if (params.get('username')) {
        gloptions.userName = params.get('username');
    }
    else {
	let suggest = localStorage.getItem("username");
	if (!suggest || suggest === null)
	    suggest = "";
        let username = prompt("User name", suggest);
	if (!username || username === null || username.trim() === "") {
            let msg = "Username unset!";
            logError(msg);
	    alert(msg);
            return;
	}
        gloptions.userName = username.toLowerCase();
    }
    document.getElementById("username").innerText = gloptions.userName;
    localStorage.setItem("username", gloptions.userName);


    console.log("gloptions", gloptions);

    let url = wsBase + "/ws/" + clientID;
    ws = new WebSocket(url);
    ws.onopen = function () {
        logMessage("Websocket opened");
	if (requestIndex)
            saveUnlockAndNext({ requestIndex: requestIndex });
	else
            saveUnlockAndNext({ stepSize: 1 });
    }
    ws.onclose = function () {
        logMessage("Websocket closed");
	clear();
	setEnabled(false);
	ws = undefined;
	alert("Application was closed from server");
    }
    ws.onmessage = function (evt) {
        let resp = JSON.parse(evt.data);
        //console.log("ws.onmessage", resp);
        if (resp.error) {
            logError("Server error: " + resp.error);
            return;
        }
        if (resp.info) {
            logMessage(resp.info);
        }
        if (resp.message_type === "stats")
            displayStats(JSON.parse(resp.payload));
        else if (resp.message_type === "explicit_unlock_completed") {
            cachedSegment = null;
            logMessage(JSON.parse(resp.payload));
        }
        else if (resp.message_type === "no_audio_chunk") {
            let msg = JSON.parse(resp.payload);
            logMessage(msg);
	    if (cachedSegment && cachedSegment !== null)
		setEnabled(true);
	    else
		setEnabled(false);
	    enableStart(true);
            alert(msg);
        }
        else if (resp.message_type === "audio_chunk")
            displayAudioChunk(JSON.parse(resp.payload));
        else if (resp.info === "" && resp.message_type !== "keep_alive")
            logWarning("Unknown message from server: [" + resp.message_type + "] " + resp.payload);
    }

    console.log("main window loaded");

    loadKeyboardShortcuts();

    let options = {
        waveformElementID: "waveform",
        timelineElementID: "waveform-timeline",
        spectrogramElementID: "waveform-spectrogram",
        // autoplayFunc: function () { return true; },
        // zoomElementID: "waveform-zoom",
        // navigationElementID: "waveform-navigation",
        debug: false,
    };

    waveform = new Waveform(options);
    // waveform.wavesurfer.on("region-created", function (region) {
    //     autoplay();
    // });

    //loadSegmentFromFile('tillstud_demo_2_Niclas_Tal_1_2020-08-24_141655_b35aa260_00021.json');

};

function loadKeyboardShortcuts() {
    let ele = document.getElementById("shortcuts");
    ele.innerHTML = "";
    Object.keys(shortcuts).forEach(function (key) {
        let id = shortcuts[key].buttonID;
        let tooltip = shortcuts[key].tooltip;
        if (!tooltip)
            tooltip = key.toLowerCase();
        if (id && tooltip) {
            let ele = document.getElementById(id);
            if (ele) {
                if (!ele.title)
                    ele.title = "key: " + tooltip;
            } else
                throw Error(`No element with id ${id}`);
        }
        if (tooltip && shortcuts[key].funcDesc) {
            let tr = document.createElement("tr");
            let td1 = document.createElement("td");
            let td2 = document.createElement("td");
            td1.innerHTML = tooltip;
            td2.innerHTML = shortcuts[key].funcDesc;
            tr.appendChild(td1);
            tr.appendChild(td2);
            ele.appendChild(tr);
        }
    });
}

const shortcuts = {
    // 'ctrl ArrowLeft': { tooltip: 'ctrl left', funcDesc: 'Move left boundary to the left', func: function () { waveform.moveStartForRegionIndex(0, -5) } },
    'ctrl ArrowLeft': { funcDesc: `Move left boundary ${gloptions.boundaryMovementShort} ms to the left`, buttonID: 'move-left2left-short' },
    'ctrl ArrowRight': { funcDesc: `Move left boundary ${gloptions.boundaryMovementShort} ms to the right`, buttonID: 'move-left2right-short' },
    'ctrl ArrowUp': { funcDesc: `Move left boundary ${gloptions.boundaryMovementLong} ms to the left`, buttonID: 'move-left2left-long' },
    'ctrl ArrowDown': { funcDesc: `Move left boundary ${gloptions.boundaryMovementLong} ms to the right`, buttonID: 'move-left2right-long' },
    'shift ArrowLeft': { funcDesc: `Move right boundary ${gloptions.boundaryMovementShort} ms to the left`, buttonID: 'move-right2left-short' },
    'shift ArrowRight': { funcDesc: `Move right boundary ${gloptions.boundaryMovementShort} ms to the right`, buttonID: 'move-right2right-short' },
    'shift ArrowUp': { funcDesc: `Move right boundary ${gloptions.boundaryMovementLong} ms to the left`, buttonID: 'move-right2left-long' },
    'shift ArrowDown': { funcDesc: `Move right boundary ${gloptions.boundaryMovementLong} ms to the right`, buttonID: 'move-right2right-long' },
    'ArrowLeft': { tooltip: 'left', funcDesc: 'Play left context', buttonID: 'play-left' },
    'ArrowRight': { tooltip: 'right', funcDesc: 'Play right context', buttonID: 'play-right' },
    'ArrowDown': { tooltip: 'down', funcDesc: 'Play all audio', buttonID: 'play-all' },
    ' ': { tooltip: 'space', funcDesc: 'Play label', buttonID: 'play-label' },
    'ctrl  ': { buttonID: 'play-label' }, // hidden from shortcut view
    'shift  ': { buttonID: 'play-label' }, // hidden from shortcut view
    // 'n': { tooltip: 'n', buttonID: 'next', funcDesc: "Get next segment" },
    // 'p': { tooltip: 'p', buttonID: 'prev', funcDesc: "Get previous segment" },
    'o': { buttonID: 'save-ok-next', funcDesc: "Save as ok and get next" },
    's': { buttonID: 'save-skip-next', funcDesc: "Save as skip and get next" },
    'b': { buttonID: 'save-badsample-next', funcDesc: "Save as skip with label 'bad sample', and get next" },
};

window.addEventListener("keydown", function (evt) {
    //console.log(evt.which);
    if (document.activeElement.tagName.toLowerCase() === "textarea")
        return;
    let key = evt.key;
    if (evt.altKey)
        key = "alt " + key;
    if (evt.ctrlKey)
        key = "ctrl " + key;
    if (evt.shiftKey)
        key = "shift " + key;
    //console.log(evt.key, evt.keyCode, evt.ctrlKey, evt.altKey, "=>", key);
    if (shortcuts[key]) {
        let shortcut = shortcuts[key];
        if ((!shortcut.alt && !evt.altKey) || (!shortcut.ctrl && !evt.ctrlKey) || (!shortcut.shift && !evt.shiftKey)
            (shortcut.ctrl && evt.ctrlKey) || (shortcut.alt && evt.altKey) || (shortcut.shift && evt.shiftKey)) {
            if (shortcut.buttonID) {
                document.getElementById(shortcut.buttonID).click();
            } else if (shortcut.func) {
                shortcut.func();
            }
            return false;
        }
    }
});
