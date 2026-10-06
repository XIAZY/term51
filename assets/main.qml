import bb.cascades 1.4

// Milestone 0: the native window Go draws into fills the page, and key
// presses reach Go by one of two paths, switched with the action:
// - Cascades keys: the page's KeyListener, sent to Go as a "key" command;
// - raw keys: the ForeignWindowControl forwards Screen keyboard events to the
//   native window, where Go reads them.
Page {
    id: page

    property bool rawKeys: false
    property bool created: false

    function send(name, args) { bridge.send(name, JSON.stringify(args || {})) }

    function sized(frame) {
        if (frame.width <= 0 || frame.height <= 0)
            return
        if (!created) {
            created = true
            send("window", { group: fwc.windowGroup, id: fwc.windowId, w: frame.width, h: frame.height })
        } else {
            send("size", { w: frame.width, h: frame.height })
        }
    }

    function applyKeys() {
        // InputType: None = 0, Keyboard = 1 (bb::cascades::InputType, 10.3.1).
        fwc.forwardedInputTypes = rawKeys ? 1 : 0
        fwc.requestFocus()
        send("keys", { raw: rawKeys })
    }

    function keyArgs(event, down) {
        return { down: down, key: event.key, keycap: event.keycap, unicode: event.unicode,
                 shift: event.shiftPressed, ctrl: event.ctrlPressed, alt: event.altPressed }
    }

    actions: [
        ActionItem {
            title: page.rawKeys ? "Use Cascades keys" : "Use raw keys"
            ActionBar.placement: ActionBarPlacement.OnBar
            onTriggered: { page.rawKeys = !page.rawKeys; page.applyKeys() }
        },
        ActionItem {
            title: "Esc"
            ActionBar.placement: ActionBarPlacement.OnBar
            onTriggered: page.send("button", { name: "esc" })
        }
    ]

    keyListeners: KeyListener {
        onKeyPressed: page.send("key", page.keyArgs(event, true))
        onKeyReleased: page.send("key", page.keyArgs(event, false))
    }

    Container {
        layout: DockLayout {}
        background: Color.Black

        ForeignWindowControl {
            id: fwc
            windowId: "term51-m0"
            // Position, Size and Visible: Cascades keeps the native window
            // over the control's area.
            updatedProperties: WindowProperty.Position | WindowProperty.Size | WindowProperty.Visible
            horizontalAlignment: HorizontalAlignment.Fill
            verticalAlignment: VerticalAlignment.Fill
            focusPolicy: FocusPolicy.KeyAndTouch
            onWindowAttached: page.send("attached", {})
            onBoundToWindowChanged: page.send("bound", { bound: boundToWindow })
            onTouch: page.send("touch", { type: event.touchType, x: event.localX, y: event.localY })
            attachedObjects: LayoutUpdateHandler {
                onLayoutFrameChanged: page.sized(layoutFrame)
            }
        }
    }

    onCreationCompleted: fwc.requestFocus()
}
