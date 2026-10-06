// Package ui runs the Cascades UI (assets/*.qml) and connects it to Go. The
// C++ side is one QObject: Go posts JSON to it from any goroutine, queued to
// the UI thread, and QML calls back with commands.
//
// moc_bridge.cpp is generated from bridge.h by tools/build.sh.
package ui

/*
#cgo CXXFLAGS: -Wno-unused-parameter
#cgo LDFLAGS: -lbbcascades -lbbsystem -lQtDeclarative -lQtCore
#include <stdlib.h>
#include "ui.h"
*/
import "C"

import (
	"unsafe"
)

var handler func(name, argsJSON string)

// Run shows the UI and returns when the app exits. It must run on a goroutine
// locked to an OS thread for its whole life; term51 uses the main thread.
// commands is called on the UI thread and must not block.
func Run(commands func(name, argsJSON string)) int {
	handler = commands
	return int(C.ui_run())
}

//export goCommand
func goCommand(name, argsJSON *C.char) {
	if handler != nil {
		handler(C.GoString(name), C.GoString(argsJSON))
	}
}

// Publisher posts state and events to the UI. Safe from any goroutine.
type Publisher struct{}

func (Publisher) State(json string) {
	s := C.CString(json)
	C.ui_state(s)
	C.free(unsafe.Pointer(s))
}

func (Publisher) Event(json string) {
	s := C.CString(json)
	C.ui_event(s)
	C.free(unsafe.Pointer(s))
}
