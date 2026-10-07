import bb.cascades 1.4
import bb.system 1.2

// The terminal page: a key bar with the keys the phone's keyboard lacks, the
// terminal, and the input field. The terminal itself is not drawn by
// Cascades: Go draws it with OpenGL into a window of its own, and the
// ForeignWindowControl below is the place in the page where that window
// shows.
Page {
    id: page

    property bool ctrl: false
    property bool alt: false
    property string termTitle: "term51"
    property int tabCount: 1
    property int tabIndex: 0

    property bool created: false     // the terminal's window has been asked for
    property real cellW: 0           // a cell in pixels, from Go
    property real cellH: 0

    function send(name, args) { bridge.send(name, JSON.stringify(args || {})) }

    function keyArgs(event, down) {
        return { down: down, key: event.key, keycap: event.keycap, unicode: event.unicode,
                 shift: event.shiftPressed, ctrl: event.ctrlPressed, alt: event.altPressed }
    }

    // Tells Go where the terminal goes and how big it is: the first time
    // it creates the window, afterwards it resizes it.
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

    onCreationCompleted: {
        input.requestFocus()
        bridge.state.connect(function(json) {
            var s = JSON.parse(json)
            if (s.ctrl !== undefined) page.ctrl = s.ctrl
            if (s.alt !== undefined) page.alt = s.alt
            if (s.title !== undefined) page.termTitle = s.title
            if (s.cw !== undefined) page.cellW = s.cw
            if (s.ch !== undefined) page.cellH = s.ch
            if (s.tabs !== undefined) page.tabCount = s.tabs.length
            if (s.current !== undefined) page.tabIndex = s.current
        })
        bridge.event.connect(function(json) {
            var e = JSON.parse(json)
            if (e.clipboard !== undefined) {
                bridge.copy(e.clipboard)
                return
            }
            if (e.notice !== undefined) {
                toast.body = e.notice
                toast.show()
            }
        })
    }

    // With several tabs the title says which one is shown.
    titleBar: TitleBar {
        title: page.tabCount > 1 ? (page.tabIndex + 1) + "/" + page.tabCount + "  " + page.termTitle : page.termTitle
    }

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
            title: "New tab"
            onTriggered: page.send("newtab")
        },
        ActionItem {
            title: "Next tab"
            enabled: page.tabCount > 1
            onTriggered: page.send("tab", { index: (page.tabIndex + 1) % page.tabCount })
        },
        ActionItem {
            title: "Close tab"
            onTriggered: page.send("closetab")
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

    // Characters come through the input field at the bottom of the page (so
    // that the phone shows its on-screen keyboard rows, and symbols can be
    // typed). The field always has the focus; its key listener sends only
    // the keys that are not characters: Backspace, Tab, Escape, the arrows.
    // Enter is the field's "submitted".
    function isCharacterKey(event) {
        var k = event.key
        if (k == 13 || k == 10 || k == 0xF00D)
            return true // Enter: handled by the input field
        return k >= 0x20 && k != 0x7F && k < 0xF000
    }
    attachedObjects: SystemToast { id: toast }

    Container {
        layout: StackLayout {}
        background: Color.create("#101418")

        // The key bar. Its buttons never take the focus: a focused button
        // would take the next Enter or Space as a click. Ctrl and Alt are sticky: they apply to the next key
        // and show as armed until then.
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

        // The terminal. Position, Size and Visible: Cascades keeps Go's
        // window over this control's area. A vertical drag scrolls the
        // scrollback; a long press selects the word under the finger.
        ForeignWindowControl {
            id: fwc
            windowId: "term51"
            updatedProperties: WindowProperty.Position | WindowProperty.Size | WindowProperty.Visible
            horizontalAlignment: HorizontalAlignment.Fill
            layoutProperties: StackLayoutProperties { spaceQuota: 1 }
            property real touchY: 0
            attachedObjects: LayoutUpdateHandler {
                onLayoutFrameChanged: page.sized(layoutFrame)
            }
            gestureHandlers: LongPressHandler {
                onLongPressed: {
                    if (page.cellW > 0 && page.cellH > 0)
                        page.send("selectword", { x: Math.floor(event.x / page.cellW), y: Math.floor(event.y / page.cellH) })
                }
            }
            onTouch: {
                if (event.isDown()) {
                    fwc.touchY = event.localY
                } else if (event.isMove() && page.cellH > 0) {
                    var d = Math.floor((event.localY - fwc.touchY) / page.cellH)
                    if (d != 0) {
                        page.send("scroll", { rows: -d })
                        fwc.touchY += d * page.cellH
                    }
                }
            }
        }

        // The input field: it holds the focus, so the phone shows its
        // on-screen keyboard rows as for any text field. What is typed goes
        // to the shell at once and the field is emptied.
        TextField {
            id: input
            hintText: "type here"
            horizontalAlignment: HorizontalAlignment.Fill
            inputMode: TextFieldInputMode.Text
            input.flags: TextInputFlag.AutoCapitalizationOff | TextInputFlag.AutoCorrectionOff | TextInputFlag.PredictionOff | TextInputFlag.SpellCheckOff | TextInputFlag.WordSubstitutionOff | TextInputFlag.AutoPeriodOff
            input.submitKey: SubmitKey.EnterKey
            input.submitKeyFocusBehavior: SubmitKeyFocusBehavior.Keep
            input.onSubmitted: page.send("key", { down: true, key: 13, keycap: 13, unicode: "", shift: false, ctrl: false, alt: false })
            onTextChanging: {
                if (text.length > 0) {
                    page.send("text", { text: text })
                    input.text = ""
                }
            }
            keyListeners: KeyListener {
                onKeyPressed: {
                    if (!page.isCharacterKey(event))
                        page.send("key", page.keyArgs(event, true))
                }
            }
        }
    }
}
