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
    inputController.configure(config);
    ui.configure(config);
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
};
for (const name in inputcontrollerModule) {
    if (name.startsWith("KEY_")) {
        window.app[name] = inputcontrollerModule[name];
    }
}
