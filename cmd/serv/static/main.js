'use strict';

const baseURL = window.location.protocol + '//' + window.location.host;// + window.location.pathname.replace(/\/$/g,"");

let waveform;

async function getAudioBlob(payload) {

    let url = baseURL + "/echo_json";
    
    let xhr = new XMLHttpRequest();
    xhr.open("POST", url, true);
    xhr.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');   
    xhr.onloadend = function(data) {
	console.log("data", data);
    };    
    xhr.send(JSON.stringify(payload));

    
    fetch(url,
    	  {
    	      method: "POST",
    	      headers: {
    		  'Accept': 'application/json, text/plain, */*',
    		  'Content-Type': 'application/json'
    	      },
    	      body: JSON.stringify(payload)
	  })
	.then(function(res){ return res.json(); })
 	.then(function(data){ alert( JSON.stringify( data ) ) })
    	.catch (function (error) {
    	    console.log('Request failed', error);
    	});    
	      

    // let BLOB = [];
    // waveform.loadBlob(BLOB, [
    //         { start: 500, end: 1890 },
    //         { start: 2440, end: 4210 },
    //         { start: 4960, end: 8000 },
    //     ]);
    
}

function loadAudio(url, chunk) {
    waveform.loadAudio(url, [chunk]);
}

document.getElementById("play-all").addEventListener("click", function() {
    waveform.play(0.0);
});
document.getElementById("play-label").addEventListener("click", function() {
    waveform.playRegionIndex(0);
});
document.getElementById("play-left").addEventListener("click", function() {
    waveform.playLeftOfRegionIndex(0);
});
document.getElementById("play-right").addEventListener("click", function() {
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

    // let payload = {
    // 	url: baseURL + '/audio/a_pause.wav',
    // 	chunk: { start: 500, end: 1480 },
    // };

    let payload = {
	url: baseURL + '/audio/three_sentences.wav',
	chunk: { start: 500, end: 1890 },
    };
    
    waveform = new Waveform(options);

    // getAudioBlob(payload);

    loadAudio(payload.url, payload.chunk);
    

    
});
