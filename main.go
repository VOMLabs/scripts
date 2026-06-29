package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	EVIOCGNAME = 0x82004506
)

const KEY_MICMUTE = 248

var (
	confDir = os.Getenv("HOME") + "/.config"
	echoCfg = confDir + "/pipewire/pipewire-pulse.conf.d/99-echo-cancel.conf"
	echoDis = echoCfg + ".disabled"

	srcRe = regexp.MustCompile(`alsa_input\.usb-Logitech_PRO_X.*\.mono-fallback`)
)

type inputEvent struct {
	_     [16]byte
	Type  uint16
	Code  uint16
	Value int32
}

func main() {
	fmt.Println("== Logitech PRO X Mic Fix ==\n")

	src := detectSource()
	if src == "" {
		fatal("Logitech PRO X not found. Is it plugged in?")
	}
	fmtf("✓ Source: %s\n", src)

	restart := false

	if id := findEchoMod(); id >= 0 {
		fmtf("✗ Echo-cancel module loaded (id=%d) — unloading... ", id)
		exec.Command("pactl", "unload-module", strconv.Itoa(id)).Run()
		fmtf("OK\n")
		restart = true
	} else {
		fmtf("✓ No echo-cancel module loaded\n")
	}

	if _, err := os.Stat(echoCfg); err == nil {
		fmtf("✗ Echo-cancel config active — disabling... ")
		os.Rename(echoCfg, echoDis)
		fmtf("OK\n")
		restart = true
	} else {
		fmtf("✓ Echo-cancel config not active\n")
	}

	if d := getDefSrc(); d != src {
		fmtf("✗ Default source is %s — fixing... ", d)
		exec.Command("pactl", "set-default-source", src).Run()
		fmtf("OK\n")
		restart = true
	} else {
		fmtf("✓ Default source correct\n")
	}

	exec.Command("pactl", "set-source-volume", src, "65536").Run()
	fmtf("✓ Volume set to 100%%\n")

	exec.Command("pactl", "set-source-mute", src, "0").Run()
	fmtf("✓ Mic unmuted\n")

	on, _ := ledState()
	if on {
		fmtf("⚠ Mute LED is ON — trying software toggle... ")
		if err := toggle(); err != nil {
			fmtf("failed: %v\n", err)
			fmtf("  → Press the mute button on your headset cable.\n")
		} else {
			fmtf("OK\n")
			restart = true
		}
	} else {
		fmtf("✓ Mute LED off\n")
	}

	if restart {
		fmtf("\nRestarting PipeWire-Pulse... ")
		exec.Command("systemctl", "--user", "restart", "pipewire-pulse").Run()
		time.Sleep(2 * time.Second)
		fmtf("OK\n")
	}

	if verifyAudio(src) {
		fmtf("\n✓ Mic working — audio detected!\n")
	} else {
		fmtf("\n⚠ No audio — hardware mute on cable may be engaged.\n")
		fmtf("  Press the mute button on the inline remote.\n")
	}
}

func detectSource() string {
	b, _ := exec.Command("pactl", "list", "sources", "short").Output()
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && srcRe.MatchString(f[1]) {
			return f[1]
		}
	}
	return ""
}

func findEchoMod() int {
	b, _ := exec.Command("pactl", "list", "modules", "short").Output()
	for _, line := range strings.Split(string(b), "\n") {
		if strings.Contains(line, "module-echo-cancel") && strings.Contains(line, "logitech") {
			f := strings.Fields(line)
			if len(f) >= 1 {
				id, _ := strconv.Atoi(f[0])
				return id
			}
		}
	}
	return -1
}

func getDefSrc() string {
	b, _ := exec.Command("pactl", "get-default-source").Output()
	return strings.TrimSpace(string(b))
}

func ledState() (bool, error) {
	es, err := os.ReadDir("/sys/class/leds")
	if err != nil {
		return false, err
	}
	for _, e := range es {
		if !strings.HasSuffix(e.Name(), "::mute") {
			continue
		}
		b, err := os.ReadFile("/sys/class/leds/" + e.Name() + "/brightness")
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(b)) == "1" {
			return true, nil
		}
	}
	return false, nil
}

func findEventDev() (string, error) {
	es, err := os.ReadDir("/dev/input")
	if err != nil {
		return "", err
	}
	for _, e := range es {
		if !strings.HasPrefix(e.Name(), "event") {
			continue
		}
		p := "/dev/input/" + e.Name()
		fd, err := syscall.Open(p, syscall.O_RDWR, 0)
		if err != nil {
			continue
		}
		var name [256]byte
		syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(EVIOCGNAME), uintptr(unsafe.Pointer(&name[0])))
		syscall.Close(fd)
		n := strings.TrimRight(string(name[:]), "\x00")
		if strings.Contains(n, "Logitech") && strings.Contains(n, "PRO X") {
			return p, nil
		}
	}
	return "", errors.New("not found")
}

func toggle() error {
	d, err := findEventDev()
	if err != nil {
		return err
	}
	fd, err := syscall.Open(d, syscall.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", d, err)
	}
	defer syscall.Close(fd)

	writeEv := func(v int32) error {
		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, inputEvent{Type: 1, Code: KEY_MICMUTE, Value: v})
		b := buf.Bytes()
		_, _, e := syscall.Syscall(syscall.SYS_WRITE, uintptr(fd), uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
		if e != 0 {
			return e
		}
		return nil
	}
	writeEv(1)
	time.Sleep(50 * time.Millisecond)
	writeEv(0)
	return nil
}

func verifyAudio(src string) bool {
	cmd := exec.Command("parec", "--device="+src, "--record", "--channels=1", "--rate=48000", "--format=s16le")
	r, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return false
	}
	defer cmd.Process.Kill()

	rd := bufio.NewReader(r)
	buf := make([]byte, 256)
	dl := time.After(3 * time.Second)

	for {
		select {
		case <-dl:
			return false
		default:
			n, err := rd.Read(buf)
			if err != nil {
				return false
			}
			for i := 0; i+1 < n; i += 2 {
				if int16(buf[i])|int16(buf[i+1])<<8 > 15 ||
					int16(buf[i])|int16(buf[i+1])<<8 < -15 {
					return true
				}
			}
		}
	}
}

func fmtf(f string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, f, a...)
}

func fatal(m string) {
	fmt.Fprintln(os.Stderr, m)
	os.Exit(1)
}
