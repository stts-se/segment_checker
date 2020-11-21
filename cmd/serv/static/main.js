'use strict';

const baseURL = window.location.protocol + '//' + window.location.host;// + window.location.pathname.replace(/\/$/g,"");

let waveform;

async function getAudioBlob(payload) {

    let url = baseURL + "/extract_chunk/";

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
            let json = JSON.stringify(data)
            console.log("got", json);

            // https://stackoverflow.com/questions/16245767/creating-a-blob-from-a-base64-string-in-javascript#16245768
            let byteCharacters = atob(data.audio);

            let byteNumbers = new Array(byteCharacters.length);
            for (let i = 0; i < byteCharacters.length; i++) {
                byteNumbers[i] = byteCharacters.charCodeAt(i);
            }
            let byteArray = new Uint8Array(byteNumbers);

            let blob = new Blob([byteArray], { 'type': data.file_type });
            loadAudioBlob(blob, data.chunk);
        })
        .catch(function (error) {
            console.log('Request failed', error);
        });

}

function loadAudioBlob(url, chunk) {
    waveform.loadAudioBlob(url, [chunk]);
}

function loadAudioURL(url, chunk) {
    waveform.loadAudioURL(url, [chunk]);
}

document.getElementById("play-all").addEventListener("click", function () {
    waveform.play(0.0);
});
document.getElementById("play-label").addEventListener("click", function () {
    waveform.playRegionIndex(0);
});
document.getElementById("play-left").addEventListener("click", function () {
    waveform.playLeftOfRegionIndex(0);
});
document.getElementById("play-right").addEventListener("click", function () {
    waveform.playRightOfRegionIndex(0);
});

window.addEventListener("load", function () {

    let params = new URLSearchParams(window.location.search);
    if (params.get('autoplay') && params.get('autoplay').match(/^[t1]/g))
        document.getElementById("autoplay").checked = true;

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

    // let payload = {
    //     url: baseURL + '/audio/a_pause.wav',
    //     chunk: { start: 500, end: 1480 },
    // };

    let payload = {
        url: baseURL + '/audio/three_sentences.wav',
        segment_type: "silence",
        left_context: 1000,
        right_context: 1000,
        chunk: { start: 1660, end: 2640 },
    };

    waveform = new Waveform(options);

    getAudioBlob(payload);

    //loadAudioURL(payload.url, payload.chunk);

});

function loadKeyboardShortcuts() {
    let ele = document.getElementById("shortcuts");
    ele.innerHTML = "";
    Object.keys(shortcuts).forEach(function (key) {
        let id = shortcuts[key].buttonID;
        let desc = shortcuts[key].desc;
        if (id && desc) {
            document.getElementById(id).title = desc;
        }
        if (desc && shortcuts[key].funcDesc) {
            let tr = document.createElement("tr");
            let td1 = document.createElement("td");
            let td2 = document.createElement("td");
            td1.innerHTML = desc;
            td2.innerHTML = shortcuts[key].funcDesc;
            tr.appendChild(td1);
            tr.appendChild(td2);
            ele.appendChild(tr);
        }
    });
}

let shortcuts = {
    'ArrowLeft' : { desc: 'left', funcDesc: 'Play left context', buttonID: 'play-left'},
    'ctrl ArrowLeft' : { desc: 'ctrl left', funcDesc: 'Move left boundary to the left', func: function() { waveform.moveStartForRegionIndex(0, -5)}},
    'ctrl ArrowRight' : { desc: 'ctrl right', funcDesc: 'Move left boundary to the right', func: function() { waveform.moveStartForRegionIndex(0, 5)}},
    'shift ArrowLeft' : { desc: 'shift left', funcDesc: 'Move right boundary to the left', func: function() { waveform.moveEndForRegionIndex(0, -5)}},
    'shift ArrowRight' : { desc: 'shift right', funcDesc: 'Move right boundary to the right', func: function() { waveform.moveEndForRegionIndex(0, 5)}},
    'ArrowRight': { desc: 'right', funcDesc: 'Play right context', buttonID: 'play-right'},
    'ArrowDown': { desc: 'down', funcDesc: 'Play all audio', buttonID: 'play-all'},
    ' ': { desc: 'space', funcDesc: 'Play label', buttonID: 'play-label'},
    'ctrl  ': { buttonID: 'play-label'},
    'shift  ': { buttonID: 'play-label'},
};

window.addEventListener("keydown", function(evt) {
    //console.log(evt.key, evt.keyCode, evt.ctrlKey, evt.altKey);
    let key = evt.key;
    if (evt.altKey)
        key = "alt " + key;
    if (evt.ctrlKey)
        key = "ctrl " + key;
    if (evt.shiftKey)
        key = "shift " + key;
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
