import bb.cascades 1.4
import bb.system 1.2
import bb.multimedia 1.4

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
    property string termBackground: "#101418" // the theme's, around the terminal
    property bool bell: true         // a program's bell makes a sound
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
            if (s.title !== undefined) page.termTitle = s.title
            if (s.cw !== undefined) page.cellW = s.cw
            if (s.bell !== undefined) page.bell = s.bell
            if (s.settings !== undefined) page.termBackground = s.settings["theme.background"]
            if (s.ch !== undefined) page.cellH = s.ch
            if (s.tabs !== undefined) {
                page.tabCount = s.tabs.length
                // The list of tabs, for the tabs sheet.
                var rows = []
                for (var i = 0; i < s.tabs.length; i++)
                    rows.push({ title: (i + 1) + "   " + s.tabs[i],
                                status: s.rang[i] ? "bell" : (i == s.current ? "shown" : "") })
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

    // The app's menu. BlackBerry 10 has an icon of its own for settings;
    // the other four are ours (tools/icons). (The system's delete item,
    // which has an icon too, is not accepted in an app menu.)
    Menu.definition: MenuDefinition {
        actions: [
            ActionItem {
                title: "New tab"
                imageSource: "asset:///images/newtab.png"
                onTriggered: page.send("newtab")
            },
            ActionItem {
                title: "Close tab"
                imageSource: "asset:///images/closetab.png"
                onTriggered: page.send("closetab")
            },
            ActionItem {
                title: "Tabs"
                imageSource: "asset:///images/tabs.png"
                onTriggered: tabsSheet.open()
            },
            ActionItem {
                title: "Paste"
                imageSource: "asset:///images/paste.png"
                onTriggered: page.pasteClipboard()
            }
        ]
        settingsAction: SettingsActionItem {
            onTriggered: settingsSheet.open()
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
    attachedObjects: [
        SystemToast { id: toast },
        SystemSound { id: bellSound; sound: SystemSound.GeneralNotification },
        ArrayDataModel { id: tabsModel },
        SettingsSheet { id: settingsSheet },

        // The tabs: a tap shows one, a press and hold offers to close it.
        // The one on screen says "shown"; one whose bell rang while another
        // was shown says "bell".
        Sheet {
            id: tabsSheet
            Page {
                titleBar: TitleBar {
                    title: "Tabs"
                    dismissAction: ActionItem {
                        title: "Close"
                        onTriggered: tabsSheet.close()
                    }
                    acceptAction: ActionItem {
                        title: "New"
                        onTriggered: {
                            page.send("newtab")
                            tabsSheet.close()
                        }
                    }
                }
                Container {
                    ListView {
                        dataModel: tabsModel
                        layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                        function closeTab(index) { page.send("closetab", { index: index }) }
                        listItemComponents: ListItemComponent {
                            StandardListItem {
                                id: row
                                title: ListItemData.title
                                status: ListItemData.status
                                contextActions: ActionSet {
                                    title: ListItemData.title
                                    DeleteActionItem {
                                        title: "Close"
                                        onTriggered: row.ListItem.view.closeTab(row.ListItem.indexPath[0])
                                    }
                                }
                            }
                        }
                        onTriggered: {
                            page.send("tab", { index: indexPath[0] })
                            tabsSheet.close()
                        }
                    }
                    Label {
                        text: "Tap a tab to show it. Press and hold for Close."
                        textStyle.base: SystemDefaults.TextStyles.SubtitleText
                        textStyle.color: Color.Gray
                        horizontalAlignment: HorizontalAlignment.Center
                    }
                }
            }
        }
    ]

    Container {
        layout: StackLayout {}
        background: Color.create(page.termBackground)

        // The terminal. Position, Size and Visible: Cascades keeps Go's
        // window over this control's area. A vertical drag scrolls the
        // scrollback. A press and hold selects the word under the finger
        // and opens the standard context menu (on text that is already
        // selected it keeps the selection). A tap brings up the on-screen
        // keyboard rows (the phone's own swipe down puts them away).
        ForeignWindowControl {
            id: fwc
            windowId: "term51"
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
                    title: "Select all"
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
                    title: "Cancel"
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
