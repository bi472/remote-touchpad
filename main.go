/*
 *    Copyright (c) 2018-2019 Unrud <unrud@outlook.com>
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

package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"github.com/unrud/remote-touchpad/inputcontrol"
	"github.com/unrud/remote-touchpad/terminal"
	"golang.org/x/net/websocket"
	"log"
	mathrand "math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	defaultSecretLength     int           = 8
	authenticationRateLimit time.Duration = time.Second / 10
	authenticationRateBurst int           = 10
	challengeLength         int           = 8
	defaultBind             string        = ":0"
	version                 string        = "1.5.3"
	prettyAppName           string        = "Remote Touchpad"
)

type config struct {
	UpdateRate       uint    `json:"updateRate"`
	ScrollSpeed      float64 `json:"scrollSpeed"`
	MoveSpeed        float64 `json:"moveSpeed"`
	MouseScrollSpeed float64 `json:"mouseScrollSpeed"`
	MouseMoveSpeed   float64 `json:"mouseMoveSpeed"`
}

func processCommand(controller inputcontrol.Controller, command string) error {
	if len(command) == 0 {
		return errors.New("empty command")
	}
	if command == "S" {
		return controller.PointerScroll(0, 0, true)
	}
	if command[0] == 't' {
		text := command[1:]
		if !utf8.ValidString(text) {
			return errors.New("invalid utf-8")
		}
		log.Printf("processCommand: typing text %q", text)
		return controller.KeyboardText(text)
	}
	if command[0] == 'c' {
		action := command[1:]
		if action == "sleep" {
			log.Printf("Executing sleep command...")
			cmd := exec.Command("systemctl", "suspend")
			return cmd.Run()
		}
		if action == "display-profiles" {
			log.Printf("Executing display-profiles script...")
			cmd := exec.Command("/home/fnder77/.local/bin/display-profiles.sh")
			return cmd.Run()
		}
		if action == "display-profiles-extended" {
			log.Printf("Executing display-profiles-extended script...")
			cmd := exec.Command("/home/fnder77/.local/bin/display-profiles-extended.sh")
			return cmd.Run()
		}
		if action == "display-profiles-huawei-only" {
			log.Printf("Executing display-profiles-huawei-only script...")
			cmd := exec.Command("/home/fnder77/.local/bin/display-profiles-huawei-only.sh")
			return cmd.Run()
		}
		if action == "open-mpv" {
			log.Printf("Opening mpv...")
			cmd := exec.Command("mpv", "--fs")
			err := cmd.Start()
			if err != nil {
				log.Printf("Error starting mpv: %v", err)
				return err
			}
			go func() {
				cmd.Wait()
			}()
			return nil
		}
		if action == "open-youtube" {
			log.Printf("Opening YouTube...")
			cmd := exec.Command("xdg-open", "https://youtube.com")
			err := cmd.Start()
			if err != nil {
				log.Printf("Error opening YouTube: %v", err)
				return err
			}
			go func() {
				cmd.Wait()
			}()
			return nil
		}
		if action == "mpv:play-pause" {
			return runPlayerctlCmd("play-pause")
		}
		if strings.HasPrefix(action, "mpv:seek:") {
			secs := action[len("mpv:seek:"):]
			if strings.HasPrefix(secs, "-") {
				return runPlayerctlCmd("position", secs[1:]+"-")
			} else {
				return runPlayerctlCmd("position", secs+"+")
			}
		}
		if strings.HasPrefix(action, "mpv:seek-to:") {
			percentStr := action[len("mpv:seek-to:"):]
			if percent, e := strconv.ParseFloat(percentStr, 64); e == nil {
				mediaStateMu.Lock()
				dur := mediaState.Duration
				mediaStateMu.Unlock()
				if dur > 0 {
					targetSecs := (percent / 100.0) * dur
					return runPlayerctlCmd("position", fmt.Sprintf("%f", targetSecs))
				}
			}
			return nil
		}
		if strings.HasPrefix(action, "mpv:volume:") {
			volStr := action[len("mpv:volume:"):]
			if vol, e := strconv.ParseFloat(volStr, 64); e == nil {
				return exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", fmt.Sprintf("%d%%", int(vol))).Run()
			}
			return nil
		}
		if action == "mpv:playlist:next" {
			return runPlayerctlCmd("next")
		}
		if action == "mpv:playlist:prev" {
			return runPlayerctlCmd("previous")
		}
		if strings.HasPrefix(action, "player:set-player:") {
			player := action[len("player:set-player:"):]
			mediaStateMu.Lock()
			currentPlayer = player
			mediaStateMu.Unlock()
			return nil
		}
		if strings.HasPrefix(action, "audio:set-sink:") {
			sink := action[len("audio:set-sink:"):]
			log.Printf("Setting default audio sink to: %s", sink)
			cmd := exec.Command("pactl", "set-default-sink", sink)
			err := cmd.Run()
			if err == nil {
				mediaStateMu.Lock()
				mediaState.Sinks, mediaState.CurrentSink = fetchAudioSinks()
				broadcastMessage(mediaState)
				mediaStateMu.Unlock()
			}
			return err
		}
		if action == "cancel-sleep-timer" {
			log.Printf("Cancelling sleep timer...")
			cmd := exec.Command("pkill", "-f", "sleep_timer.sh")
			return cmd.Run()
		}
		if strings.HasPrefix(action, "sleep-timer:") {
			parts := strings.Split(action, ":")
			if len(parts) == 2 {
				val := parts[1]
				log.Printf("Starting sleep timer for %s minutes...", val)
				exec.Command("pkill", "-f", "sleep_timer.sh").Run()
				cmd := exec.Command("/home/fnder77/Проекты/sleep_timer.sh", val)
				err := cmd.Start()
				if err != nil {
					log.Printf("Error starting sleep timer: %v", err)
					return err
				}
				go func() {
					cmd.Wait()
				}()
				return nil
			}
		}
		return errors.New("unknown custom action")
	}
	arguments := strings.Split(command[1:], ";")
	if command[0] == 'k' && len(arguments) != 1 ||
		command[0] != 'k' && len(arguments) != 2 {
		return errors.New("wrong number of arguments")
	}
	x, err := strconv.ParseInt(arguments[0], 10, 32)
	if err != nil {
		return err
	}
	if command[0] == 'k' {
		if x < 0 || x >= int64(inputcontrol.KeyLimit) {
			return errors.New("unsupported key")
		}
		return controller.KeyboardKey(inputcontrol.Key(x))
	}
	y, err := strconv.ParseInt(arguments[1], 10, 32)
	if err != nil {
		return err
	}
	if command[0] == 'm' {
		return controller.PointerMove(int(x), int(y))
	}
	if command[0] == 's' {
		return controller.PointerScroll(int(x), int(y), false)
	}
	if command[0] == 'S' {
		return controller.PointerScroll(int(x), int(y), true)
	}
	if command[0] == 'b' {
		if x < 0 || x >= int64(inputcontrol.PointerButtonLimit) {
			return errors.New("unsupported pointer button")
		}
		b := true
		if y == 0 {
			b = false
		}
		return controller.PointerButton(inputcontrol.PointerButton(x), b)
	}
	return errors.New("unsupported command")
}

type challenge struct {
	message, expectedResponse string
}

func (c challenge) verify(response string) bool {
	return c.expectedResponse == response
}

func authenticationChallengeGenerator(secret string, challenges chan<- challenge) {
	unsecureSource := mathrand.NewSource(time.Now().UnixNano())
	unsecureRand := mathrand.New(unsecureSource)
	b := make([]byte, challengeLength)
	for {
		if _, err := unsecureRand.Read(b[:]); err != nil {
			log.Fatal(err)
		}
		message := base64.StdEncoding.EncodeToString(b[:])
		mac := hmac.New(sha256.New, []byte(message))
		mac.Write([]byte(secret))
		challenges <- challenge{
			message:          message,
			expectedResponse: base64.StdEncoding.EncodeToString(mac.Sum(nil)),
		}
		time.Sleep(authenticationRateLimit)
	}
}

func secureRandBase64(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b[:]); err != nil {
		log.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b[:])
}

func main() {
	terminal.SetTitle(prettyAppName)
	var bind, certFile, keyFile, secret string
	var showVersion bool
	var config config
	flag.BoolVar(&showVersion, "version", false, "show program's version number and exit")
	flag.StringVar(&bind, "bind", defaultBind, "bind server to [HOSTNAME]:PORT")
	flag.StringVar(&secret, "secret", "", "shared secret for client authentication")
	flag.StringVar(&certFile, "cert", "", "file containing TLS certificate")
	flag.StringVar(&keyFile, "key", "", "file containing TLS private key")
	flag.UintVar(&config.UpdateRate, "update-rate", 30, "number of updates per second")
	flag.Float64Var(&config.MoveSpeed, "move-speed", 1, "move speed multiplier")
	flag.Float64Var(&config.ScrollSpeed, "scroll-speed", 0.3, "scroll speed multiplier")
	flag.Float64Var(&config.MouseMoveSpeed, "mouse-move-speed", 1, "mouse move speed multiplier")
	flag.Float64Var(&config.MouseScrollSpeed, "mouse-scroll-speed", 0.3, "mouse scroll speed multiplier")
	flag.Parse()
	if showVersion {
		fmt.Println(version)
		return
	}
	if certFile != "" && keyFile == "" {
		log.Fatal("TLS private key file missing")
	}
	if certFile == "" && keyFile != "" {
		log.Fatal("TLS certificate file missing")
	}
	tls := certFile != "" && keyFile != ""
	if secret == "" {
		secret = secureRandBase64(defaultSecretLength)
	}
	if len(inputcontrol.Controllers) == 0 {
		log.Fatal("compiled without controller")
	}
	var controller inputcontrol.Controller
	var controllerName string
	var platformErrs []error
	for _, controllerInfo := range inputcontrol.Controllers {
		controllerName = controllerInfo.Name
		var err error
		controller, err = controllerInfo.Init()
		if err == nil {
			break
		} else {
			var unsupportedErr *inputcontrol.UnsupportedPlatformError
			wrappedErr := fmt.Errorf("%v controller: %w", controllerName, err)
			if errors.As(err, &unsupportedErr) {
				platformErrs = append(platformErrs, wrappedErr)
			} else {
				log.Fatal(wrappedErr)
			}
		}
	}
	if controller == nil {
		log.Fatal(fmt.Errorf("unsupported platform:\n%w", errors.Join(platformErrs...)))
	}
	defer controller.Close()
	go startMprisManager()
	authenticationChallenges := make(chan challenge, authenticationRateBurst)
	go authenticationChallengeGenerator(secret, authenticationChallenges)
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatal(err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	host := ""
	bindHost, _, err := net.SplitHostPort(bind)
	if err != nil {
		log.Fatal(err)
	}
	for _, b := range addr.IP {
		if b != 0 {
			host = bindHost
			break
		}
	}
	if host == "" {
		host = findDefaultHost()
	}
	port := addr.Port
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
		fmt.Fprintf(w, `{
  "name": "Remote Touchpad",
  "short_name": "Touchpad",
  "start_url": "/#%s",
  "display": "standalone",
  "background_color": "#404040",
  "theme_color": "#404040",
  "orientation": "portrait",
  "icons": [
    {
      "src": "icon-192.png",
      "type": "image/png",
      "sizes": "192x192",
      "purpose": "any maskable"
    },
    {
      "src": "icon-512.png",
      "type": "image/png",
      "sizes": "512x512",
      "purpose": "any maskable"
    }
  ]
}`, secret)
	})
	mux.Handle("/", http.FileServer(http.FS(webdataFS)))
	mux.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		var message string
		challenge := <-authenticationChallenges
		websocket.Message.Send(ws, challenge.message)
		if err := websocket.Message.Receive(ws, &message); err != nil {
			return
		}
		if !challenge.verify(message) {
			return
		}
		wsClientsMu.Lock()
		wsClients[ws] = true
		wsClientsMu.Unlock()
		defer func() {
			wsClientsMu.Lock()
			delete(wsClients, ws)
			wsClientsMu.Unlock()
		}()

		websocket.JSON.Send(ws, config)
		mediaStateMu.Lock()
		websocket.JSON.Send(ws, mediaState)
		mediaStateMu.Unlock()
		for {
			if err := websocket.Message.Receive(ws, &message); err != nil {
				return
			}
			if err := processCommand(controller, message); err != nil {
				log.Print(fmt.Errorf("%s controller: %w", controllerName, err))
				return
			}
		}
	}))
	domain := host
	if port != 80 && !tls || port != 443 && tls {
		domain = net.JoinHostPort(host, strconv.Itoa(port))
	}
	scheme := "http"
	if tls {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/#%s", scheme, domain, secret)
	fmt.Println(url)
	if qrCode, err := terminal.GenerateQRCode(url, terminal.SupportsColor(os.Stdout.Fd())); err == nil {
		fmt.Print(qrCode)
	} else {
		log.Printf("QR code error: %v", err)
	}
	if !tls {
		fmt.Println("▌   WARNING: TLS is not enabled    ▐")
		fmt.Println("▌Don't use in an untrusted network!▐")
	}
	if tls {
		err = http.ServeTLS(listener, mux, certFile, keyFile)
	} else {
		err = http.Serve(listener, mux)
	}
	log.Fatal(err)
}

type MediaState struct {
	Type          string   `json:"type"`
	Title         string   `json:"title"`
	Position      float64  `json:"position"`
	Duration      float64  `json:"duration"`
	Paused        bool     `json:"paused"`
	Volume        float64  `json:"volume"`
	Sinks         []string `json:"sinks"`
	CurrentSink   string   `json:"current_sink"`
	Active        bool     `json:"active"`
	Players       []string `json:"players"`
	CurrentPlayer string   `json:"current_player"`
}

var (
	wsClientsMu   sync.Mutex
	wsClients     = make(map[*websocket.Conn]bool)
	mediaState     MediaState
	mediaStateMu   sync.Mutex
	currentPlayer  string
)

func broadcastMessage(msg interface{}) {
	wsClientsMu.Lock()
	defer wsClientsMu.Unlock()
	for ws := range wsClients {
		websocket.JSON.Send(ws, msg)
	}
}

func runPlayerctlCmd(actionCmd string, extraArgs ...string) error {
	mediaStateMu.Lock()
	p := currentPlayer
	mediaStateMu.Unlock()

	args := []string{}
	if p != "" {
		args = append(args, "-p", p)
	}
	args = append(args, actionCmd)
	args = append(args, extraArgs...)

	return exec.Command("playerctl", args...).Run()
}

func fetchSystemVolume() float64 {
	out, err := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@").Output()
	if err != nil {
		return 100.0
	}
	str := string(out)
	idx := strings.Index(str, "%")
	if idx == -1 {
		return 100.0
	}
	start := idx - 1
	for start >= 0 && (str[start] >= '0' && str[start] <= '9' || str[start] == ' ' || str[start] == '\t') {
		start--
	}
	valStr := strings.TrimSpace(str[start+1 : idx])
	if val, err := strconv.ParseFloat(valStr, 64); err == nil {
		return val
	}
	return 100.0
}

func fetchActivePlayers() []string {
	out, err := exec.Command("playerctl", "-l").Output()
	if err != nil {
		return []string{}
	}
	lines := strings.Split(string(out), "\n")
	players := []string{}
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name != "" {
			players = append(players, name)
		}
	}
	return players
}

func fetchAudioSinks() ([]string, string) {
	sinks := []string{}
	currentSink := ""

	out, err := exec.Command("pactl", "get-default-sink").Output()
	if err == nil {
		currentSink = strings.TrimSpace(string(out))
	}

	outSinks, err := exec.Command("pactl", "list", "short", "sinks").Output()
	if err == nil {
		lines := strings.Split(string(outSinks), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				sinks = append(sinks, parts[1])
			}
		}
	}
	return sinks, currentSink
}

func parsePlayerctlOutput(output string) (title string, position float64, duration float64, paused bool, volume float64, err error) {
	parts := strings.Split(strings.TrimSpace(output), "||")
	if len(parts) < 5 {
		return "", 0, 0, false, 0, fmt.Errorf("invalid format")
	}

	title = parts[0]

	if parts[1] != "" {
		if posMicro, e := strconv.ParseFloat(parts[1], 64); e == nil {
			position = posMicro / 1000000.0
		}
	}

	if parts[2] != "" {
		if lenMicro, e := strconv.ParseFloat(parts[2], 64); e == nil {
			duration = lenMicro / 1000000.0
		}
	}

	paused = (parts[3] != "Playing")

	if parts[4] != "" {
		if volFloat, e := strconv.ParseFloat(parts[4], 64); e == nil {
			volume = volFloat * 100.0
		}
	}

	return title, position, duration, paused, volume, nil
}

func startMprisManager() {
	for {
		time.Sleep(500 * time.Millisecond)

		players := fetchActivePlayers()
		if len(players) == 0 {
			mediaStateMu.Lock()
			if mediaState.Active {
				mediaState = MediaState{Type: "media-state", Active: false}
				broadcastMessage(mediaState)
			}
			mediaStateMu.Unlock()
			continue
		}

		mediaStateMu.Lock()
		p := currentPlayer
		found := false
		for _, pl := range players {
			if pl == p {
				found = true
				break
			}
		}
		if !found {
			currentPlayer = ""
			p = ""
		}
		mediaStateMu.Unlock()

		args := []string{"metadata", "--format", "{{title}}||{{position}}||{{mpris:length}}||{{status}}||{{volume}}"}
		if p != "" {
			args = append([]string{"-p", p}, args...)
		}
		out, err := exec.Command("playerctl", args...).Output()
		if err != nil {
			mediaStateMu.Lock()
			if mediaState.Active {
				mediaState = MediaState{Type: "media-state", Active: false}
				broadcastMessage(mediaState)
			}
			mediaStateMu.Unlock()
			continue
		}

		title, position, duration, paused, _, err := parsePlayerctlOutput(string(out))
		if err != nil {
			continue
		}

		mediaStateMu.Lock()
		mediaState.Type = "media-state"
		mediaState.Active = true
		mediaState.Title = title
		mediaState.Position = position
		mediaState.Duration = duration
		mediaState.Paused = paused
		mediaState.Volume = fetchSystemVolume()
		mediaState.Players = players
		mediaState.CurrentPlayer = currentPlayer
		mediaState.Sinks, mediaState.CurrentSink = fetchAudioSinks()

		broadcastMessage(mediaState)
		mediaStateMu.Unlock()
	}
}
