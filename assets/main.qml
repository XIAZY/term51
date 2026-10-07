import bb.cascades 1.4
import bb.system 1.2
import bb.multimedia 1.4

// The app's one screen, made of BlackBerry 10's own pieces as its browser
// is: a tabbed pane whose bar at the bottom is the system's action bar (the
// tabs button on the left, the three dots on the right, between them the
// keys the phone's keyboard lacks as the bar's own buttons) and whose tabs
// are in the system's drawer, which slides in from the left. There is one
// tab per shell and a "New Tab" row above them; all tabs show the same page,
// the terminal. The terminal itself is not drawn by Cascades: Go draws it
// with OpenGL into a window of its own, and the ForeignWindowControl below
// is the place in the page where that window shows.
TabbedPane {
    id: pane
    showTabsOnActionBar: false // the keys are on the bar; the tabs in the drawer
    peekEnabled: false         // a sideways drag belongs to the terminal

    // The tabs follow Go's list of shells: tab 0 is "New Tab", tab i + 1 is
    // shell i. Tabs are added and taken away at the end only, and retitled,
    // so that an open drawer is not rebuilt.
    function syncTabs(titles, rang, current) {
        while (pane.count() - 1 < titles.length) {
            var made = shellTab.createObject()
            made.shell = pane.count() - 1
            pane.add(made)
        }
        if (current >= 0 && current < titles.length)
            pane.activeTab = pane.at(current + 1)
        while (pane.count() - 1 > titles.length) {
            var last = pane.at(pane.count() - 1)
            pane.remove(last)
            last.destroy()
        }
        for (var i = 0; i < titles.length; i++) {
            var tab = pane.at(i + 1)
            tab.title = titles[i]
            tab.newContentAvailable = rang[i] // the system's mark: its bell rang
        }
    }

    Tab {
        title: "New Tab"
        imageSource: "asset:///images/newtab.png"
        // Go answers with the new list, and syncTabs shows the new shell's
        // tab as the active one.
        onTriggered: page.send("newtab")
    }

    onCreationCompleted: pane.activePane = page

    attachedObjects: [
        ComponentDefinition {
            id: shellTab
            Tab {
                property int shell: 0 // which shell, counted from 0
                imageSource: "asset:///images/tabs.png"
                onTriggered: page.send("tab", { index: shell })
            }
        },
        SystemToast { id: toast },
        SystemSound { id: bellSound; sound: SystemSound.GeneralNotification },
        SettingsSheet {
            id: settingsSheet
            onOpened: page.covered = true
            onClosed: page.covered = false
        },

        // The terminal page, shown for every tab.
        Page {
            id: page

            property bool ctrl: false
            property bool alt: false
            property bool created: false     // the terminal's window has been asked for
            property string termBackground: "#101418" // the theme's, around the terminal
            property bool bell: true         // a program's bell makes a sound
            property bool covered: false     // the settings sheet is over the terminal
            property real cellW: 0           // a cell in pixels, from Go
            property real cellH: 0

            function send(name, args) { bridge.send(name, JSON.stringify(args || {})) }

            // Types the clipboard's text into the shell.
            function pasteClipboard() {
                var text = bridge.paste()
                if (text.length > 0)
                    send("paste", { text: text })
            }

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
                    if (s.cw !== undefined) page.cellW = s.cw
                    if (s.ch !== undefined) page.cellH = s.ch
                    if (s.bell !== undefined) page.bell = s.bell
                    if (s.settings !== undefined) page.termBackground = s.settings["theme.background"]
                    if (s.tabs !== undefined) pane.syncTabs(s.tabs, s.rang, s.current)
                })
                bridge.event.connect(function(json) {
                    var e = JSON.parse(json)
                    if (e.clipboard !== undefined) {
                        bridge.copy(e.clipboard)
                        return
                    }
                    if (e.bell !== undefined && page.bell)
                        bellSound.play()
                    if (e.notice !== undefined) {
                        toast.body = e.notice
                        toast.show()
                    }
                })
            }

            // The system's bar: the keys as its buttons, in this order as
            // far as they fit (what does not fit, the system puts into the
            // menu of the three dots, before the four below). Ctrl and Alt
            // are sticky: they apply to the next key, and their icon is in
            // the primary colour until then.
            actions: [
                ActionItem {
                    title: "Esc"
                    imageSource: "asset:///images/key-esc.png"
                    ActionBar.placement: ActionBarPlacement.OnBar
                    onTriggered: page.send("bar", { name: "esc" })
                },
                ActionItem {
                    title: "Ctrl"
                    imageSource: page.ctrl ? "asset:///images/key-ctrl-armed.png" : "asset:///images/key-ctrl.png"
                    ActionBar.placement: ActionBarPlacement.OnBar
                    onTriggered: page.send("bar", { name: "ctrl" })
                },
                ActionItem {
                    title: "Tab"
                    imageSource: "asset:///images/key-tab.png"
                    ActionBar.placement: ActionBarPlacement.OnBar
                    onTriggered: page.send("bar", { name: "tab" })
                },
                ActionItem {
                    title: "Left"
                    imageSource: "asset:///images/key-left.png"
                    ActionBar.placement: ActionBarPlacement.OnBar
                    onTriggered: page.send("bar", { name: "left" })
                },
                ActionItem {
                    title: "Down"
                    imageSource: "asset:///images/key-down.png"
                    ActionBar.placement: ActionBarPlacement.OnBar
                    onTriggered: page.send("bar", { name: "down" })
                },
                ActionItem {
                    title: "Up"
                    imageSource: "asset:///images/key-up.png"
                    ActionBar.placement: ActionBarPlacement.OnBar
                    onTriggered: page.send("bar", { name: "up" })
                },
                ActionItem {
                    title: "Right"
                    imageSource: "asset:///images/key-right.png"
                    ActionBar.placement: ActionBarPlacement.OnBar
                    onTriggered: page.send("bar", { name: "right" })
                },
                ActionItem {
                    title: "Alt"
                    imageSource: page.alt ? "asset:///images/key-alt-armed.png" : "asset:///images/key-alt.png"
                    ActionBar.placement: ActionBarPlacement.OnBar
                    onTriggered: page.send("bar", { name: "alt" })
                },
                ActionItem {
                    title: "Paste"
                    imageSource: "asset:///images/paste.png"
                    ActionBar.placement: ActionBarPlacement.InOverflow
                    onTriggered: page.pasteClipboard()
                },
                ActionItem {
                    title: "Select All"
                    imageSource: "asset:///images/selectall.png"
                    ActionBar.placement: ActionBarPlacement.InOverflow
                    onTriggered: page.send("selectall")
                },
                ActionItem {
                    title: "Close Tab"
                    imageSource: "asset:///images/closetab.png"
                    ActionBar.placement: ActionBarPlacement.InOverflow
                    onTriggered: page.send("closetab")
                },
                ActionItem {
                    title: "Settings"
                    imageSource: "asset:///images/settings.png"
                    ActionBar.placement: ActionBarPlacement.InOverflow
                    onTriggered: settingsSheet.open()
                }
            ]

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
            Container {
                layout: DockLayout {}
                background: Color.create(page.termBackground)

                // The input field: while it has the focus the phone shows its
                // on-screen keyboard rows, as for any text field. What is typed
                // goes to the shell at once and the field is emptied. It is not
                // seen, and lies under the rest of the page so that it takes no room
                // (in the column it kept its height although it was told to have
                // none).
                TextField {
                    id: input
                    opacity: 0
                    verticalAlignment: VerticalAlignment.Bottom
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

                // The terminal. Position, Size and Visible: Cascades keeps Go's
                // window over this control's area. A vertical drag scrolls the
                // scrollback. A press and hold selects the word under the finger
                // and opens the standard context menu (on text that is already
                // selected it keeps the selection). A tap brings up the on-screen
                // keyboard rows (the phone's own swipe down puts them away).
                ForeignWindowControl {
                    id: fwc
                    windowId: "term51"
                    // Under a sheet the terminal's window is taken off the screen:
                    // otherwise the phone blends the sheet's window with it on
                    // every frame, and the sheet scrolls slowly.
                    visible: !page.covered
                    updatedProperties: WindowProperty.Position | WindowProperty.Size | WindowProperty.Visible
                    horizontalAlignment: HorizontalAlignment.Fill
                    verticalAlignment: VerticalAlignment.Fill
                    property real touchX: 0 // where the finger went down,
                    property real downY: 0  // for the press and hold
                    property real touchY: 0 // the drag so far
                    attachedObjects: LayoutUpdateHandler {
                        onLayoutFrameChanged: page.sized(layoutFrame)
                    }
                    gestureHandlers: TapHandler {
                        onTapped: input.requestFocus()
                    }
                    contextMenuHandler: ContextMenuHandler {
                        onPopulating: {
                            if (page.cellW > 0 && page.cellH > 0)
                                page.send("selectword", { x: Math.floor(fwc.touchX / page.cellW), y: Math.floor(fwc.downY / page.cellH) })
                        }
                    }
                    contextActions: ActionSet {
                        title: "Selection"
                        ActionItem {
                            title: "Copy"
                            imageSource: "asset:///images/copy.png"
                            onTriggered: page.send("copy")
                        }
                        ActionItem {
                            title: "Select All"
                            imageSource: "asset:///images/selectall.png"
                            onTriggered: page.send("selectall")
                        }
                        ActionItem {
                            title: "Paste"
                            imageSource: "asset:///images/paste.png"
                            onTriggered: {
                                page.send("unselect")
                                page.pasteClipboard()
                            }
                        }
                        ActionItem {
                            title: "Cancel Selection"
                            imageSource: "asset:///images/cancel.png"
                            onTriggered: page.send("unselect")
                        }
                    }
                    onTouch: {
                        if (event.isDown()) {
                            fwc.touchX = event.localX
                            fwc.downY = event.localY
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
            }
        }
    ]
}
