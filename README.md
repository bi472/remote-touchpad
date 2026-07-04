# Remote Touchpad (Customized Fork)

This is a customized fork of [remote-touchpad](https://github.com/Unrud/remote-touchpad) tailored for mobile usability, host system controls, and dynamic PWA integration.

## Key Enhancements in this Fork

* **Dynamic PWA (Progressive Web App)**:
  * Dynamic `manifest.json` serving that embeds the session's authentication secret in the PWA `start_url`. This allows the standalone Home Screen web app to connect automatically on iOS Safari (bypassing sandbox restrictions).
  * High-resolution full-bleed custom app icon design cropped specifically for mobile launchers.
  * Offline-capable Service Worker setup for instant loading.
* **Host System Controls**:
  * Built-in dedicated system scene accessible via a top-right `⚙` button.
  * Actions to run local display layout profiles (`display-profiles.sh`) and trigger sleep mode (`systemctl suspend`).
* **KDE Keyboard Layout Auto-Switching**:
  * Integrates with the KDE D-Bus interface (`org.kde.keyboard`) to query active indexes.
  * Intelligently sets the matching active layout index persistently when entering text (resolving Cyrillic/Russian keyboard issues on Wayland portals).
* **Usability & Sensitivity Tuning**:
  * Mouse move speed boosted to `2.0` for responsive cursor travel.
  * Scroll wheel sensitivity optimized to a default multiplier of `0.3` for precise controls.
  * Restyled UI with smaller grid elements and toggle-based keyboard visibility.

Supports Flatpak's RemoteDesktop portal (for Wayland), Windows and X11.

## Installation

* [Flatpak](https://flathub.org/apps/details/com.github.unrud.RemoteTouchpad)
* [Snap](https://snapcraft.io/remote-touchpad)
* [Windows](https://github.com/Unrud/remote-touchpad/releases/latest)
* Golang:
  * Portal & uinput & X11:

    ```sh
    go install -tags portal,uinput,x11 github.com/unrud/remote-touchpad@latest
    ```
  * Windows:

    ```sh
    go install github.com/unrud/remote-touchpad@latest
    ```

## Screenshots

![screenshot 1](https://raw.githubusercontent.com/Unrud/remote-touchpad/master/screenshots/1.png)

![screenshot 2](https://raw.githubusercontent.com/Unrud/remote-touchpad/master/screenshots/2.png)

![screenshot 3](https://raw.githubusercontent.com/Unrud/remote-touchpad/master/screenshots/3.png)

![screenshot 4](https://raw.githubusercontent.com/Unrud/remote-touchpad/master/screenshots/4.png)
