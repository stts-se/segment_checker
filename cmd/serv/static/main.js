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
            console.log("got", json)

            // https://stackoverflow.com/questions/16245767/creating-a-blob-from-a-base64-string-in-javascript#16245768
            let byteCharacters = atob(json.audio);

            let byteNumbers = new Array(byteCharacters.length);
            for (let i = 0; i < byteCharacters.length; i++) {
                byteNumbers[i] = byteCharacters.charCodeAt(i);
            }
            let byteArray = new Uint8Array(byteNumbers);

            let blob = new Blob([byteArray], { 'type': json.file_type });
            waveform.loadAudioBlob(blob, payload.chunk);
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

    let options = {
        waveformElementID: "waveform",
        timelineElementID: "waveform-timeline",
        spectrogramElementID: "waveform-spectrogram",
        // zoomElementID: "waveform-zoom",
        // navigationElementID: "waveform-navigation",
        debug: false,
    };

    let payload = {
        url: baseURL + '/audio/a_pause.wav',
        chunk: { start: 500, end: 1480 },
    };

    // let payload = {
    // 	url: baseURL + '/audio/three_sentences.wav',
    // 	chunk: { start: 500, end: 1890 },
    // };

    waveform = new Waveform(options);

    getAudioBlob(payload);

    loadAudioURL(payload.url, payload.chunk);



});
