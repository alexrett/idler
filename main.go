package main

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#cgo CFLAGS: -x objective-c -fmodules -fobjc-arc
#cgo LDFLAGS: -framework ApplicationServices -framework CoreGraphics
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/pwr_mgt/IOPMLib.h>
#include <ApplicationServices/ApplicationServices.h>

IOReturn createAssertions(IOPMAssertionID *sysID, IOPMAssertionID *dispID) {
    IOReturn rc = IOPMAssertionCreateWithName(
        kIOPMAssertionTypePreventUserIdleSystemSleep,
        kIOPMAssertionLevelOn,
        CFSTR("Go KeepAwake (system)"),
        sysID);
    if (rc != kIOReturnSuccess) return rc;

    return IOPMAssertionCreateWithName(
        kIOPMAssertionTypePreventUserIdleDisplaySleep,
        kIOPMAssertionLevelOn,
        CFSTR("Go KeepAwake (display)"),
        dispID);
}

IOReturn pokeUser() {
    IOPMAssertionID tmp;
    return IOPMAssertionDeclareUserActivity(
        CFSTR("Go KeepAwake user activity"),
        kIOPMUserActiveLocal,
        &tmp);
}

double idleSeconds() {
  return CGEventSourceSecondsSinceLastEventType(
    kCGEventSourceStateCombinedSessionState, kCGAnyInputEventType);
}

void nudgeMouse(int dx, int dy) {
  CGEventRef e = CGEventCreate(NULL);
  CGPoint p = CGEventGetLocation(e);
  CFRelease(e);

  p.x += dx; p.y += dy;
  CGEventRef move1 = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, p, kCGMouseButtonLeft);
  CGEventPost(kCGHIDEventTap, move1);
  CFRelease(move1);

  p.x -= dx; p.y -= dy;
  CGEventRef move2 = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, p, kCGMouseButtonLeft);
  CGEventPost(kCGHIDEventTap, move2);
  CFRelease(move2);
}
*/
import "C"

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

const (
	activeIcon   = "👨‍💻"
	inactiveIcon = "😴"
)

var (
	isActive     = false
	mu           sync.Mutex
	cancelTicker chan struct{}
	sysID        C.IOPMAssertionID
	dispID       C.IOPMAssertionID
	mToggle      *systray.MenuItem
)

func onReady() {
	systray.SetTitle(inactiveIcon)
	systray.SetTooltip("Go KeepAwake for macOS")

	mToggle = systray.AddMenuItem("Prevent Sleep", "Toggle system/display sleep prevention")
	mQuit := systray.AddMenuItem("Quit", "Exit the application")

	go func() {
		for {
			select {
			case <-mToggle.ClickedCh:
				toggleBlocker()
			case <-mQuit.ClickedCh:
				if isActive {
					stopBlocker()
				}
				systray.Quit()
				os.Exit(0)
			}
		}
	}()
}

func toggleBlocker() {
	mu.Lock()
	defer mu.Unlock()

	if isActive {
		stopBlocker()
		systray.SetTitle(inactiveIcon)
		mToggle.SetTitle("Prevent Sleep")
		log.Println("🔓 Sleep is now allowed")
	} else {
		if rc := C.createAssertions(&sysID, &dispID); rc != C.kIOReturnSuccess {
			log.Fatalf("IOKit error: 0x%v", rc)
		}
		isActive = true
		cancelTicker = make(chan struct{})
		go keepAlive(cancelTicker)
		systray.SetTitle(activeIcon)
		mToggle.SetTitle("Allow Sleep")
		log.Println("💡 Sleep prevention is now active")
	}
}

func stopBlocker() {
	C.IOPMAssertionRelease(sysID)
	C.IOPMAssertionRelease(dispID)
	isActive = false
	if cancelTicker != nil {
		close(cancelTicker)
		cancelTicker = nil
	}
}

func keepAlive(stopCh <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			C.pokeUser()
			C.nudgeMouse(1, 1)
		case <-stopCh:
			return
		}
	}
}

func onExit() {
	log.Println("🚪 Exiting application")
	if isActive {
		stopBlocker()
	}
}

func main() {
	if runtime.GOOS != "darwin" {
		fmt.Println("❌ Only supported on macOS (darwin)")
		return
	}
	systray.Run(onReady, onExit)
}
