{{ with .Site.Params.plantuml }}
{{ if .enable }}
{{ if .svg }}
// https://unpkg.com/external-svg-loader@1.3.4/svg-loader.min.js
(function(){function t(e,r,n){function s(i,a){if(!r[i]){if(!e[i]){var c="function"==typeof require&&require;if(!a&&c)return c(i,!0);if(o)return o(i,!0);var u=new Error("Cannot find module '"+i+"'");throw u.code="MODULE_NOT_FOUND",u}var l=r[i]={exports:{}};e[i][0].call(l.exports,(function(t){var r=e[i][1][t];return s(r||t)}),l,l.exports,t,e,r,n)}return r[i].exports}for(var o="function"==typeof require&&require,i=0;i<n.length;i++)s(n[i]);return s}return t})()({1:[function(t,e,r){"use strict";let n=0;e.exports={incr(){return++n},decr(){return--n},curr(){return n}}},{}],2:[function(t,e,r){"use strict";e.exports=(t,e,r="")=>{const n=/url\("?#([a-zA-Z-0-9][\w:.-]*)"?\)/g;const s=/#([a-zA-Z][\w:.-]*)/g;if(e.match(n)){e=e.replace(n,(function(e,r){if(!t[r]){return e}return`url(#${t[r]})`}))}if(["href","xlink:href"].includes(r)){if(e.match(s)){e=e.replace(s,(function(e,r){if(!t[r]){return e}return`#${t[r]}`}))}}return e}},{}],3:[function(t,e,r){"use strict";e.exports=(t,e)=>{const r=new RegExp("([^\r\n,{}]+)(,(?=[^}]*{)|s*{)","g");t=t.replace(r,(function(t,r,n){if(r.match(/^\s*(@media|@.*keyframes|to|from|@font-face|1?[0-9]?[0-9])/)){return r+n}r=r.replace(/^(\s*)/,"$1"+e+" ");return r+n}));return t}},{}],4:[function(t,e,r){"use strict";Object.defineProperty(r,"__esModule",{value:true});class n{constructor(t="keyval-store",e="keyval"){this.storeName=e;this._dbp=new Promise(((r,n)=>{const s=indexedDB.open(t,1);s.onerror=()=>n(s.error);s.onsuccess=()=>r(s.result);s.onupgradeneeded=()=>{s.result.createObjectStore(e)}}))}_withIDBStore(t,e){return this._dbp.then((r=>new Promise(((n,s)=>{const o=r.transaction(this.storeName,t);o.oncomplete=()=>n();o.onabort=o.onerror=()=>s(o.error);e(o.objectStore(this.storeName))}))))}}let s;function o(){if(!s)s=new n;return s}function i(t,e=o()){let r;return e._withIDBStore("readonly",(e=>{r=e.get(t)})).then((()=>r.result))}function a(t,e,r=o()){return r._withIDBStore("readwrite",(r=>{r.put(e,t)}))}function c(t,e=o()){return e._withIDBStore("readwrite",(e=>{e.delete(t)}))}function u(t=o()){return t._withIDBStore("readwrite",(t=>{t.clear()}))}function l(t=o()){const e=[];return t._withIDBStore("readonly",(t=>{(t.openKeyCursor||t.openCursor).call(t).onsuccess=function(){if(!this.result)return;e.push(this.result.key);this.result.continue()}})).then((()=>e))}r.Store=n;r.get=i;r.set=a;r.del=c;r.clear=u;r.keys=l},{}],5:[function(t,e,r){"use strict";const{get:n,set:s,del:o}=t("idb-keyval");const i=t("./lib/scope-css");const a=t("./lib/css-url-fixer");const c=t("./lib/counter");const u=async t=>{try{let e=await n(`loader_${t}`);if(!e){return}e=JSON.parse(e);if(Date.now()<e.expiry){return e.data}else{o(`loader_${t}`);return}}catch(t){return}};const l=async(t,e,r)=>{try{const n=parseInt(r,10);await s(`loader_${t}`,JSON.stringify({data:e,expiry:Date.now()+(Number.isNaN(n)?60*60*1e3*24:n)}))}catch(t){console.error(t)}};const d=[];const f=()=>{if(d.length){return d}for(const t in document.head){if(t.startsWith("on")){d.push(t)}}return d};const b={};const h=(t,e,r)=>{const{enableJs:n,disableUniqueIds:s,disableCssScoping:o}=e;const u=new DOMParser;const l=u.parseFromString(r,"text/html");const d=l.querySelector("svg");const h=f();const g=b[t.getAttribute("data-id")]||new Set;const p=t.getAttribute("data-id")||`svg-loader_${c.incr()}`;const m={};if(!s){Array.from(l.querySelectorAll("[id]")).forEach((t=>{const e=t.getAttribute("id");const r=`${e}_${c.incr()}`;t.setAttribute("id",r);m[e]=r}))}Array.from(l.querySelectorAll("*")).forEach((t=>{if(t.tagName==="script"){if(!n){t.remove();return}else{const e=document.createElement("script");e.innerHTML=t.innerHTML;document.body.appendChild(e)}}for(let e=0;e<t.attributes.length;e++){const{name:r,value:s}=t.attributes[e];const o=a(m,s,r);if(s!==o){t.setAttribute(r,o)}if(h.includes(r.toLowerCase())&&!n){t.removeAttribute(r);continue}if(["href","xlink:href"].includes(r)&&s.startsWith("javascript")&&!n){t.removeAttribute(r)}}if(t.tagName==="style"&&!o){let e=i(t.innerHTML,`[data-id="${p}"]`);e=a(m,e);if(e!==t.innerHTML)t.innerHTML=e}}));for(let e=0;e<d.attributes.length;e++){const{name:r,value:n}=d.attributes[e];if(!t.getAttribute(r)||g.has(r)){g.add(r);t.setAttribute(r,n)}}b[p]=g;t.setAttribute("data-id",p);t.innerHTML=d.innerHTML};const g={};const p={};const m=async t=>{const e=t.getAttribute("data-src");const r=t.getAttribute("data-cache");const n=t.getAttribute("data-js")==="enabled";const s=t.getAttribute("data-unique-ids")==="disabled";const o=t.getAttribute("data-css-scoping")==="disabled";const i=await u(e);const a=r!=="disabled";const c=h.bind(this,t,{enableJs:n,disableUniqueIds:s,disableCssScoping:o});if(p[e]||a&&i){const t=p[e]||i;c(t)}else{if(g[e]){setTimeout((()=>m(t)),20);return}g[e]=true;fetch(e).then((t=>{if(!t.ok){throw Error(`Request for '${e}' returned ${t.status} (${t.statusText})`)}return t.text()})).then((t=>{const n=t.toLowerCase().trim();if(!(n.startsWith("<svg")||n.startsWith("<?xml"))){throw Error(`Resource '${e}' returned an invalid SVG file`)}if(a){l(e,t,r)}p[e]=t;c(t)})).catch((t=>{console.error(t)})).finally((()=>{delete g[e]}))}};let y;if(globalThis.IntersectionObserver){const t=new IntersectionObserver((e=>{e.forEach((e=>{if(e.isIntersecting){m(e.target);t.unobserve(e.target)}}))}),{rootMargin:"1200px"})}const v=[];function w(){Array.from(document.querySelectorAll("svg[data-src]:not([data-id])")).forEach((t=>{if(v.indexOf(t)!==-1){return}v.push(t);if(t.getAttribute("data-loading")==="lazy"){y.observe(t)}else{m(t)}}))}let A=false;const x=()=>{if(A){return}A=true;const t=new MutationObserver((t=>{const e=t.some((t=>Array.from(t.addedNodes).some((t=>t.nodeType===Node.ELEMENT_NODE&&(t.getAttribute("data-src")&&!t.getAttribute("data-id")||t.querySelector("svg[data-src]:not([data-id])"))))));if(e){w()}t.forEach((t=>{if(t.type==="attributes"){m(t.target)}}))}));t.observe(document.documentElement,{attributeFilter:["data-src"],attributes:true,childList:true,subtree:true})};if(globalThis.addEventListener){const t=setInterval((()=>{w()}),100);globalThis.addEventListener("DOMContentLoaded",(()=>{clearInterval(t);w();x()}))}},{"./lib/counter":1,"./lib/css-url-fixer":2,"./lib/scope-css":3,"idb-keyval":4}]},{},[5]);
{{ end }}

(function($) {

    function encode64(data) {
        r = "";
        for (i = 0; i < data.length; i += 3) {
            if (i + 2 == data.length) {
                r += append3bytes(data.charCodeAt(i), data.charCodeAt(i + 1), 0);
            } else if (i + 1 == data.length) {
                r += append3bytes(data.charCodeAt(i), 0, 0);
            } else {
                r += append3bytes(data.charCodeAt(i), data.charCodeAt(i + 1),
                    data.charCodeAt(i + 2));
            }
        }
        return r;
    }

    function append3bytes(b1, b2, b3) {
        c1 = b1 >> 2;
        c2 = ((b1 & 0x3) << 4) | (b2 >> 4);
        c3 = ((b2 & 0xF) << 2) | (b3 >> 6);
        c4 = b3 & 0x3F;
        r = "";
        r += encode6bit(c1 & 0x3F);
        r += encode6bit(c2 & 0x3F);
        r += encode6bit(c3 & 0x3F);
        r += encode6bit(c4 & 0x3F);
        return r;
    }

    function encode6bit(b) {
        if (b < 10) {
            return String.fromCharCode(48 + b);
        }
        b -= 10;
        if (b < 26) {
            return String.fromCharCode(65 + b);
        }
        b -= 26;
        if (b < 26) {
            return String.fromCharCode(97 + b);
        }
        b -= 26;
        if (b == 0) {
            return '-';
        }
        if (b == 1) {
            return '_';
        }
        return '?';
    }

    var needPlantuml = false;
    $('.language-plantuml').parent().replaceWith(function() {
        let s = unescape(encodeURIComponent($(this).text()));
        {{ if .svg }}
        return $('<svg data-src="{{.svg_image_url | default "//www.plantuml.com/plantuml/svg/"}}' + encode64(deflate(s, 9)) + '">')
        {{ else }}
        return $('<img src="{{.svg_image_url | default "//www.plantuml.com/plantuml/svg/"}}' + encode64(deflate(s, 9)) + '">')
        {{ end }}
    });
})(jQuery);
{{ end }}
{{ end }}
