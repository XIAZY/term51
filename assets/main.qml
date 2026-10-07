import bb.cascades 1.4
import bb.system 1.2

// The terminal page, made of what BlackBerry 10 gives every app: the
// standard title bar, the terminal, and at the bottom a row of buttons for
// the keys the phone's keyboard lacks. The app's menu (a swipe down from the
// top edge) has the tabs, Paste and Settings. The terminal itself is not drawn by
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

    // The standard title bar: the shown tab's title and, with several
    // tabs, which one it is. There is no action bar: what would be in it is
    // in the app's menu.
    actionBarVisibility: ChromeVisibility.Hidden
    titleBar: TitleBar {
        kind: TitleBarKind.FreeForm
        kindProperties: FreeFormTitleBarKindProperties {
            Container {
                layout: DockLayout {}
                leftPadding: ui.du(3)
                rightPadding: ui.du(3)
                Label {
                    text: page.termTitle
                    textStyle.base: SystemDefaults.TextStyles.TitleText
                    horizontalAlignment: HorizontalAlignment.Left
                    verticalAlignment: VerticalAlignment.Center
                }
                Label {
                    visible: page.tabCount > 1
                    text: "tab " + (page.tabIndex + 1) + " of " + page.tabCount
                    textStyle.base: SystemDefaults.TextStyles.SubtitleText
                    textStyle.color: Color.Gray
                    horizontalAlignment: HorizontalAlignment.Right
                    verticalAlignment: VerticalAlignment.Center
                }
            }
        }
    }

    // The app's menu. BlackBerry 10 has icons of its own for closing and
    // for settings; the other three are ours (tools/icons).
    Menu.definition: MenuDefinition {
        actions: [
            ActionItem {
                title: "New tab"
                imageSource: "asset:///images/newtab.png"
                onTriggered: page.send("newtab")
            },
            DeleteActionItem {
                title: "Close tab"
                onTriggered: page.send("closetab")
            },
            ActionItem {
                // Until the tabs sheet is built: the next tab.
                title: "Tabs"
                imageSource: "asset:///images/tabs.png"
                enabled: page.tabCount > 1
                onTriggered: page.send("tab", { index: (page.tabIndex + 1) % page.tabCount })
            },
            ActionItem {
                title: "Paste"
                imageSource: "asset:///images/paste.png"
                onTriggered: {
                    var text = bridge.paste()
                    if (text.length > 0)
                        page.send("paste", { text: text })
                }
            }
        ]
        settingsAction: SettingsActionItem {
            enabled: false // until the settings page is built
        }
    }

    // Keys reach the shell in one of two ways. Until the terminal is
    // tapped nothing has the focus, and the page's key listener below sends
    // every key. A tap gives the hidden input field the focus, which makes
    // the phone show its on-screen keyboard rows (so that symbols can be
    // typed); then characters come through the field, its key listener
    // sends only the keys that are not characters (Backspace, Tab, Escape,
    // the arrows), and Enter is the field's "submitted".
    function isCharacterKey(event) {
        var k = event.key
        if (k == 13 || k == 10 || k == 0xF00D)
            return true // Enter: handled by the input field
        return k >= 0x20 && k != 0x7F && k < 0xF000
    }
    keyListeners: KeyListener {
        onKeyPressed: {
            if (!input.focused)
                page.send("key", page.keyArgs(event, true))
        }
        onKeyReleased: {
            if (!input.focused)
                page.send("key", page.keyArgs(event, false))
        }
    }
    attachedObjects: SystemToast { id: toast }

    Container {
        layout: StackLayout {}
        background: Color.create("#101418")

        // The terminal. Position, Size and Visible: Cascades keeps Go's
        // window over this control's area. A vertical drag scrolls the
        // scrollback; a long press selects the word under the finger; a
        // tap brings up the on-screen keyboard rows (the phone's own swipe
        // down puts them away).
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
            gestureHandlers: [
                LongPressHandler {
                    onLongPressed: {
                        if (page.cellW > 0 && page.cellH > 0)
                            page.send("selectword", { x: Math.floor(event.x / page.cellW), y: Math.floor(event.y / page.cellH) })
                    }
                },
                TapHandler {
                    onTapped: input.requestFocus()
                }
            ]
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

        // The key row, the last thing on the page (right above the phone's
        // on-screen keyboard rows when those are up). Its buttons never take
        // the focus: a focused button would take the next Enter or Space as
        // a click. Ctrl and Alt are sticky: they apply to the next key and
        // show in the primary colour until then.
        Container {
            layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
            horizontalAlignment: HorizontalAlignment.Fill

            Button { text: "Esc"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                     focusPolicy: FocusPolicy.None
                     onClicked: page.send("bar", { name: "esc" }) }
            Button { text: "Ctrl"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                     focusPolicy: FocusPolicy.None
                     appearance: page.ctrl ? ControlAppearance.Primary : ControlAppearance.Default
                     onClicked: page.send("bar", { name: "ctrl" }) }
            Button { text: "Alt"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                     focusPolicy: FocusPolicy.None
                     appearance: page.alt ? ControlAppearance.Primary : ControlAppearance.Default
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

        // The input field: while it has the focus the phone shows its
        // on-screen keyboard rows, as for any text field. What is typed
        // goes to the shell at once and the field is emptied. It is not
        // seen and takes no room.
        TextField {
            id: input
            opacity: 0
            maxHeight: 0
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
