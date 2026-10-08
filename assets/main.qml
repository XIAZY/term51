import bb.cascades 1.4
import bb.system 1.2
import bb.multimedia 1.4

// The app's one screen, made of BlackBerry 10's own pieces as its browser
// is: a tabbed pane whose bar at the bottom is the system's action bar (the
// tabs button on the left, the three dots on the right) and whose tabs are
// in the system's drawer, which slides in from the left. The keys the
// phone's keyboard lacks are a slim strip of ours directly on top of the
// bar, in the bar's colour: the system's bar takes three buttons at most and
// repeats them in the menu of the three dots, where keys do not belong. There is one
// tab per shell and a "New Tab" row above them; all tabs show the same page,
// the terminal. The terminal itself is not drawn by Cascades: Go draws it
// with OpenGL into a window of its own, and the ForeignWindowControl below
// is the place in the page where that window shows.
TabbedPane {
    id: pane
    showTabsOnActionBar: false // the tabs are in the drawer
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
            property real padThreshold: 64   // the arrow pad's two numbers, from Go:
            property real padAxisRule: 1.5   // see internal/arrowpad
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
                    if (s.padThreshold !== undefined) page.padThreshold = s.padThreshold
                    if (s.padAxis !== undefined) page.padAxisRule = s.padAxis
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
                    if (e.pad !== undefined)
                        fwc.padSent = e.pad // an arrow key has gone out
                    if (e.notice !== undefined) {
                        toast.body = e.notice
                        toast.show()
                    }
                })
            }

            // The system's bar and the menu of its three dots. Paste is on
            // the bar too.
            actions: [
                ActionItem {
                    title: "Paste"
                    imageSource: "asset:///images/paste.png"
                    ActionBar.placement: ActionBarPlacement.OnBar
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
                    // Only this page gives it the focus (requestFocus, on a
                    // tap on the terminal): never a key, a touch or the
                    // phone's key navigation. Left to itself the field was
                    // the page's one target for the Enter key, so Enter on
                    // the physical keyboard brought the on-screen rows up,
                    // which no other key did.
                    focusPolicy: FocusPolicy.None
                    navigation.focusPolicy: NavigationFocusPolicy.NotFocusable
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
                Container {
                    horizontalAlignment: HorizontalAlignment.Fill
                    verticalAlignment: VerticalAlignment.Fill

                    ForeignWindowControl {
                        id: fwc
                        windowId: "term51"
                        // Under a sheet the terminal's window is taken off the screen:
                        // otherwise the phone blends the sheet's window with it on
                        // every frame, and the sheet scrolls slowly.
                        visible: !page.covered
                        updatedProperties: WindowProperty.Position | WindowProperty.Size | WindowProperty.Visible
                        horizontalAlignment: HorizontalAlignment.Fill
                        layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                        property real touchX: 0 // where the finger went down,
                        property real downY: 0  // for the press and hold
                        property real touchY: 0 // the drag so far
                        property real w: 0      // the control's size
                        property real h: 0

                        // The arrow pad. A press and hold brings it up near
                        // the finger. From then on the finger is a joystick
                        // whose middle is where it went down: Go is told its
                        // offset and sends the arrow keys (internal/arrowpad);
                        // this page only draws how far the finger is along
                        // which arrow, by the same two numbers Go uses. A
                        // finger that lifts without having swiped has
                        // selected the word under it instead, and gets the
                        // menu with Copy.
                        property bool padOn: false
                        property bool padSwiped: false  // the finger left the middle
                        property real padX: 0           // the pad's corner in the control
                        property real padY: 0
                        property string padDir: ""      // the arrow the finger is along,
                        property real padFill: 0        // and how far: 1 at the threshold
                        property string padAxis: ""     // "h" or "v" once past the threshold
                        property string padSent: ""     // the arrow whose key has gone out
                        property bool menuDue: false    // the next context menu is ours to show

                        function padMove(dx, dy) {
                            var ax = Math.abs(dx), ay = Math.abs(dy)
                            var horizontal = ax >= ay
                            if (padAxis == "h") horizontal = ay < page.padAxisRule * ax
                            else if (padAxis == "v") horizontal = ax >= page.padAxisRule * ay
                            var dist = horizontal ? ax : ay
                            var dir = horizontal ? (dx < 0 ? "left" : "right") : (dy < 0 ? "up" : "down")
                            padAxis = dist >= page.padThreshold ? (horizontal ? "h" : "v") : ""
                            if (dir != padDir || dist < page.padThreshold)
                                padSent = ""
                            padDir = dir
                            padFill = Math.min(dist / page.padThreshold, 1)
                            if (ax > 12 || ay > 12)
                                padSwiped = true
                            page.send("pad", { phase: "move", dx: dx, dy: dy })
                        }

                        attachedObjects: LayoutUpdateHandler {
                            onLayoutFrameChanged: {
                                fwc.w = layoutFrame.width
                                fwc.h = layoutFrame.height
                                page.sized(layoutFrame)
                            }
                        }
                        gestureHandlers: TapHandler {
                            onTapped: input.requestFocus()
                        }
                        contextMenuHandler: ContextMenuHandler {
                            // The system tells of a press and hold by being
                            // about to show the context menu. That menu is
                            // stopped and the arrow pad shown instead: 300
                            // pixels square, above the finger, or below it
                            // where there is no room above. The one menu let
                            // through is the one this page asks for itself
                            // when the finger lifts without a swipe.
                            onPopulating: {
                                if (fwc.menuDue) {
                                    fwc.menuDue = false
                                    return
                                }
                                event.abort()
                                fwc.padDir = ""
                                fwc.padFill = 0
                                fwc.padAxis = ""
                                fwc.padSent = ""
                                fwc.padSwiped = false
                                fwc.padX = Math.max(0, Math.min(fwc.touchX - 150, fwc.w - 300))
                                fwc.padY = fwc.downY - 300 - 60 >= 0 ? fwc.downY - 300 - 60 : fwc.downY + 60
                                fwc.padOn = true
                                page.send("pad", { phase: "hold" })
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
                            } else if (event.isMove() && fwc.padOn) {
                                fwc.padMove(event.localX - fwc.touchX, event.localY - fwc.downY)
                            } else if (event.isMove() && page.cellH > 0) {
                                var d = Math.floor((event.localY - fwc.touchY) / page.cellH)
                                if (d != 0) {
                                    page.send("scroll", { rows: -d })
                                    fwc.touchY += d * page.cellH
                                }
                            } else if ((event.isUp() || event.isCancel()) && fwc.padOn) {
                                fwc.padOn = false
                                page.send("pad", { phase: "lift" })
                                if (event.isUp() && !fwc.padSwiped && page.cellW > 0 && page.cellH > 0) {
                                    page.send("selectword", { x: Math.floor(fwc.touchX / page.cellW), y: Math.floor(fwc.downY / page.cellH) })
                                    fwc.menuDue = true
                                    fwc.showContextMenu()
                                }
                            }
                        }
                    }

                    // The keys the phone's keyboard lacks, but for the arrows,
                    // which are the arrow pad's: a strip directly on top of the
                    // system's bar and in its colour, with a hair-line above.
                    // Nothing on it takes the focus. Ctrl and Alt are sticky:
                    // they apply to the next key and stay lit until then.
                    Container {
                        background: Color.create("#383838")
                        preferredHeight: 2
                        horizontalAlignment: HorizontalAlignment.Fill
                    }
                    Container {
                        layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
                        background: Color.create("#262626")
                        preferredHeight: 84
                        horizontalAlignment: HorizontalAlignment.Fill
                        KeyCell { text: "Esc"
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("bar", { name: "esc" }) }
                        KeyCell { text: "Ctrl"; armed: page.ctrl
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("bar", { name: "ctrl" }) }
                        KeyCell { text: "Alt"; armed: page.alt
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("bar", { name: "alt" }) }
                        KeyCell { text: "Tab"
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("bar", { name: "tab" }) }
                    }
                }
                // The arrow pad's picture: its ground, the four arrows in
                // grey, and over each arrow the same arrow in white, cut off
                // so that only as much of it shows, from the pad's middle
                // outwards, as the finger has gone that way. The white turns
                // to the primary colour once the arrow's key has gone out.
                // It takes no touches: the finger stays the terminal's.
                Container {
                    visible: fwc.padOn
                    layout: AbsoluteLayout {}
                    preferredWidth: 300
                    preferredHeight: 300
                    translationX: fwc.padX
                    translationY: fwc.padY
                    horizontalAlignment: HorizontalAlignment.Left
                    verticalAlignment: VerticalAlignment.Top
                    touchPropagationMode: TouchPropagationMode.None
                    ImageView {
                        imageSource: "asset:///images/pad.png"
                    }
                    ImageView {
                        imageSource: "asset:///images/arrow-up.png"
                        filterColor: Color.create("#808080")
                        layoutProperties: AbsoluteLayoutProperties { positionX: 110; positionY: 18 }
                    }
                    Container {
                        layout: AbsoluteLayout {}
                        clipContentToBounds: true
                        preferredWidth: 81; preferredHeight: 81 * (fwc.padDir == "up" ? fwc.padFill : 0)
                        layoutProperties: AbsoluteLayoutProperties { positionX: 110; positionY: 18 + 81 * (1 - (fwc.padDir == "up" ? fwc.padFill : 0)) }
                        ImageView {
                            imageSource: "asset:///images/arrow-up.png"
                            filterColor: fwc.padSent == "up" ? ui.palette.primary : Color.White
                            preferredWidth: 81
                            preferredHeight: 81
                            layoutProperties: AbsoluteLayoutProperties { positionX: 0; positionY: -81 * (1 - (fwc.padDir == "up" ? fwc.padFill : 0)) }
                        }
                    }
                    ImageView {
                        imageSource: "asset:///images/arrow-down.png"
                        filterColor: Color.create("#808080")
                        layoutProperties: AbsoluteLayoutProperties { positionX: 110; positionY: 201 }
                    }
                    Container {
                        layout: AbsoluteLayout {}
                        clipContentToBounds: true
                        preferredWidth: 81; preferredHeight: 81 * (fwc.padDir == "down" ? fwc.padFill : 0)
                        layoutProperties: AbsoluteLayoutProperties { positionX: 110; positionY: 201 }
                        ImageView {
                            imageSource: "asset:///images/arrow-down.png"
                            filterColor: fwc.padSent == "down" ? ui.palette.primary : Color.White
                            preferredWidth: 81
                            preferredHeight: 81
                            layoutProperties: AbsoluteLayoutProperties { positionX: 0; positionY: 0 }
                        }
                    }
                    ImageView {
                        imageSource: "asset:///images/arrow-left.png"
                        filterColor: Color.create("#808080")
                        layoutProperties: AbsoluteLayoutProperties { positionX: 18; positionY: 110 }
                    }
                    Container {
                        layout: AbsoluteLayout {}
                        clipContentToBounds: true
                        preferredWidth: 81 * (fwc.padDir == "left" ? fwc.padFill : 0); preferredHeight: 81
                        layoutProperties: AbsoluteLayoutProperties { positionX: 18 + 81 * (1 - (fwc.padDir == "left" ? fwc.padFill : 0)); positionY: 110 }
                        ImageView {
                            imageSource: "asset:///images/arrow-left.png"
                            filterColor: fwc.padSent == "left" ? ui.palette.primary : Color.White
                            preferredWidth: 81
                            preferredHeight: 81
                            layoutProperties: AbsoluteLayoutProperties { positionX: -81 * (1 - (fwc.padDir == "left" ? fwc.padFill : 0)); positionY: 0 }
                        }
                    }
                    ImageView {
                        imageSource: "asset:///images/arrow-right.png"
                        filterColor: Color.create("#808080")
                        layoutProperties: AbsoluteLayoutProperties { positionX: 201; positionY: 110 }
                    }
                    Container {
                        layout: AbsoluteLayout {}
                        clipContentToBounds: true
                        preferredWidth: 81 * (fwc.padDir == "right" ? fwc.padFill : 0); preferredHeight: 81
                        layoutProperties: AbsoluteLayoutProperties { positionX: 201; positionY: 110 }
                        ImageView {
                            imageSource: "asset:///images/arrow-right.png"
                            filterColor: fwc.padSent == "right" ? ui.palette.primary : Color.White
                            preferredWidth: 81
                            preferredHeight: 81
                            layoutProperties: AbsoluteLayoutProperties { positionX: 0; positionY: 0 }
                        }
                    }
                }
            }
        }
    ]
}
