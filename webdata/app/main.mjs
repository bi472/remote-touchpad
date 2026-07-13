/*
 *    Copyright (c) 2018-2019, 2023 Unrud <unrud@outlook.com>
 *
 *    This file is part of Remote-Touchpad.
 *
 *    Remote-Touchpad is free software: you can redistribute it and/or modify
 *    it under the terms of the GNU General Public License as published by
 *    the Free Software Foundation, either version 3 of the License, or
 *    (at your option) any later version.
 *
 *    Remote-Touchpad is distributed in the hope that it will be useful,
 *    but WITHOUT ANY WARRANTY; without even the implied warranty of
 *    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *    GNU General Public License for more details.
 *
 *    You should have received a copy of the GNU General Public License
 *    along with Remote-Touchpad.  If not, see <http://www.gnu.org/licenses/>.
 */

import InputController, * as inputcontrollerModule from "./inputcontroller.mjs";
import Socket from "./socket.mjs";
import UI from "./ui.mjs";

const url = new URL("ws", location.href);
url.protocol = url.protocol == "http:" ? "ws:" : "wss:";

let secret = window.location.hash.substr(1);
if (secret) {
    localStorage.setItem("auth_secret", secret);
} else {
    secret = localStorage.getItem("auth_secret") || "";
}

const socket = new Socket(url, secret);
const inputController = new InputController(socket);
const ui = new UI(inputController);

if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        navigator.serviceWorker.register('./sw.js').catch((err) => {
            console.error('Service Worker registration failed:', err);
        });
    });
}

socket.addEventListener("config", (event) => {
    const config = event.detail;
    if (config.type === "media-state") {
        window.app.handleMediaState(config);
    } else {
        inputController.configure(config);
        ui.configure(config);
    }
});

socket.addEventListener("close", () => {
    ui.close();
});

window.app = {
    key: inputController.keyboardKey.bind(inputController),
    text: inputController.keyboardText.bind(inputController),
    toggleFullscreen: ui.toggleFullscreen.bind(ui),
    showTextInput: ui.showTextInput.bind(ui),
    showKeys: ui.showKeys.bind(ui),
    setKeysPage: ui.setKeysPage.bind(ui),
    focusKeyboard: ui.focusKeyboard.bind(ui),
    showSystemControls: ui.showSystemControls.bind(ui),
    customAction: inputController.customAction.bind(inputController),
    openSleepTimerModal: () => {
        document.getElementById('sleep-timer-modal').classList.remove('hidden');
        window.app.selectTimerPreset(30);
    },
    closeSleepTimerModal: () => {
        document.getElementById('sleep-timer-modal').classList.add('hidden');
    },
    selectTimerPreset: (val) => {
        document.getElementById('selected-timer-val').textContent = val;
        const presets = [15, 30, 45, 60];
        presets.forEach(p => {
            const btn = document.getElementById(`timer-btn-${p}`);
            if (p === val) {
                btn.style.background = '#2a2a2a';
                btn.style.border = '2px solid #007acc';
            } else {
                btn.style.background = '#333';
                btn.style.border = '1px solid #555';
            }
        });
    },
    confirmSleepTimer: () => {
        const val = parseInt(document.getElementById('selected-timer-val').textContent, 10);
        if (!isNaN(val) && val > 0) {
            inputController.customAction(`sleep-timer:${val}`);
        }
        window.app.closeSleepTimerModal();
    },
    handleMediaState: (state) => {
        if (!state.active) {
            if (history.state === "media-controls") {
                history.back();
            }
            return;
        }
        
        if (history.state !== "media-controls") {
            ui.showMediaControls();
        }

        document.getElementById('media-title').textContent = state.title || "Streaming Audio/Video";

        const formatTime = (sec) => {
            if (isNaN(sec) || sec < 0) return "00:00";
            const h = Math.floor(sec / 3600);
            const m = Math.floor((sec % 3600) / 60);
            const s = Math.floor(sec % 60);
            return (h > 0 ? (h + ":" + String(m).padStart(2, '0')) : String(m).padStart(2, '0')) + ":" + String(s).padStart(2, '0');
        };

        document.getElementById('media-time-pos').textContent = formatTime(state.position);
        document.getElementById('media-time-dur').textContent = formatTime(state.duration);

        const progressSlider = document.getElementById('media-progress');
        if (document.activeElement !== progressSlider) {
            const percent = state.duration > 0 ? (state.position / state.duration) * 100 : 0;
            progressSlider.value = percent;
        }

        document.getElementById('media-play-pause-btn').textContent = state.paused ? "▶" : "⏸";

        const volumeSlider = document.getElementById('media-volume');
        if (document.activeElement !== volumeSlider) {
            volumeSlider.value = state.volume;
            document.getElementById('media-volume-val').textContent = Math.round(state.volume) + "%";
        }

        const select = document.getElementById('media-audio-sink');
        if (document.activeElement !== select) {
            select.innerHTML = '<option value="">Default Sink</option>';
            if (state.sinks && state.sinks.length > 0) {
                state.sinks.forEach(sink => {
                    const opt = document.createElement('option');
                    opt.value = sink;
                    let name = sink;
                    if (name.startsWith('alsa_output.')) name = name.substring('alsa_output.'.length);
                    opt.textContent = name;
                    opt.selected = (sink === state.current_sink);
                    select.appendChild(opt);
                });
            }
        }
    },
    mediaSeekSliderInput: (value) => {
        inputController.customAction(`mpv:seek-to:${value}`);
    },
    mediaVolumeSliderInput: (value) => {
        document.getElementById('media-volume-val').textContent = value + "%";
        inputController.customAction(`mpv:volume:${value}`);
    },
    mediaAudioSinkChange: (value) => {
        if (value) {
            inputController.customAction(`audio:set-sink:${value}`);
        }
    },
};
for (const name in inputcontrollerModule) {
    if (name.startsWith("KEY_")) {
        window.app[name] = inputcontrollerModule[name];
    }
}
