"use strict";

// using http://wavesurfer-js.org with modules: regions, timeline

const
deleteKeyCode = 46,
enterKeyCode = 13,
spaceKeyCode = 32,
rightArrowKeyCode = 39,
leftArrowKeyCode = 37,
endKeyCode = 35,
homeKeyCode = 36
;

class Waveform {

    constructor(options) {
	this.options = options;
	console.log("waveform constructor called with options", options);

	var wsPlugins = [
	    WaveSurfer.regions.create({
		// dragSelection: {
		//     slop: 5
		// }
	    }),
	];

	if (this.options.timelineElementID)
	    wsPlugins.push(WaveSurfer.timeline.create({
		container: '#' + this.options.timelineElementID
	    }));

	if (this.options.spectrogramElementID)
	    wsPlugins.push(WaveSurfer.spectrogram.create({
		container: '#' + this.options.spectrogramElementID,
		labels: true,
	    }));


	if (!this.options.autoplayFunc)
	    this.options.autoplayFunc = function () { return false; }

	this.defaultRegionBackground = 'hsla(200, 50%, 70%, 0.4)';
	this.selectedRegionBackground = this.defaultRegionBackground; //'hsla(120, 100%, 75%, 0.3)';
	var wsOptions = {
	    container: '#' + this.options.waveformElementID,
	    waveColor: 'purple',
	    progressColor: 'purple',
	    loaderColor: 'purple',
	    autoCenter: true,
	    barHeight: 3,
	    plugins: wsPlugins,
	    normalise: true,
	};


	this.wavesurfer = WaveSurfer.create(wsOptions);

	if (this.options.navigationElementID) {
	    document.getElementById(this.options.navigationElementID).innerHTML = `<span class='btn noborder' id='waveform-skiptofirst'>&#x23EE;</span>
		    <span class='btn noborder' id='waveform-skipback'>&#x23EA;</span>
		    <span class='btn noborder' id='waveform-playpause'>&#x23EF;</span>
		    <span class='btn noborder' id='waveform-skipforward'>&#x23E9;</span>
		    <span class='btn noborder' id='waveform-skiptolast'>&#x23ED;</span>
		</span>`;
	}
	if (this.options.zoomElementID) {
	    document.getElementById(this.options.zoomElementID).innerHTML = `<span class="slidecontainer" style="vertical-align: middle; display:inline">waveform zoom
		<input id="waveform-zoom-input" title="Waveform zoom" style="vertical-align: middle; display:inline" type="range" min="20" max="1000" value="0" class="slider">
	</span>`;
	}

	let main = this;

	this.wavesurfer.on("audioprocess", function (evt) {
	    main.debug("audioprocess", evt);
	});

	this.wavesurfer.on("region-created", function (region) {
	    region.color = main.defaultRegionBackground;
	    region.element.addEventListener("click", async function (evt) {
		main.debug("click", evt);
		main.logEvent(evt);
		//await LIB.sleep(5); // TODO
		main.setSelectedRegion(region, evt.ctrlKey || main.options.autoplayFunc());
	    });
	    region.element.title = main.floatWithDecimals(region.start, 2) + " - " + main.floatWithDecimals(region.end, 2);
	});

	this.wavesurfer.on("region-updated", async function (region) {
	    //console.log("region-updated", region.element);
	    if (region.element.classList.contains("selected"))
		main.setSelectedRegion(region);
	    // region.element.addEventListener("contextmenu", function (evt) {
	    //     console.log("rightclick", evt);
	    //     evt.preventDefault();
	    //     return false;
	    // });
	    region.element.title = main.floatWithDecimals(region.start, 2) + " - " + main.floatWithDecimals(region.end, 2);
	});

	this.wavesurfer.on("error", function (evt) {
	    throw evt;
	});

	this.wavesurfer.on("ready", function () {
	    console.log("wavesurfer ready");
	    let wave = document.getElementById(this.options.waveformElementID).getElementsByTagName("wave")[0];
	    wave.style["height"] = "178px";
	});

	if (this.options.zoomElementID) {
	    document.getElementById('waveform-zoom-input').addEventListener("input", function (evt) {
		let value = evt.target.value;
		let selected = main.getSelectedRegion();
		main.wavesurfer.zoom(Number(value));
		if (selected)
		    main.setSelectedRegion(selected, false);
	    });
	}

	if (this.options.navigationElementID) {
	    document.getElementById("waveform-playpause").addEventListener("click", function (evt) {
		main.logEvent(evt);
		if (main.wavesurfer.isPlaying())
		    main.wavesurfer.pause();
		else
		    main.wavesurfer.play();
	    });

	    document.getElementById("waveform-skipforward").addEventListener("click", function (evt) {
		main.logEvent(evt);
		main.selectNextRegion();
	    });


	    document.getElementById("waveform-skipback").addEventListener("click", function (evt) {
		main.logEvent(evt);
		main.selectPrevRegion();
	    });

	    document.getElementById("waveform-skiptolast").addEventListener("click", function (evt) {
		main.logEvent(evt);
		let regions = main.listRegions();
		if (regions.length > 0)
		    main.setSelectedRegion(regions[regions.length - 1]);
	    });


	    document.getElementById("waveform-skiptofirst").addEventListener("click", function (evt) {
		main.logEvent(evt);
		main.setSelectedIndex(0);
	    });
	}


	if (this.options.navigationElementID) {
	    document.addEventListener("keydown", function (evt) {
		main.logEvent(evt);

		if (evt.keyCode === deleteKeyCode) {
		    let regions = main.listRegions();
		    for (let id in regions) {
			let region = regions[id];
			if (region.element.classList.contains("selected")) {
			    region.remove();
			}
		    }
		} else if (evt.keyCode === rightArrowKeyCode && evt.ctrlKey) {
		    document.getElementById("waveform-skipforward").click();
		} else if (evt.keyCode === leftArrowKeyCode && evt.ctrlKey) {
		    document.getElementById("waveform-skipback").click();
		} else if (evt.keyCode === homeKeyCode && evt.ctrlKey) {
		    document.getElementById("waveform-skiptofirst").click();
		} else if (evt.keyCode === endKeyCode && evt.ctrlKey) {
		    document.getElementById("waveform-skiptolast").click();
		} else if (evt.keyCode === spaceKeyCode) {
		    if (main.wavesurfer.isPlaying())
			main.wavesurfer.pause();
		    else
			main.wavesurfer.play();
		}
		return true;
	    });
	}

	let waveformResizeObserver = new ResizeObserver(function (source) {
	    let waveform = source[0].target;
	    let wave = waveform.getElementsByTagName("wave")[0];
	    if (waveform.style.height)
		wave.style.height = waveform.style.height;
	});
	waveformResizeObserver.observe(document.querySelector("#" + this.options.waveformElementID));

	console.log("waveform ready");
    }

    play(start, end) {
	this.wavesurfer.play(start, end);
    }

    playRegionIndex(index) {
	let regions = this.listRegions();
	for (let i = 0; i < regions.length; i++) {
	    if (i === index) {
		regions[i].play();
		break;
	    }
	}
    }

    playLeftOfRegionIndex(index) {
	let regions = this.listRegions();
	for (let i = 0; i < regions.length; i++) {
	    if (i === index) {
		this.wavesurfer.play(0, regions[i].start);
		break;
	    }
	}
    }

    playRightOfRegionIndex(index) {
	let regions = this.listRegions();
	for (let i = 0; i < regions.length; i++) {
	    if (i === index) {
		this.wavesurfer.play(regions[i].end);
		break;
	    }
	}
    }

    loadAudioBlob(blob, timeChunks) {
	console.log("waveform loadBlob", timeChunks);
	this.wavesurfer.regions.clear();
	this.wavesurfer.loadBlob(blob);
	this.loadChunks(timeChunks);
    }

    loadAudioURL(audioFile, timeChunks) {
	console.log("waveform loadAudio", audioFile, timeChunks);
	this.wavesurfer.load(audioFile);
	this.loadChunks(timeChunks);
    }

    loadChunks(chunks, clearBefore) {
	console.log("loadChunks", chunks);
	if (clearBefore)
	    this.wavesurfer.clearRegions();
	for (let i in chunks) {
	    let chunk = chunks[i];
	    this.wavesurfer.addRegion({
		start: chunk.start / 1000.0,
		end: chunk.end / 1000.0,
		color: this.defaultRegionBackground,
	    });
	}
	this.setSelectedIndex(0, false);
    }

    getRegion(index) {
	let regions = this.listRegions();
	if (regions.length > index) {
	    let region = regions[index];
	    return {start: this.roundToInt(region.start*1000), end: this.roundToInt(region.end*1000) };
	}	
    }
    
    playRegion(region) {
	region.play();
    }

    updateRegion(index, start, end) {
	let regions = this.listRegions();
	if (regions.length > index) {
	    let region = regions[index];
	    region.start = start / 1000.0;
	    region.end = end / 1000.0;
	    region.update({start: region.start, end: region.end});
	}
    }
    
    moveStartForRegionIndex(index, moveAmountInMilliseconds) {
	let regions = this.listRegions();
	if (regions.length > index) {
	    let region = regions[index];
	    let newStart = region.start + (moveAmountInMilliseconds/1000.0)
	    if (newStart < 0)
		newStart = 0.0;
	    if (newStart >= region.end)
		newStart = region.end;
	    region.start = newStart;
	    region.update({end: region.end, start: newStart});
	}
    }

    moveEndForRegionIndex(index, moveAmountInMilliseconds) {
	let regions = this.listRegions();
	if (regions.length > index) {
	    let region = regions[index];
	    let newEnd = region.end + (moveAmountInMilliseconds/1000.0)
	    if (newEnd < this.length)
		newEnd = this.length-0.5; // TODO
	    if (newEnd <= region.start)
		newEnd = region.start;
	    region.end = newEnd;
	    region.update({start: region.start, end: newEnd});
	}
    }
    
    getSelectedRegion() {
	this.debug("getSelectedRegion");
	let regions = this.listRegions();
	for (let id in regions) {
	    let region = regions[id];
	    if (region.element.classList.contains("selected")) {
		//console.log("getSelectedRegion", region);
		return region;
	    }
	}
    }

    unselect(regionElements) {
	for (let j in regionElements) {
	    let e = regionElements[j];
	    let id = e.getAttribute("data-id")
	    if (e.localName === "region") {
		e.classList.remove("selected");
		e.style["background-color"] = this.defaultRegionBackground;
	    }
	}
    }

    setSelectedRegion(region, playMode) {
	this.debug("setSelectedRegion", region);
	region.element.classList.add("selected");
	region.element.style["background-color"] = this.selectedRegionBackground; //"rgba(0,255,0,0.3)";
	this.unselect(LIB.siblings(region.element, false));
	if (playMode === true)
	    this.playRegion(region);
	else if (this.options.autoplayFunc() && playMode !== false)
	    this.playRegion(region);
    }

    setSelectedIndex(index, playMode) {
	this.debug("setSelectedIndex", index, playMode);
	let regions = this.listRegions();
	if (regions.length > index)
	    this.setSelectedRegion(regions[index], playMode);
    }

    // list regions sorted by start time
    listRegions() {
	let regions = Object.values(this.wavesurfer.regions.list);
	regions.sort(function (a, b) { return a.start - b.start });
	return regions;
    }

    selectPrevRegion() {
	let regions = this.listRegions();
	let lastSelected;
	for (let i = regions.length - 1; i >= 0; i--) {
	    let region = regions[i];
	    this.debug("selectPrevRegion", regions.length, i, region, lastSelected);
	    if (lastSelected) {
		this.setSelectedRegion(region);
		break;
	    }
	    if (region.element.classList.contains("selected"))
		lastSelected = region;
	}
    }

    selectNextRegion() {
	let regions = this.listRegions();
	let lastSelected;
	for (let i = 0; i < regions.length; i++) {
	    let region = regions[i];
	    this.debug("selectNextRegion", region, lastSelected);
	    if (lastSelected) {
		this.setSelectedRegion(region);
		break;
	    }
	    if (region.element.classList.contains("selected"))
		lastSelected = region;
	}
    }

    clear() {	
	let wsPane = document.getElementById("waveform-pane");
	let h = wsPane.offsetHeight;
	let w = wsPane.offsetWidth;
	this.wavesurfer.regions.clear();
	// this.wavesurfer.timeline.destroy();
	// this.wavesurfer.spectrogram.destroy();
	this.wavesurfer.empty();
	//this.wavesurfer.cursorColor = "transparent";
	// wsPane.style["height"] = h + "px";
	// wsPane.style["width"] = w + "px";
    }
    
    // LIB

    debug(msg) {
	if (this.options.debug) console.log("waveform debug", msg);
    }

    logEvent(evt) {
	this.debug("LOG EVENT | type: " + evt.type + ", element id:" + evt.target.id, evt);
    }

    floatWithDecimals(f0, decimalCount) {
	let f = Number((f0).toFixed(decimalCount));
	let res = f + "";
	if (!res.includes("."))
	    return res + ".00";
	if (res.match(/[.][0-9]$/g))
	    return res + "0";
	// if (res.match(/[.][0-9][0-9]$/g))
	//     return res + "0";
	else return res;
    }

    roundToInt(f) {
	return Number((f).toFixed(0));
    }

}

