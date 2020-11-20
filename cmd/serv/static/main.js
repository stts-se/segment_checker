'use strict';

let waveform;

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

    waveform = new Waveform(options);

    // let BLOB = [];
    // waveform.loadBlob(BLOB, [
    //         { start: 500, end: 1890 },
    //         { start: 2440, end: 4210 },
    //         { start: 4960, end: 8000 },
    //     ]);
    

});
