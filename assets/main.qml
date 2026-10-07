import bb.cascades 1.4
import bb.system 1.2
import bb.multimedia 1.4

// The terminal page, shaped after BlackBerry 10's browser: the terminal from
// the top edge down to one bar at the bottom. The bar has the tabs button
// (with the number of tabs) on the left, the three dots on the right, and
// between them, where the browser has its address field, the keys the
// phone's keyboard lacks. The tabs button opens a drawer from the left; the
// dots open the system's menu from the right. The terminal itself is not
// drawn by Cascades: Go draws it with OpenGL into a window of its own, and
// the ForeignWindowControl below is the place in the page where that window
// shows.
Page {
    id: page

    property bool ctrl: false
    property bool alt: false
    property string termTitle: "term51"
    property int tabCount: 1
    property int tabIndex: 0

    property bool created: false     // the terminal's window has been asked for
    property string termBackground: "#101418" // the theme's, around the terminal
    property bool bell: true         // a program's bell makes a sound
    property bool covered: false     // the settings sheet is over the terminal
    property bool drawer: false      // the tabs drawer is open
    property real cellW: 0           // a cell in pixels, from Go
    property real cellH: 0

    function send(name, args) { bridge.send(name, JSON.stringify(args || {})) }

    // Types the clipboard's text into the shell.
    function pasteClipboard() {
        var text = bridge.paste()
        if (text.length > 0)
            send("paste", { text: text })
    }

    // A development aid (internal/app/remote.go): what a developer asks the
    // page to do from a file, to look at it without the phone in hand.
    function remote(line) {
        if (line == "drawer open") openDrawer()
        else if (line == "drawer close") closeDrawer()
        else if (line == "menu") page.openActionMenu()
        else if (line == "settings") settingsSheet.open()
        else if (line == "back") settingsSheet.close()
        else if (line.indexOf("shot ") == 0) bridge.shot(line.substring(5))
    }

    // The tabs drawer slides in from the left and out again.
    function openDrawer() {
        page.drawer = true
        dimIn.play()
        slideIn.play()
    }
    function closeDrawer() {
        dimOut.play()
        slideOut.play() // page.drawer goes false when it has ended
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
            if (s.title !== undefined) page.termTitle = s.title
            if (s.cw !== undefined) page.cellW = s.cw
            if (s.bell !== undefined) page.bell = s.bell
            if (s.settings !== undefined) page.termBackground = s.settings["theme.background"]
            if (s.ch !== undefined) page.cellH = s.ch
            if (s.tabs !== undefined) {
                page.tabCount = s.tabs.length
                // The list of tabs, for the drawer.
                var rows = []
                for (var i = 0; i < s.tabs.length; i++)
                    rows.push({ title: s.tabs[i], shown: i == s.current,
                                status: s.rang[i] ? "bell" : "" })
                tabsModel.clear()
                tabsModel.append(rows)
            }
            if (s.current !== undefined) page.tabIndex = s.current
        })
        bridge.event.connect(function(json) {
            var e = JSON.parse(json)
            if (e.clipboard !== undefined) {
                bridge.copy(e.clipboard)
                return
            }
            if (e.bell !== undefined && page.bell)
                bellSound.play()
            if (e.remote !== undefined)
                page.remote(e.remote)
            if (e.notice !== undefined) {
                toast.body = e.notice
                toast.show()
            }
        })
    }

    // No title bar and no action bar. The actions below are the menu the
    // three dots open from the right (the system's own, as in the browser).
    actionBarVisibility: ChromeVisibility.Hidden
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
    attachedObjects: [
        SystemToast { id: toast },
        SystemSound { id: bellSound; sound: SystemSound.GeneralNotification },
        ArrayDataModel { id: tabsModel },
        SettingsSheet {
            id: settingsSheet
            onOpened: page.covered = true
            onClosed: page.covered = false
        }
    ]

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

        Container {
            layout: StackLayout {}
            horizontalAlignment: HorizontalAlignment.Fill
            verticalAlignment: VerticalAlignment.Fill

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
                layoutProperties: StackLayoutProperties { spaceQuota: 1 }
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

            // The bar, the last thing on the page (right above the phone's
            // on-screen keyboard rows when those are up), after the
            // browser's: one dark strip with the tabs button at the left end,
            // the three dots at the right end, and between them, as flat
            // cells, the keys the phone's keyboard lacks. Nothing on it takes
            // the focus. Ctrl and Alt are sticky: they apply to the next key
            // and stay in the primary colour until then. All sizes are in
            // pixels, from the browser's bar on this phone.
            Container {
                horizontalAlignment: HorizontalAlignment.Fill
                Container { // the line on the strip's top edge
                    background: Color.create("#141414")
                    preferredHeight: 2
                    horizontalAlignment: HorizontalAlignment.Fill
                }
                Container {
                    layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
                    background: Color.create("#1F1F1F")
                    preferredHeight: 118
                    horizontalAlignment: HorizontalAlignment.Fill

                    BarCell { badge: "" + page.tabCount; icon: "asset:///images/tabs.png"
                              preferredWidth: 144
                              onPressed: page.openDrawer() }
                    Container { // a divider, short of the strip's edges
                        topPadding: 18
                        bottomPadding: 18
                        verticalAlignment: VerticalAlignment.Fill
                        Container {
                            background: Color.create("#3C3C3C")
                            preferredWidth: 2
                            verticalAlignment: VerticalAlignment.Fill
                        }
                    }
                    Container {
                        layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
                        layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                        leftPadding: 8
                        rightPadding: 8
                        topPadding: 16
                        bottomPadding: 16
                        verticalAlignment: VerticalAlignment.Fill
                        BarCell { text: "Esc"; plain: Color.create("#2E2E2E")
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  onPressed: page.send("bar", { name: "esc" }) }
                        BarCell { text: "Ctrl"; plain: Color.create("#2E2E2E"); armed: page.ctrl
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  leftMargin: 8; onPressed: page.send("bar", { name: "ctrl" }) }
                        BarCell { text: "Alt"; plain: Color.create("#2E2E2E"); armed: page.alt
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  leftMargin: 8; onPressed: page.send("bar", { name: "alt" }) }
                        BarCell { text: "Tab"; plain: Color.create("#2E2E2E")
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  leftMargin: 8; onPressed: page.send("bar", { name: "tab" }) }
                        BarCell { text: "←"; plain: Color.create("#2E2E2E")
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  leftMargin: 8; onPressed: page.send("bar", { name: "left" }) }
                        BarCell { text: "↓"; plain: Color.create("#2E2E2E")
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  leftMargin: 8; onPressed: page.send("bar", { name: "down" }) }
                        BarCell { text: "↑"; plain: Color.create("#2E2E2E")
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  leftMargin: 8; onPressed: page.send("bar", { name: "up" }) }
                        BarCell { text: "→"; plain: Color.create("#2E2E2E")
                                  layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                  leftMargin: 8; onPressed: page.send("bar", { name: "right" }) }
                    }
                    Container { // a divider, short of the strip's edges
                        topPadding: 18
                        bottomPadding: 18
                        verticalAlignment: VerticalAlignment.Fill
                        Container {
                            background: Color.create("#3C3C3C")
                            preferredWidth: 2
                            verticalAlignment: VerticalAlignment.Fill
                        }
                    }
                    BarCell { icon: "asset:///images/dots.png"
                              preferredWidth: 144
                              onPressed: page.openActionMenu() }
                }
            }
        }

        // The tabs drawer, after the browser's: three quarters of the page
        // wide, sliding in from the left; New Tab, then the tabs, in rows as
        // high as the browser's. The tab on screen is a row in the primary
        // colour with a cross that closes it; one whose bell rang while
        // another was shown says "bell". The rest of the page is dimmed, and
        // a tap there closes the drawer.
        Container {
            visible: page.drawer
            layout: DockLayout {}
            horizontalAlignment: HorizontalAlignment.Fill
            verticalAlignment: VerticalAlignment.Fill

            Container {
                background: Color.Black
                opacity: 0
                horizontalAlignment: HorizontalAlignment.Fill
                verticalAlignment: VerticalAlignment.Fill
                gestureHandlers: TapHandler {
                    onTapped: page.closeDrawer()
                }
                animations: [
                    FadeTransition { id: dimIn; fromOpacity: 0; toOpacity: 0.6; duration: 200 },
                    FadeTransition { id: dimOut; fromOpacity: 0.6; toOpacity: 0; duration: 200 }
                ]
            }
            Container {
                layout: DockLayout {}
                background: Color.create("#1E1E1E")
                preferredWidth: 1080
                translationX: -1080
                horizontalAlignment: HorizontalAlignment.Left
                verticalAlignment: VerticalAlignment.Fill
                animations: [
                    TranslateTransition { id: slideIn; fromX: -1080; toX: 0; duration: 200 },
                    TranslateTransition {
                        id: slideOut
                        fromX: 0
                        toX: -1080
                        duration: 200
                        onEnded: page.drawer = false
                    }
                ]

                // The rows stand at the bottom, within the thumb's reach.
                Container {
                    horizontalAlignment: HorizontalAlignment.Fill
                    verticalAlignment: VerticalAlignment.Bottom
                    bottomPadding: 122 // the bar's height, which the drawer covers

                    Container { background: Color.create("#383838"); preferredHeight: 2; horizontalAlignment: HorizontalAlignment.Fill }
                    Container {
                        layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
                        preferredHeight: 130
                        horizontalAlignment: HorizontalAlignment.Fill
                        Container {
                            layout: DockLayout {}
                            preferredWidth: 130
                            verticalAlignment: VerticalAlignment.Fill
                            ImageView {
                                imageSource: "asset:///images/newtab.png"
                                horizontalAlignment: HorizontalAlignment.Center
                                verticalAlignment: VerticalAlignment.Center
                            }
                        }
                        Container {
                            leftPadding: 16
                            verticalAlignment: VerticalAlignment.Center
                            layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                            Label {
                                text: "New Tab"
                                textStyle.fontSize: FontSize.XLarge
                            }
                        }
                        gestureHandlers: TapHandler {
                            onTapped: {
                                page.send("newtab")
                                page.closeDrawer()
                            }
                        }
                    }
                    ListView {
                        dataModel: tabsModel
                        // As high as its rows, up to six of them; more scroll.
                        // A last line closes the block.
                        preferredHeight: Math.min(page.tabCount, 6) * 132
                        function closeTab(index) { page.send("closetab", { index: index }) }
                        listItemComponents: ListItemComponent {
                            Container {
                                id: row
                                preferredHeight: 132
                                Container { background: Color.create("#383838"); preferredHeight: 2; horizontalAlignment: HorizontalAlignment.Fill }
                                Container {
                                    layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
                                    preferredHeight: 130
                                    horizontalAlignment: HorizontalAlignment.Fill
                                    Container {
                                        layout: DockLayout {}
                                        preferredWidth: 130
                                        verticalAlignment: VerticalAlignment.Fill
                                        ImageView {
                                            imageSource: "asset:///images/tabs.png"
                                            opacity: ListItemData.shown ? 0.3 : 1
                                            horizontalAlignment: HorizontalAlignment.Center
                                            verticalAlignment: VerticalAlignment.Center
                                        }
                                    }
                                    Container {
                                        layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
                                        background: ListItemData.shown ? ui.palette.primary : Color.Transparent
                                        leftPadding: 16
                                        rightPadding: 16
                                        verticalAlignment: VerticalAlignment.Fill
                                        layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                        Label {
                                            text: ListItemData.title
                                            textStyle.fontSize: FontSize.XLarge
                                            verticalAlignment: VerticalAlignment.Center
                                            layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                                        }
                                        Label {
                                            text: ListItemData.status
                                            textStyle.color: Color.create("#e5c07b")
                                            verticalAlignment: VerticalAlignment.Center
                                        }
                                        ImageButton {
                                            visible: ListItemData.shown
                                            defaultImageSource: "asset:///images/closetab.png"
                                            verticalAlignment: VerticalAlignment.Center
                                            onClicked: row.ListItem.view.closeTab(row.ListItem.indexPath[0])
                                        }
                                    }
                                }
                            }
                        }
                        onTriggered: {
                            page.send("tab", { index: indexPath[0] })
                            page.closeDrawer()
                        }
                    }
                    Container { background: Color.create("#383838"); preferredHeight: 2; horizontalAlignment: HorizontalAlignment.Fill }
                }
            }
        }
    }
}
