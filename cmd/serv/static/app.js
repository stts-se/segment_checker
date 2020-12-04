'use strict';

const baseURL = window.location.protocol + '//' + window.location.host + window.location.pathname.replace(/\/$/g, "");
const wsBase = baseURL.replace(/^http/, "ws");  
const clientID = LIB.uuidv4();
let ws;

const boundaryMovementShort = 5;
const boundaryMovementLong = 100;

// context
const debugMode = true;
let context;

let enabled = false;
let waveform;
let cachedSegment;

let debugVar;

function logMessage(msg) {
    debugVar = msg;
    let div = document.createElement("div");
    div.innerText = msg;
    if (msg.toLowerCase().includes("error"))
	div.style["color"] = "red";
    messages.prepend(div);
}

function setEnabled(enable) {
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
        document.getElementById("comment").removeAttribute("readonly");
    } else {
        if (waveform)
            waveform.clear();
        for (let i = 0; i < buttons.length; i++) {
            let btn = buttons[i];
            if (btn) {
                btn.classList.add("disabled");
                btn.disabled = true;
            }
        }
        document.getElementById("comment").setAttribute("readonly", "readonly");
        document.getElementById("comment").value = "";
        document.getElementById("labels").innerText = "";
    }
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
    await waveform.loadAudioBlob(url, [chunk]);
    //autoplay();
}

function loadAudioURL(url, chunk) {
    waveform.loadAudioURL(url, [chunk]);
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
//         save({ status: "skip", label: "bad sample", callback: requestStats });
// });
// document.getElementById("save-skip").addEventListener("click", function (evt) {
//     if (!evt.target.disabled)
//         save({ status: "skip", callback: requestStats });
// });
// document.getElementById("save-ok").addEventListener("click", function (evt) {
//     if (!evt.target.disabled)
//         save({ status: "ok", callback: requestStats });
// });
document.getElementById("save-badsample-next").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        saveReleaseAndNext({ status: "skip", label: "bad sample", stepSize: 1 });
});
document.getElementById("save-skip-next").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        saveReleaseAndNext({ status: "skip", stepSize: 1 });
});
document.getElementById("save-ok-next").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        saveReleaseAndNext({ status: "ok", stepSize: 1 });
});

if (document.getElementById("next")) {
    document.getElementById("next").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveReleaseAndNext({ stepSize: 1 });
    });
}
if (document.getElementById("prev")) {
    document.getElementById("prev").addEventListener("click", function (evt) {
        if (!evt.target.disabled)
            saveReleaseAndNext({ stepSize: -1 });
    });
}

document.getElementById("reset").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.updateRegion(0, cachedSegment.chunk.start, cachedSegment.chunk.end);
        document.getElementById("comment").value = "";
        document.getElementById("labels").innerText = "";
        document.getElementById("current_status").innerText = "";
    }
});
document.getElementById("quit").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        releaseCurrentSegment();
        waveform.clear();
        document.getElementById("comment").value = "";
        document.getElementById("labels").innerText = "";
        document.getElementById("current_status").innerText = "";
        setEnabled(false);
        //requestStats();
    }
});

document.getElementById("release-all").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        releaseAll();
        waveform.clear();
        document.getElementById("comment").value = "";
        document.getElementById("labels").innerText = "";
        document.getElementById("current_status").innerText = "";
        setEnabled(false);
        //requestStats();
    }
});

document.getElementById("load_stats").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        requestStats();
});

document.getElementById("move-left2left-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(0, -boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-left2right-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(0, boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2left-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(0, -boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2right-short").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(0, boundaryMovementShort);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});

document.getElementById("move-left2left-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(0, -boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-left2right-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveStartForRegionIndex(0, boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2left-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(0, -boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});
document.getElementById("move-right2right-long").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.moveEndForRegionIndex(0, boundaryMovementLong);
        evt.preventDefault();
        evt.stopPropagation();
        return false;
    }
});

function displayAudioChunk(chunk) {
    // https://stackoverflow.com/questions/16245767/creating-a-blob-from-a-base64-string-in-javascript#16245768
    let byteCharacters = atob(chunk.audio);
    let byteNumbers = new Array(byteCharacters.length);
    for (let i = 0; i < byteCharacters.length; i++) {
        byteNumbers[i] = byteCharacters.charCodeAt(i);
    }
    let byteArray = new Uint8Array(byteNumbers);

    cachedSegment = chunk;
    cachedSegment.audio = null; // no need to cache the audio blob
    console.log("res => cachedSegment", JSON.stringify(cachedSegment));

    let blob = new Blob([byteArray], { 'type': chunk.file_type });
    loadAudioBlob(blob, chunk.chunk);
    document.getElementById("segment_id").innerText = chunk.index + " | " + chunk.uuid;
    let status = chunk.current_status.name;
    if (chunk.current_status.source)
        status += " (" + chunk.current_status.source + ")";
    if (chunk.current_status.timestamp)
        status += " " + chunk.current_status.timestamp;
    if (chunk.comment)
        document.getElementById("comment").value = chunk.comment;
    document.getElementById("current_status").innerText = status;
    if (chunk.labels && chunk.labels.length > 0)
        document.getElementById("labels").innerText = chunk.labels;
    else
        document.getElementById("labels").innerText = "none";
    setEnabled(true);
    logMessage("client: Loaded segment " + chunk.uuid + " from server");
    requestStats();
}

function displayStats(stats) {
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

}

function requestStats() {
    let request = {
	'client_id': clientID,
	'message_type': 'stats',
    };
    ws.send(JSON.stringify(request));
}


function releaseCurrentSegment() {
    console.log("releaseCurrentSegment called")
    if (cachedSegment === undefined || cachedSegment === null)
        return;

    let request = {
	'client_id': clientID,
	'message_type': 'release',
	'payload': JSON.stringify({
	    'uuid': cachedSegment.uuid,
	    'user_name': document.getElementById("username").innerText,
	}),
    };
    ws.send(JSON.stringify(request));
}

function releaseAll() {
    console.log("releaseAll called")
    let request = {
	'client_id': clientID,
	'message_type': 'release_all',
	'payload': JSON.stringify({
	    'user_name': document.getElementById("username").innerText,
	}),
    };
    ws.send(JSON.stringify(request));
}

document.getElementById("clear_messages").addEventListener("click", function (evt) {
    document.getElementById("messages").innerHTML = "";
});


function createQuery(stepSize) {
    let query = {
        step_size: stepSize,
        user_name: document.getElementById("username").innerText,
    }
    if (context)
        query.context = context;
    if (cachedSegment && cachedSegment !== null)
        query.curr_id = cachedSegment.uuid;

    // search for status
    if (document.getElementById("searchmode-ok").checked)
        query.request_status = ["ok"];
    else if (document.getElementById("searchmode-unchecked").checked)
        query.request_status = ["unchecked"];
    else if (document.getElementById("searchmode-checked").checked)
        query.request_status = ["ok", "skip"];
    return query;
}

function next(stepSize) {
    console.log("next called")
    releaseCurrentSegment();

    let request = {
	'client_id': clientID,
	'message_type': 'next',
	'payload': JSON.stringify(createQuery(stepSize)),
    };
    ws.send(JSON.stringify(request));
}


function saveReleaseAndNext(options) {
    console.log("saveReleaseAndNext called with options", options);
    let user = document.getElementById("username").innerText;
    if ((!user) || user === "") {
        let msg = "Username unset -- cannot save!";
        alert(msg);
        logMessage(msg);
        return;
    }
    if (!cachedSegment.uuid) {
        let msg = "No cached segment -- cannot save!";
        alert(msg);
        logMessage(msg);
        return;
    }
    let annotation = {};
    if (options.status) { // create annotation to save
        let status = {
            source: user,
            name: options.status,
            timestamp: new Date().toUTCString(),
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
            uuid: cachedSegment.uuid,
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
        }
    }
    let query = createQuery(options.stepSize);
    
    let payload = {
        annotation: annotation,
        release: { user_name: user, uuid: cachedSegment.uuid },
        query: query,
    };
    console.log("payload", JSON.stringify(payload));

    let request = {
	'client_id': clientID,
	'message_type': 'savereleaseandnext',
	'payload': JSON.stringify(payload),
    };
    ws.send(JSON.stringify(request));

}


window.addEventListener("load", function () {

    setEnabled(false);

    let params = new URLSearchParams(window.location.search);
    if (params.get('username'))
        document.getElementById("username").innerText = params.get('username');
    else {
        let msg = "Username unset! Reload page with URL param ?username=NAME";
        alert(msg);
        logMessage(msg);
        return;
    }
    if (debugMode) {
        if (params.get('context')) {
            context = params.get('context');
            document.getElementById("context").innerText = `${context} ms`;
            document.getElementById("context-view").classList.remove("hidden");
        }
        if (params.get('searchmode')) {
            let mode = params.get('searchmode');
            if (document.getElementById(`searchmode-${mode}`))
                document.getElementById(`searchmode-${mode}`).checked = true;
        }
    }

    let url = wsBase + "/ws/"+clientID;
    ws = new WebSocket(url);
    ws.onopen = function() {
	logMessage("Websocket open");
	next(1);	
    }
    ws.onmessage = function(evt) {
	let resp = JSON.parse(evt.data);
	//console.log(resp);
	if (resp.error) {
	    logMessage("server error: " + resp.error);
	    return;
	}
	if (resp.info) {
	    logMessage("server info: " + resp.info);
	    return;
	}
	if (resp.message_type === "stats") 
	    displayStats(JSON.parse(resp.payload));
	else if (resp.message_type === "audio_chunk") {
	    displayAudioChunk(JSON.parse(resp.payload));
	}
 	else if ( resp.message_type != "keep_alive" ) 
	    console.log("unknown message from server", resp);
    }

    console.log("main window loaded");

    loadKeyboardShortcuts();

    let options = {
        waveformElementID: "waveform",
        timelineElementID: "waveform-timeline",
        spectrogramElementID: "waveform-spectrogram",
        // zoomElementID: "waveform-zoom",
        // navigationElementID: "waveform-navigation",
        debug: false,
    };

    waveform = new Waveform(options);

    //loadSegmentFromFile('tillstud_demo_2_Niclas_Tal_1_2020-08-24_141655_b35aa260_00021.json');

});

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
            if (ele)
                ele.title = "key: " + tooltip;
            else
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
    'ctrl ArrowLeft': { funcDesc: `Move left boundary ${boundaryMovementShort} ms to the left`, buttonID: 'move-left2left-short' },
    'ctrl ArrowRight': { funcDesc: `Move left boundary ${boundaryMovementShort} ms to the right`, buttonID: 'move-left2right-short' },
    'ctrl ArrowUp': { funcDesc: `Move left boundary ${boundaryMovementLong} ms to the left`, buttonID: 'move-left2left-long' },
    'ctrl ArrowDown': { funcDesc: `Move left boundary ${boundaryMovementLong} ms to the right`, buttonID: 'move-left2right-long' },
    'shift ArrowLeft': { funcDesc: `Move right boundary ${boundaryMovementShort} ms to the left`, buttonID: 'move-right2left-short' },
    'shift ArrowRight': { funcDesc: `Move right boundary ${boundaryMovementShort} ms to the right`, buttonID: 'move-right2right-short' },
    'shift ArrowUp': { funcDesc: `Move right boundary ${boundaryMovementLong} ms to the left`, buttonID: 'move-right2left-long' },
    'shift ArrowDown': { funcDesc: `Move right boundary ${boundaryMovementLong} ms to the right`, buttonID: 'move-right2right-long' },
    'ArrowLeft': { tooltip: 'left', funcDesc: 'Play left context', buttonID: 'play-left' },
    'ArrowRight': { tooltip: 'right', funcDesc: 'Play right context', buttonID: 'play-right' },
    'ArrowDown': { tooltip: 'down', funcDesc: 'Play all audio', buttonID: 'play-all' },
    ' ': { tooltip: 'space', funcDesc: 'Play label', buttonID: 'play-label' },
    'ctrl  ': { buttonID: 'play-label' }, // hidden from shortcut view
    'shift  ': { buttonID: 'play-label' }, // hidden from shortcut view
    'n': { tooltip: 'n', buttonID: 'next', funcDesc: "Get next segment" },
    'p': { tooltip: 'p', buttonID: 'prev', funcDesc: "Get previous segment" },
    'o': { buttonID: 'save-ok-next', funcDesc: "Save as ok and get next" },
    's': { buttonID: 'save-skip-next', funcDesc: "Save as skip and get next" },
    'b': { buttonID: 'save-badsample-next', funcDesc: "Save as skip with label 'bad sample', and get next" },
};

window.addEventListener("keydown", function (evt) {
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
