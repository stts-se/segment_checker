'use strict';

const baseURL = window.location.protocol + '//' + window.location.host + window.location.pathname.replace(/\/$/g, "");

const boundaryMovementShort = 5;
const boundaryMovementLong = 100;

const contextParamEnabled = true;
let context = 1000;

let enabled = false;
let waveform;
let cachedSegment;

function logMessage(msg) {
    let div = document.createElement("div");
    div.innerText = msg;
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
        //document.getElementById("next"),
    ];
    if (enable) {
        for (let i = 0; i < buttons.length; i++) {
            let btn = buttons[i];
            btn.classList.remove("disabled");
            btn.removeAttribute("disabled");
            btn.disabled = false;
        }
        document.getElementById("comment").removeAttribute("readonly");
    } else {
        for (let i = 0; i < buttons.length; i++) {
            let btn = buttons[i];
            btn.classList.add("disabled");
            btn.disabled = true;
        }
        document.getElementById("comment").setAttribute("readonly", "readonly");
    }
}

function loadSegmentFromFile(sourceSegmentFile) {
    let url = baseURL + `/load?sourcefile=${sourceSegmentFile}&context=${context}&username=` + document.getElementById("username").innerText;

    fetch(url,
        {
            method: "GET",
            headers: {
                'Accept': 'application/json, text/plain, */*',
                'Content-Type': 'application/json'
            },
            //body: JSON.stringify(payload)
        })
        .then(function (res) { return res.json(); })
        .then(function (data) {
            console.log(url, "=>", JSON.stringify(data));

            if (data.error) {
                logMessage('server error: Got error from GET to ' + url + ": " + data.error);
                setEnabled(false);
                return;
            }

            if (data.message_type === "audio_chunk") {
                let res = JSON.parse(data.payload);

                // https://stackoverflow.com/questions/16245767/creating-a-blob-from-a-base64-string-in-javascript#16245768
                let byteCharacters = atob(res.audio);
                let byteNumbers = new Array(byteCharacters.length);
                for (let i = 0; i < byteCharacters.length; i++) {
                    byteNumbers[i] = byteCharacters.charCodeAt(i);
                }
                let byteArray = new Uint8Array(byteNumbers);

                cachedSegment = {
                    url: res.url,
                    uuid: res.uuid,
                    segment_type: res.segment_type,
                    offset: res.offset,
                    chunk: res.chunk,
                }

                console.log(JSON.stringify(cachedSegment));
                let blob = new Blob([byteArray], { 'type': res.file_type });
                loadAudioBlob(blob, res.chunk);
                setEnabled(true);
                logMessage("client: Loaded segment " + res.uuid + " from server");

            } else {
                logMessage('Unknown message type from server: ' + data.message_type);
            }
        })
        .catch(function (error) {
            logMessage('Request failed: ' + error);
            setEnabled(false);
        });

}

function loadAudioBlob(url, chunk) {
    waveform.loadAudioBlob(url, [chunk]);
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

document.getElementById("save-badsample").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        save({ status: "skip", label: "bad sample", callback: loadStats });
});
document.getElementById("save-skip").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        save({ status: "skip", callback: loadStats });
});
document.getElementById("save-ok").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        save({ status: "ok", callback: loadStats });
});
document.getElementById("save-badsample-next").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        save({ status: "skip", label: "bad sample", callback: next });
});
document.getElementById("save-skip-next").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        save({ status: "skip", callback: next });
});
document.getElementById("save-ok-next").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        save({ status: "ok", callback: next });
});
document.getElementById("next").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        next();
});
document.getElementById("reset").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        waveform.updateRegion(0, cachedSegment.chunk.start, cachedSegment.chunk.end);
        document.getElementById("comment").value = "";
    }
});
document.getElementById("quit").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        releaseCurrentSegment();
        waveform.clear();
        document.getElementById("comment").value = "";
        setEnabled(false);
        loadStats();
    }
});

document.getElementById("release-all").addEventListener("click", function (evt) {
    if (!evt.target.disabled) {
        releaseAll();
        waveform.clear();
        document.getElementById("comment").value = "";
        setEnabled(false);
        loadStats();
    }
});

document.getElementById("load_stats").addEventListener("click", function (evt) {
    if (!evt.target.disabled)
        loadStats();
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

function loadStats() {
    let url = baseURL + "/stats/";

    fetch(url,
        {
            method: "GET",
            headers: {
                'Accept': 'application/json, text/plain, */*',
                'Content-Type': 'application/json'
            },
        })
        .then(function (res) { return res.json(); })
        .then(function (data) {
            console.log(url, "=>", JSON.stringify(data));

            if (data.error) {
                logMessage('server error: Got error from GET to ' + url + ": " + data.error);
                setEnabled(false);
                return;
            } else if (data.info) {
                logMessage('server: ' + data.info);
                return;

            } else if (data.message_type === "stats") {
                let ele = document.getElementById("stats");
                ele.innerText = "";
                const parsedJSON = JSON.parse(data.payload);
                let keys = Object.keys(parsedJSON)
                keys.sort();
                keys.forEach(function (key) {
                    let tr = document.createElement("tr");
                    let td1 = document.createElement("td");
                    let td2 = document.createElement("td");
                    td2.style["text-align"] = "right";
                    td1.innerHTML = key;
                    td2.innerHTML = parsedJSON[key];
                    tr.appendChild(td1);
                    tr.appendChild(td2);
                    ele.appendChild(tr);
                });

            } else {
                logMessage('Unknown message type from server: ' + data.message_type);
            }
        })
        .catch(function (error) {
            logMessage('Request failed: ' + error);
            setEnabled(false);
        });
}

function releaseCurrentSegment() {
    console.log("releaseCurrentSegment called")
    if (cachedSegment === undefined || cachedSegment === null)
        return;

    let url = baseURL + "/release/" + cachedSegment.uuid + "/" + document.getElementById("username").innerText;

    fetch(url,
        {
            method: "GET",
            headers: {
                'Accept': 'application/json, text/plain, */*',
                'Content-Type': 'application/json'
            },
        })
        .then(function (res) { return res.json(); })
        .then(function (data) {
            console.log(url, "=>", JSON.stringify(data));

            if (data.error) {
                logMessage('server error: Got error from GET to ' + url + ": " + data.error);
                setEnabled(false);
                return;
            } else if (data.info) {
                logMessage('server: ' + data.info);
                cachedSegment = null;
                setEnabled(false);
                return;

            } else {
                logMessage('Unknown message type from server: ' + data.message_type);
            }
        })
        .catch(function (error) {
            logMessage('Request failed: ' + error);
            setEnabled(false);
        });

}

function releaseAll() {
    let url = baseURL + "/releaseall/" + document.getElementById("username").innerText;

    fetch(url,
        {
            method: "GET",
            headers: {
                'Accept': 'application/json, text/plain, */*',
                'Content-Type': 'application/json'
            },
        })
        .then(function (res) { return res.json(); })
        .then(function (data) {
            console.log(url, "=>", JSON.stringify(data));

            if (data.error) {
                logMessage('server error: Got error from GET to ' + url + ": " + data.error);
                setEnabled(false);
                return;
            } else if (data.info) {
                logMessage('server: ' + data.info);
                return;

            } else {
                logMessage('Unknown message type from server: ' + data.message_type);
            }
        })
        .catch(function (error) {
            logMessage('Request failed: ' + error);
            setEnabled(false);
        });

}

document.getElementById("clear_messages").addEventListener("click", function (evt) {
    document.getElementById("messages").innerHTML = "";
});


function next() {
    console.log("next called")
    releaseCurrentSegment();

    let url = baseURL + "/next/?username=" + document.getElementById("username").innerText;
    url = url + "&context=" + context;
    if (cachedSegment && cachedSegment !== null)
        url = url + "&currid=" + cachedSegment.uuid;
    console.log("next URL", url);

    fetch(url,
        {
            method: "GET",
            headers: {
                'Accept': 'application/json, text/plain, */*',
                'Content-Type': 'application/json'
            },
        })
        .then(function (res) { return res.json(); })
        .then(function (data) {
            console.log(url, "=>", JSON.stringify(data));

            if (data.error) {
                logMessage('server error: Got error from GET to ' + url + ": " + data.error);
                setEnabled(false);
                return;
            } else if (data.message_type === "audio_chunk") {
                let res = JSON.parse(data.payload);

                // https://stackoverflow.com/questions/16245767/creating-a-blob-from-a-base64-string-in-javascript#16245768
                let byteCharacters = atob(res.audio);
                let byteNumbers = new Array(byteCharacters.length);
                for (let i = 0; i < byteCharacters.length; i++) {
                    byteNumbers[i] = byteCharacters.charCodeAt(i);
                }
                let byteArray = new Uint8Array(byteNumbers);

                cachedSegment = {
                    url: res.url,
                    uuid: res.uuid,
                    segment_type: res.segment_type,
                    offset: res.offset,
                    chunk: res.chunk,
                }

                console.log(JSON.stringify(cachedSegment));
                let blob = new Blob([byteArray], { 'type': res.file_type });
                loadAudioBlob(blob, res.chunk);
                setEnabled(true);
                logMessage("client: Loaded segment " + res.uuid + " from server");
                loadStats();

            } else if (data.info) {
                logMessage('server: ' + data.info);
                return;

            } else {
                logMessage('Unknown message type from server: ' + data.message_type);
            }
        })
        .catch(function (error) {
            console.log(error);
            logMessage('Request failed: ' + error);
            setEnabled(false);
        });

}

function save(options) {
    console.log("save called with options", options);
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
    let status = {
        source: user,
        name: options.status,
        timestamp: new Date().toUTCString(),
    }
    let labels = [];
    if (options.label) {
        labels.push(options.label);
    }
    let region = waveform.getRegion(0)
    let payload = {
        uuid: cachedSegment.uuid,
        url: cachedSegment.url,
        segment_type: cachedSegment.segment_type,
        chunk: {
            start: region.start + cachedSegment.offset,
            end: region.end + cachedSegment.offset,
        },
        status: status,
        labels: labels,
        comment: document.getElementById("comment").value,
    }
    console.log("payload", JSON.stringify(payload));

    let url = baseURL + "/save/";

    fetch(url,
        {
            method: "POST",
            headers: {
                'Accept': 'application/json, text/plain, */*',
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(payload)
        })
        .then(function (res) { return res.json(); })
        .then(function (data) {
            console.log(url, "=>", JSON.stringify(data));

            if (data.error) {
                logMessage('server error: Got error from POST to ' + url + ": " + data.error);
                return;
            }

            if (data.info) {
                logMessage('server: ' + data.info);
                if (options.callback)
                    options.callback();

                return;
            }

            else {
                logMessage('Unknown message type from server: ' + data.message_type);
            }
        })
        .catch(function (error) {
            logMessage('Request failed: ' + error);
        });

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
    if (contextParamEnabled) {
        if (params.get('context'))
            context = params.get('context');
        document.getElementById("context").innerText = `${context} ms`;
        document.getElementById("context-view").classList.remove("hidden");
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

    next();

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
    'shift ArrowLeft': { funcDesc: `Move right boundary ${boundaryMovementShort} ms to the left`, buttonID: 'move-right2left-short' },
    'shift ArrowRight': { funcDesc: `Move right boundary ${boundaryMovementShort} ms to the right`, buttonID: 'move-right2right-short' },
    'ctrl ArrowUp': { funcDesc: `Move left boundary ${boundaryMovementLong} ms to the left`, buttonID: 'move-left2left-long' },
    'ctrl ArrowDown': { funcDesc: `Move left boundary ${boundaryMovementLong} ms to the right`, buttonID: 'move-left2right-long' },
    'shift ArrowUp': { funcDesc: `Move right boundary ${boundaryMovementLong} ms to the left`, buttonID: 'move-right2left-long' },
    'shift ArrowDown': { funcDesc: `Move right boundary ${boundaryMovementLong} ms to the right`, buttonID: 'move-right2right-long' },
    'ArrowLeft': { tooltip: 'left', funcDesc: 'Play left context', buttonID: 'play-left' },
    'ArrowRight': { tooltip: 'right', funcDesc: 'Play right context', buttonID: 'play-right' },
    'ArrowDown': { tooltip: 'down', funcDesc: 'Play all audio', buttonID: 'play-all' },
    ' ': { tooltip: 'space', funcDesc: 'Play label', buttonID: 'play-label' },
    'ctrl  ': { buttonID: 'play-label' }, // hidden from shortcut view
    'shift  ': { buttonID: 'play-label' }, // hidden from shortcut view
    //'n': { tooltip: 'n', buttonID: 'next', funcDesc: "Get next segment" },
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
