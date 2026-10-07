import bb.cascades 1.4
import bb.system 1.2

// The terminal page: a key bar with the keys the phone's keyboard lacks, and
// below it the native window Go draws the terminal into.
//
// Go posts its state (the title, the size in cells, which sticky modifiers
// are armed) as JSON through `bridge`; key presses and the key bar go back
// as commands.
Page {
    id: page

    property bool created: false
    property bool ctrl: false
    property bool alt: false
    property string termTitle: "term51"

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

    function keyArgs(event, down) {
        return { down: down, key: event.key, keycap: event.keycap, unicode: event.unicode,
                 shift: event.shiftPressed, ctrl: event.ctrlPressed, alt: event.altPressed }
    }

    onCreationCompleted: {
        bridge.state.connect(function(json) {
            var s = JSON.parse(json)
            if (s.ctrl !== undefined) page.ctrl = s.ctrl
            if (s.alt !== undefined) page.alt = s.alt
            if (s.title !== undefined) page.termTitle = s.title
            // Raw keys: the control forwards Screen keyboard events to the
            // native window (InputType.Keyboard = 1), where Go reads them.
            if (s.rawKeys !== undefined) {
                fwc.forwardedInputTypes = s.rawKeys ? 1 : 0
                fwc.requestFocus()
            }
        })
        bridge.event.connect(function(json) {
            var e = JSON.parse(json)
            if (e.clipboard !== undefined)
                bridge.copy(e.clipboard)
            if (e.notice !== undefined) {
                toast.body = e.notice
                toast.show()
            }
        })
        fwc.requestFocus()
    }

    attachedObjects: SystemToast { id: toast }

    titleBar: TitleBar { title: page.termTitle }

    // Copy takes the selection (long press a word, or Select all). Go
    // answers with the text; the page owns the clipboard.
    actions: [
        ActionItem {
            title: "Copy"
            onTriggered: page.send("copy")
        },
        ActionItem {
            title: "Paste"
            onTriggered: {
                var text = bridge.paste()
                if (text.length > 0)
                    page.send("paste", { text: text })
            }
        },
        ActionItem {
            title: "Select all"
            onTriggered: page.send("selectall")
        },
        ActionItem {
            title: "Clear selection"
            onTriggered: page.send("unselect")
        }
    ]

    // The phone's keyboard: every key press and release goes to Go.
    keyListeners: KeyListener {
        onKeyPressed: page.send("key", page.keyArgs(event, true))
        onKeyReleased: page.send("key", page.keyArgs(event, false))
    }

    Container {
        layout: DockLayout {}
        background: Color.Black

        Container {
            horizontalAlignment: HorizontalAlignment.Fill
            verticalAlignment: VerticalAlignment.Fill
            layout: StackLayout {}

            // The key bar. Its buttons never take the focus: a focused button
            // would take the next Enter or Space as a click. Ctrl and Alt are sticky: they apply to the next
            // key and show as armed until then.
            Container {
                layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
                horizontalAlignment: HorizontalAlignment.Fill
                background: Color.create("#202428")

                Button { text: "Esc"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                         focusPolicy: FocusPolicy.None
                         onClicked: page.send("bar", { name: "esc" }) }
                Button { text: page.ctrl ? "[Ctrl]" : "Ctrl"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                         focusPolicy: FocusPolicy.None
                         onClicked: page.send("bar", { name: "ctrl" }) }
                Button { text: page.alt ? "[Alt]" : "Alt"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                         focusPolicy: FocusPolicy.None
                         onClicked: page.send("bar", { name: "alt" }) }
                Button { text: "Tab"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                         focusPolicy: FocusPolicy.None
                         onClicked: page.send("bar", { name: "tab" }) }
                Button { text: "←"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                         focusPolicy: FocusPolicy.None
                         onClicked: page.send("bar", { name: "left" }) }
                Button { text: "↓"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                         focusPolicy: FocusPolicy.None
                         onClicked: page.send("bar", { name: "down" }) }
                Button { text: "↑"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                         focusPolicy: FocusPolicy.None
                         onClicked: page.send("bar", { name: "up" }) }
                Button { text: "→"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                         focusPolicy: FocusPolicy.None
                         onClicked: page.send("bar", { name: "right" }) }
            }

            // The terminal. Position, Size and Visible: Cascades keeps the
            // native window over this control's area. Touch on it goes to the
            // native window (a drag scrolls the scrollback).
            ForeignWindowControl {
                id: fwc
                windowId: "term51"
                updatedProperties: WindowProperty.Position | WindowProperty.Size | WindowProperty.Visible
                horizontalAlignment: HorizontalAlignment.Fill
                layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                focusPolicy: FocusPolicy.KeyAndTouch
                attachedObjects: LayoutUpdateHandler {
                    onLayoutFrameChanged: page.sized(layoutFrame)
                }
            }
        }
    }
}
