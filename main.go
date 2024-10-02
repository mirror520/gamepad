package main

/*
#cgo CFLAGS: -Wno-pragma-pack
#cgo CFLAGS: -IViGEm
#cgo LDFLAGS: -LViGEm -lViGEmClient
#include <stdlib.h>
#include <Windows.h>
#include <ViGEm/Client.h>
*/
import "C"
import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Initialize ViGEm client
	client := C.vigem_alloc()
	if client == nil {
		fmt.Println("Failed to allocate ViGEm client")
		return
	}
	defer C.vigem_free(client)

	// Connect to ViGEmBus
	if C.vigem_connect(client) != C.VIGEM_ERROR_NONE {
		fmt.Println("Failed to connect to ViGEmBus")
		return
	}
	defer C.vigem_disconnect(client)

	// Create a virtual Xbox 360 controller
	target := C.vigem_target_x360_alloc()
	if target == nil {
		fmt.Println("Failed to allocate Xbox 360 target")
		return
	}
	defer C.vigem_target_free(target)

	// Add the virtual controller to the system
	if C.vigem_target_add(client, target) != C.VIGEM_ERROR_NONE {
		fmt.Println("Failed to add virtual controller")
		return
	}
	defer C.vigem_target_remove(client, target)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		abxyPad := []C.ushort{C.XUSB_GAMEPAD_A, C.XUSB_GAMEPAD_B, C.XUSB_GAMEPAD_X, C.XUSB_GAMEPAD_Y}
		dPad := []C.ushort{
			C.XUSB_GAMEPAD_DPAD_UP,                               // ↑
			C.XUSB_GAMEPAD_DPAD_UP | C.XUSB_GAMEPAD_DPAD_RIGHT,   // ↗
			C.XUSB_GAMEPAD_DPAD_RIGHT,                            // →
			C.XUSB_GAMEPAD_DPAD_RIGHT | C.XUSB_GAMEPAD_DPAD_DOWN, // ↘
			C.XUSB_GAMEPAD_DPAD_DOWN,                             // ↓
			C.XUSB_GAMEPAD_DPAD_DOWN | C.XUSB_GAMEPAD_DPAD_LEFT,  // ↙
			C.XUSB_GAMEPAD_DPAD_LEFT,                             // ←
			C.XUSB_GAMEPAD_DPAD_LEFT | C.XUSB_GAMEPAD_DPAD_UP,    // ↖
		}

		abxyCount := 0
		dPadCount := 0

		// Create Xbox 360 input report
		var report C.XUSB_REPORT

		abxyPadTicker := time.NewTicker(300 * time.Millisecond)
		dPadTicker := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return

			case <-abxyPadTicker.C:
				button := abxyPad[abxyCount%len(abxyPad)]

				if button&report.wButtons == 0 {
					report.wButtons |= button
				} else {
					report.wButtons ^= button
				}

				// Send the report to the virtual controller
				if C.vigem_target_x360_update(client, target, report) != C.VIGEM_ERROR_NONE {
					fmt.Println("Failed to update virtual controller")
					return
				}

				abxyCount++

			case <-dPadTicker.C:
				button := dPad[dPadCount%len(dPad)]

				report.wButtons &= 0xFFF0
				report.wButtons |= button

				// Send the report to the virtual controller
				if C.vigem_target_x360_update(client, target, report) != C.VIGEM_ERROR_NONE {
					fmt.Println("Failed to update virtual controller")
					return
				}

				dPadCount++
			}
		}
	}(ctx)

	<-quit

	cancel()
	fmt.Println("Virtual controller removed")
}
