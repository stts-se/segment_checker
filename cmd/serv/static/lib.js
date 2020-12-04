"use strict";

let LIB = {};

LIB.siblings = function (elem, includeSelf) {
    let siblings = [];

    // if no parent, return empty list
    if (!elem.parentNode) {
        return siblings;
    }

    // first child of the parent node
    let sibling = elem.parentNode.firstElementChild;

    do {
        if (includeSelf || sibling !== elem)
            siblings.push(sibling);
    } while (sibling = sibling.nextElementSibling);

    return siblings;
}

LIB.uuidv4 = function () {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
        var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

LIB.sleep = function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

LIB.timestampYYYYMMDDHHMMSS = function() {
    let date = new Date();
    let yyyy = date.getFullYear();
    let mo = date.getMonth()+1;
    if (mo < 10) mo = "0" + mo;
    let da = date.getDate();
    if (da < 10) da = "0" + da;
    let hh = date.getHours();
    if (hh < 10) hh = "0" + hh;
    let mi = date.getMinutes();
    if (mi < 10) mi = "0" + mi;
    let ss = date.getSeconds();
    if (ss < 10) ss = "0" + ss;
    return yyyy + "-" + mo + "-" + da + " " + hh + ":" + mi + ":" + ss;
}

LIB.timestampHHMMSS = function() {
    let date = new Date();
    let hh = date.getHours();
    if (hh < 10) hh = "0" + hh;
    let mi = date.getMinutes();
    if (mi < 10) mi = "0" + mi;
    let ss = date.getSeconds();
    if (ss < 10) ss = "0" + ss;
    return hh + ":" + mi + ":" + ss;
}
