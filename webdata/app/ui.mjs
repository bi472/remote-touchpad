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

import Keyboard from "./keyboard.mjs";
import Mouse from "./mouse.mjs";
import Touchpad from "./touchpad.mjs";
import * as compat from "./compat.mjs";
import { KEY_BACK_SPACE } from "./inputcontroller.mjs";

const IGNORE_CLICK_AFTER_TOUCH_DURATION = 1000; // milliseconds
const CLICK_VIBRATION_PATTERN = [10];

const buttons = document.querySelectorAll("button");
const scenes = document.querySelectorAll("body > .scene");
const openingScene = document.getElementById("opening");
const closedScene = document.getElementById("closed");
const padScene = document.getElementById("pad");
const keysScene = document.getElementById("keys");
const keysPages = keysScene.querySelectorAll(":scope > .page");
const keysTextarea = document.getElementById("keys-textarea");
const textInputScene = document.getElementById("text-input");
const textInput = textInputScene.querySelector("textarea");
const mouseScene = document.getElementById("mouse");
const sendText = document.getElementById("send-text");

export default class UI {
    #activeScene = null;
    #keysActiveName = "";
    #ready = false;
    #closed = false;
    #ignoreClickUntilTimeStamp = Number.MIN_VALUE;
    #inputController;
    #mouse;
    #keyboard;
    #touchpad;
    #oldValue = "";

    constructor(inputController) {
        this.#inputController = inputController;
        this.#mouse = new Mouse(inputController, mouseScene);
        this.#keyboard = new Keyboard(inputController,
            () => this.#activeScene?.classList.contains("keyboard-input"));
        this.#touchpad = new Touchpad(inputController,
            (target) => target.classList.contains("touch-input"));
        document.addEventListener("mousedown", this.#handleMousedown.bind(this));
        document.addEventListener("touchend", this.#handleTouchend.bind(this));
        textInput.addEventListener("input", () => {
            let newValue = textInput.value;
            if (newValue.length > this.#oldValue.length) {
                const added = newValue.substring(this.#oldValue.length);
                this.#inputController.keyboardText(added);
            } else if (newValue.length < this.#oldValue.length) {
                const diff = this.#oldValue.length - newValue.length;
                for (let i = 0; i < diff; i++) {
                    this.#inputController.keyboardKey(KEY_BACK_SPACE);
                }
            }
            if (newValue.length === 0) {
                newValue = "\n";
                textInput.value = "\n";
            }
            this.#oldValue = newValue;
            this.#updateTextInput();
        });
        sendText.addEventListener("click", this.#handleSendText.bind(this));
        window.addEventListener("popstate", () => { this.#update(); });
        compat.addFullscreenchangeEventListener(() => { this.#update(); });
        compat.addPointerlockchangeEventListener(() => { this.#update(); });
        keysTextarea.addEventListener("input", () => {
            let newValue = keysTextarea.value;
            if (newValue.length > this.#oldValue.length) {
                const added = newValue.substring(this.#oldValue.length);
                this.#inputController.keyboardText(added);
            } else if (newValue.length < this.#oldValue.length) {
                const diff = this.#oldValue.length - newValue.length;
                for (let i = 0; i < diff; i++) {
                    this.#inputController.keyboardKey(KEY_BACK_SPACE);
                }
            }
            if (newValue.length === 0) {
                newValue = "\n";
                keysTextarea.value = "\n";
            }
            this.#oldValue = newValue;
        });


        for (const button of buttons) {
            button.setAttribute("tabindex", "-1");
            button.addEventListener("click", this.#handleButtonClick.bind(this));
        }
        this.#update();
    }

   configure(config) {
        this.#mouse.configure(config);
        this.#keyboard.configure(config);
        this.#touchpad.configure(config);
        this.#closed = false;
        this.#ready = true;
        this.#update();
    }

    close() {
        this.#ready = false;
        this.#closed = true;
        this.#update();
    }

    #handleTouchend(event) {
        // HACK: event.preventDefault doesn't reliably stop click events in Firefox (112)
        this.#ignoreClickUntilTimeStamp = event.timeStamp + IGNORE_CLICK_AFTER_TOUCH_DURATION;
    }
    
    #handleMousedown(event) {
        if (this.#activeScene != mouseScene && event.buttons == 1 &&
            this.#ignoreClickUntilTimeStamp <= event.timeStamp &&
            event.target.classList.contains("mouse-input")) {
            compat.requestPointerLock(mouseScene);
        }
    }

    #updateTextInput(newValue = null) {
        if (newValue != null) {
            textInput.value = newValue;
        }
        sessionStorage.setItem("text-input", textInput.value);
        sendText.textContent = sendText.getAttribute("data-back-content");
        textInput.focus();
    }

    #handleButtonClick(event) {
        event.target.classList.add("click");
        setTimeout(() => event.target.classList.remove("click"), 0);
        compat.vibrate(CLICK_VIBRATION_PATTERN);
    }

    #handleSendText() {
        history.back();
    }

    #showScene(scene) {
        this.#activeScene = scene;
        if (compat.fullscreenElement() && !scene.classList.contains("allow-fullscreen")) {
            compat.exitFullscreen();
        }
        if (compat.pointerLockElement() && this.#activeScene != mouseScene) {
            compat.exitPointerLock();
        }
        textInput.value = "";
        for (const otherScene of scenes) {
            otherScene.classList.toggle("hidden", otherScene != scene);
        }
    }

    setKeysPage(index, relative = false) {
        if (relative) {
            for (let i = 0; i < keysPages.length && keysPages[i].classList.contains("hidden");
                 i += 1, index += 1);
        }
        index = ((index % keysPages.length) + keysPages.length) % keysPages.length;
        sessionStorage.setItem(this.#keysActiveName, index);
        for (let i = 0; i < keysPages.length; i += 1) {
            keysPages[i].classList.toggle("hidden", i != index);
        }
    }

    showKeys(name = "", defaultPageIndex = 0) {
        this.#showScene(keysScene);
        this.#keysActiveName = "keys" + (name ? ":" + name : "");
        let pageIndex = parseInt(sessionStorage.getItem(this.#keysActiveName));
        if (isNaN(pageIndex)) {
            pageIndex = defaultPageIndex;
        }
        this.setKeysPage(pageIndex);
        if (history.state != this.#keysActiveName) {
            history.pushState(this.#keysActiveName, "");
        }
    }

    showTextInput() {
        this.#showScene(textInputScene);
        this.#oldValue = "\n";
        this.#updateTextInput("\n");
        if (history.state != "text-input") {
            history.pushState("text-input", "");
        }
    }

    focusKeyboard() {
        if (document.activeElement === keysTextarea) {
            keysTextarea.blur();
        } else {
            this.#oldValue = "\n";
            keysTextarea.value = "\n";
            keysTextarea.focus();
        }
    }

    showSystemControls() {
        const systemControlsScene = document.getElementById("system-controls");
        this.#showScene(systemControlsScene);
        if (history.state != "system-controls") {
            history.pushState("system-controls", "");
        }
    }

    toggleFullscreen() {
        if (compat.fullscreenElement()) {
            compat.exitFullscreen();
        } else {
            compat.requestFullscreen(document.documentElement, {navigationUI: "hide"});
        }
    }

    #update() {
        const fullscreenEnabled = compat.fullscreenEnabled();
        for (const element of document.querySelectorAll(".visble-if-fullscreen-enabled")) {
            element.classList.toggle("hidden", !fullscreenEnabled);
        }
        if (!this.#ready) {
            this.#showScene(this.#closed ? closedScene : openingScene);
        } else if (compat.pointerLockElement()) {
            this.#showScene(mouseScene);
        } else if ((history.state || "").split(":")[0] == "keys") {
            this.showKeys(history.state.substr("keys:".length));
        } else if (history.state == "text-input") {
            this.showTextInput();
        } else if (history.state == "system-controls") {
            this.showSystemControls();
        } else {
            this.showKeys("", 0);
        }
    }
}
