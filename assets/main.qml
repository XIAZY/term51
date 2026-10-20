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
            onOpened: {
                page.covered = true
                // Is there a newer PowerShell than the one that is here?
                page.send("pwsh", { action: "check" })
            }
            onClosed: page.covered = false
        },

        // The terminal page, shown for every tab.
        Page {
            id: page

            property bool ctrl: false
            property bool alt: false
            property bool letGo: false       // the input field is letting go of the focus by our own call
            property bool created: false     // the terminal's window has been asked for
            property string termBackground: "#101418" // the theme's, around the terminal
            property bool bell: true         // a program's bell makes a sound
            property bool covered: false     // the settings sheet is over the terminal
            // What Go says of the selection with every frame: "" for none,
            // "open" for one that a drag may still stretch, "done" for one
            // that has been copied and only stays shown until the next touch.
            property string selected: ""
            property real selBarX: 0         // where the selection's bar goes
            property real selBarY: 0
            property bool selStartOn: false  // the selection's two ends: whether
            property int selStartX: 0        // each is in the view, and its cell
            property int selStartY: 0
            property bool selEndOn: false
            property int selEndX: 0
            property int selEndY: 0
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
                    if (s.selected !== undefined) page.selected = s.selected
                    if (s.selBarX !== undefined) page.selBarX = s.selBarX
                    if (s.selBarY !== undefined) page.selBarY = s.selBarY
                    if (s.selStartOn !== undefined) page.selStartOn = s.selStartOn
                    if (s.selStartX !== undefined) page.selStartX = s.selStartX
                    if (s.selStartY !== undefined) page.selStartY = s.selStartY
                    if (s.selEndOn !== undefined) page.selEndOn = s.selEndOn
                    if (s.selEndX !== undefined) page.selEndX = s.selEndX
                    if (s.selEndY !== undefined) page.selEndY = s.selEndY
                    if (s.padThreshold !== undefined) page.padThreshold = s.padThreshold
                    if (s.padAxis !== undefined) page.padAxisRule = s.padAxis
                    if (s.ch !== undefined) page.cellH = s.ch
                    if (s.bell !== undefined) page.bell = s.bell
                    if (s.settings !== undefined) page.termBackground = s.settings["theme.background"]
                    if (s.pwsh !== undefined) settingsSheet.pwsh(s.pwsh)
                    if (s.drawing !== undefined) settingsSheet.drawing = s.drawing
                    if (s.es3 !== undefined) settingsSheet.es3 = s.es3
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

            // The menu of the system bar's three dots, and after it the three
            // keys that are on the bar itself, which is all the bar takes:
            // Esc, Ctrl and Tab. The system lists what is on the bar in the
            // menu as well; they come last here so as to be out of the way
            // there. Each key's picture is a sign and its title the name. The
            // sticky Ctrl is in the primary colour while it is armed.
            actions: [
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
                    // The input field lets go first: it would take the focus
                    // back from the sheet's fields otherwise.
                    onTriggered: { page.letGoOfFocus(); settingsSheet.open() }
                },
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
                }
            ]

            // The input field gives up the focus only when this page lets
            // it go (a tap on the terminal with the rows up, the settings
            // sheet); a loss to anything else (see the input field for why
            // a touch hardly ever takes it) is undone at the moment it
            // happens.
            function letGoOfFocus() {
                if (input.focused) {
                    page.letGo = true
                    input.loseFocus()
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
                    // A touch takes the focus from it only when it lands on a
                    // control that takes the focus itself, which nothing on
                    // this page does: our key strip, the system's bar (its
                    // buttons work without the focus), the terminal and the
                    // selection leave it, and the on-screen rows, where they
                    // are. By the phone's default rule any touch elsewhere
                    // took it, a quick press with the finger sliding a little
                    // as well (as scrolling), and the rows went down with it.
                    focusRetentionPolicyFlags: FocusRetentionPolicy.LoseToFocusable
                    verticalAlignment: VerticalAlignment.Bottom
                    horizontalAlignment: HorizontalAlignment.Fill
                    inputMode: TextFieldInputMode.Text
                    input.flags: TextInputFlag.AutoCapitalizationOff | TextInputFlag.AutoCorrectionOff | TextInputFlag.PredictionOff | TextInputFlag.SpellCheckOff | TextInputFlag.WordSubstitutionOff | TextInputFlag.AutoPeriodOff
                    input.submitKey: SubmitKey.EnterKey
                    input.submitKeyFocusBehavior: SubmitKeyFocusBehavior.Keep
                    onFocusedChanged: {
                        if (focused)
                            return
                        if (page.letGo)
                            page.letGo = false
                        else
                            input.requestFocus()
                    }
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
                // (on text that is already selected it keeps the selection),
                // and a selection has its bar of buttons. A tap brings up the on-screen
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
                        // The largest height it has had at this width: the
                        // on-screen rows are up while it is clearly less (the
                        // phone does not say whether they show).
                        property real fullH: 0
                        function rowsUp() { return h < fullH - 10 }

                        // The arrow pad. A press and hold brings it up near
                        // the finger. From then on the finger is a joystick
                        // whose middle is where it went down: Go is told its
                        // offset and sends the arrow keys (internal/arrowpad);
                        // this page only draws how far the finger is along
                        // which arrow, by the same two numbers Go uses. A
                        // finger that lifts without having swiped has
                        // selected the word under it instead, and gets the
                        // selection's bar with Copy.
                        property bool padOn: false
                        property bool padSwiped: false  // the finger left the middle
                        property real padX: 0           // the pad's corner in the control
                        property real padY: 0
                        property string padDir: ""      // the arrow the finger is along,
                        property real padFill: 0        // and how far: 1 at the threshold
                        property string padAxis: ""     // "h" or "v" once past the threshold
                        property string padSent: ""     // the arrow whose key has gone out

                        // Stretching the selection. While there is one that
                        // has not been copied, a drag does not scroll: the
                        // finger moves the selection's nearer end, and Go is
                        // told the cell it came down on and each cell it is
                        // over (internal/session, internal/stretch for the
                        // scrolling at the edges). The selection's bar is
                        // away meanwhile. A copied selection is dropped by the
                        // first drag, which then scrolls as usual.
                        property bool stretching: false
                        property bool dropped: false    // this drag has dropped a copied selection
                        function stretch(phase, x, y) {
                            page.send("stretch", { phase: phase, x: x / page.cellW, y: y / page.cellH })
                        }

                        // The selection's handles: a dot hanging under each
                        // end, on a thin line from the top of the end's row.
                        // A dot that is dragged stretches the selection by
                        // its own end: Go is told the finger came down on
                        // that end's cell, and then the cell that is as far
                        // from there as the finger is from where it took the
                        // dot, so the end stays above the finger and the
                        // text in sight. The dot hangs 100 pixels under its
                        // row; less under the last rows, where it would
                        // leave the page: there it lies on the key bar.
                        property real grabX: 0  // the grabbed end's cell, its middle
                        property real grabY: 0
                        property real dragX: 0  // where the drag has taken it
                        property real dragY: 0
                        function hang(row) {
                            return Math.min(100, fwc.h + 86 - 22 - (row + 1) * page.cellH)
                        }
                        function handleGrabbed(col, row) {
                            grabX = (col + 0.5) * page.cellW
                            grabY = (row + 0.5) * page.cellH
                            dragX = grabX
                            dragY = grabY
                            stretching = true
                            stretch("down", grabX, grabY)
                        }
                        function handleDragged(dx, dy) {
                            dragX = grabX + dx
                            dragY = grabY + dy
                            stretch("move", dragX, dragY)
                        }
                        function handleDropped() {
                            stretching = false
                            stretch("up", dragX, dragY)
                        }

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
                                if (layoutFrame.width != fwc.w || layoutFrame.height > fwc.fullH)
                                    fwc.fullH = layoutFrame.height // a new width (rotation) starts over
                                fwc.w = layoutFrame.width
                                fwc.h = layoutFrame.height
                                page.sized(layoutFrame)
                            }
                        }
                        gestureHandlers: TapHandler {
                            // A tap brings the on-screen rows up, and puts them
                            // away when they are up; but while something is
                            // selected it only drops the selection, and leaves
                            // the rows as they are. The phone's swipe down puts
                            // the rows away and leaves the input field the
                            // focus, which it then gives up and asks for
                            // again, so that the rows show.
                            onTapped: {
                                if (page.selected != "")
                                    page.send("unselect")
                                else if (fwc.rowsUp())
                                    page.letGoOfFocus()
                                else {
                                    page.letGoOfFocus()
                                    input.requestFocus()
                                }
                            }
                        }
                        contextMenuHandler: ContextMenuHandler {
                            // The system tells of a press and hold by being
                            // about to show the context menu. That menu is
                            // always stopped, and the arrow pad shown instead:
                            // 300 pixels square, above the finger, or below it
                            // where there is no room above.
                            onPopulating: {
                                event.abort()
                                if (fwc.stretching)
                                    return // a finger resting while it stretches
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
                        // Never shown: the handler above stops every menu.
                        // It is here because a hold is known to be told for a
                        // control that has actions for the context menu, and
                        // not known to be told for one that has none.
                        contextActions: ActionSet {
                            ActionItem {
                                title: "Copy"
                                onTriggered: page.send("copy")
                            }
                        }
                        onTouch: {
                            if (event.isDown()) {
                                fwc.touchX = event.localX
                                fwc.downY = event.localY
                                fwc.touchY = event.localY
                                fwc.dropped = false
                            } else if (event.isMove() && fwc.padOn) {
                                fwc.padMove(event.localX - fwc.touchX, event.localY - fwc.downY)
                            } else if (event.isMove() && page.cellH > 0 && (fwc.stretching || page.selected == "open")) {
                                // A finger that has hardly moved may still
                                // become a press and hold.
                                if (!fwc.stretching && Math.abs(event.localX - fwc.touchX) <= 12 && Math.abs(event.localY - fwc.downY) <= 12)
                                    return
                                if (!fwc.stretching) {
                                    fwc.stretching = true
                                    fwc.stretch("down", fwc.touchX, fwc.downY)
                                }
                                fwc.stretch("move", event.localX, event.localY)
                            } else if ((event.isUp() || event.isCancel()) && fwc.stretching) {
                                fwc.stretching = false
                                fwc.stretch("up", event.localX, event.localY)
                            } else if (event.isMove() && page.cellH > 0) {
                                if (page.selected == "done" && !fwc.dropped) {
                                    fwc.dropped = true
                                    page.send("unselect")
                                }
                                var d = Math.floor((event.localY - fwc.touchY) / page.cellH)
                                if (d != 0) {
                                    page.send("scroll", { rows: -d, x: Math.round(event.localX), y: Math.round(event.localY) })
                                    fwc.touchY += d * page.cellH
                                }
                            } else if ((event.isUp() || event.isCancel()) && fwc.padOn) {
                                fwc.padOn = false
                                page.send("pad", { phase: "lift" })
                                if (event.isUp() && !fwc.padSwiped && page.cellW > 0 && page.cellH > 0)
                                    page.send("selectword", { x: Math.floor(fwc.touchX / page.cellW), y: Math.floor(fwc.downY / page.cellH) })
                            }
                        }
                    }

                    // The key bar, directly on top of the system's bar and in
                    // its colour, with a hair-line above: Alt, and six
                    // characters that a shell needs all the time and the
                    // keyboard only has behind its symbol key. (Esc, Ctrl and
                    // Tab are on the system's bar; the arrows are the arrow
                    // pad's.) Nothing on it takes the focus. Alt and Ctrl are
                    // sticky: they apply to the next key, one of these
                    // characters included, and Alt stays lit until then.
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
                        KeyCell { text: "Alt"; armed: page.alt
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("bar", { name: "alt" }) }
                        KeyCell { text: "-"; symbol: true
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("text", { text: "-" }) }
                        KeyCell { text: "_"; symbol: true
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("text", { text: "_" }) }
                        KeyCell { text: "/"; symbol: true
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("text", { text: "/" }) }
                        KeyCell { text: "$"; symbol: true
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("text", { text: "$" }) }
                        KeyCell { text: "|"; symbol: true
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("text", { text: "|" }) }
                        KeyCell { text: "~"; symbol: true
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("text", { text: "~" }) }
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
                // The selection's two handles (see the terminal's
                // handleGrabbed): for each end that is in the view a line,
                // which takes no touches, and its dot. They show for as long
                // as the selection is open, also while it is stretched.
                Container {
                    visible: page.selected == "open" && page.selStartOn && !fwc.padOn
                    background: Color.create("#00a8df")
                    preferredWidth: 2
                    preferredHeight: page.cellH + fwc.hang(page.selStartY)
                    translationX: page.selStartX * page.cellW - 1
                    translationY: page.selStartY * page.cellH
                    horizontalAlignment: HorizontalAlignment.Left
                    verticalAlignment: VerticalAlignment.Top
                    touchPropagationMode: TouchPropagationMode.None
                }
                SelHandle {
                    visible: page.selected == "open" && page.selStartOn && !fwc.padOn
                    translationX: page.selStartX * page.cellW - 50
                    translationY: (page.selStartY + 1) * page.cellH + fwc.hang(page.selStartY) - 50
                    horizontalAlignment: HorizontalAlignment.Left
                    verticalAlignment: VerticalAlignment.Top
                    onGrabbed: fwc.handleGrabbed(page.selStartX, page.selStartY)
                    onDragged: fwc.handleDragged(dx, dy)
                    onDropped: fwc.handleDropped()
                }
                Container {
                    visible: page.selected == "open" && page.selEndOn && !fwc.padOn
                    background: Color.create("#00a8df")
                    preferredWidth: 2
                    preferredHeight: page.cellH + fwc.hang(page.selEndY)
                    translationX: (page.selEndX + 1) * page.cellW - 1
                    translationY: page.selEndY * page.cellH
                    horizontalAlignment: HorizontalAlignment.Left
                    verticalAlignment: VerticalAlignment.Top
                    touchPropagationMode: TouchPropagationMode.None
                }
                SelHandle {
                    visible: page.selected == "open" && page.selEndOn && !fwc.padOn
                    translationX: (page.selEndX + 1) * page.cellW - 50
                    translationY: (page.selEndY + 1) * page.cellH + fwc.hang(page.selEndY) - 50
                    horizontalAlignment: HorizontalAlignment.Left
                    verticalAlignment: VerticalAlignment.Top
                    onGrabbed: fwc.handleGrabbed(page.selEndX, page.selEndY)
                    onDragged: fwc.handleDragged(dx, dy)
                    onDropped: fwc.handleDropped()
                }
                // The selection's bar: Copy, Paste, Select Line and Select
                // All, floating by the selected text where Go says
                // (internal/selbar) for as long as the selection is open,
                // but not while a finger is stretching it. It is not part of
                // the terminal: a touch on it is the bar's alone.
                Container {
                    visible: page.selected == "open" && !fwc.stretching && !fwc.padOn
                    layout: AbsoluteLayout {}
                    preferredWidth: 448
                    preferredHeight: 104
                    translationX: page.selBarX
                    translationY: page.selBarY
                    horizontalAlignment: HorizontalAlignment.Left
                    verticalAlignment: VerticalAlignment.Top
                    ImageView {
                        imageSource: "asset:///images/selbar.png"
                    }
                    Container {
                        layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
                        SelCell {
                            imageSource: "asset:///images/copy.png"
                            onPressed: page.send("copy")
                        }
                        SelCell {
                            imageSource: "asset:///images/paste.png"
                            onPressed: {
                                page.send("unselect")
                                page.pasteClipboard()
                            }
                        }
                        SelCell {
                            imageSource: "asset:///images/selectline.png"
                            onPressed: page.send("selectline")
                        }
                        SelCell {
                            imageSource: "asset:///images/selectall.png"
                            onPressed: page.send("selectall")
                        }
                    }
                }
            }
        }
    ]
}
