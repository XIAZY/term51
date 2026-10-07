import bb.cascades 1.4

// The terminal as a column of text labels, one per row, in a monospace
// font. Go sends the rows that changed as rich text; this page only shows
// them. The key bar and the keyboard work as on the pixel page.
//
// The size of a cell is measured, not assumed: a hidden label holds 80
// characters, and its laid-out size gives the width of one and the height
// of a row.
Page {
    id: page

    property bool ctrl: false
    property bool alt: false
    property string termTitle: "term51"

    property int fontPt: 7           // the labels' font size, in points
    property real cellW: 0
    property real cellH: 0
    property real narrowW: -1        // the width of 80 "i", to check the font
    property real areaW: 0
    property real areaH: 0
    property int cols: 0
    property int rows: 0
    property variant labels: []

    function send(name, args) { bridge.send(name, JSON.stringify(args || {})) }

    function keyArgs(event, down) {
        return { down: down, key: event.key, keycap: event.keycap, unicode: event.unicode,
                 shift: event.shiftPressed, ctrl: event.ctrlPressed, alt: event.altPressed }
    }

    // Tells Go how many cells fit, once both the cell and the area are known.
    function fit() {
        if (cellW <= 0 || cellH <= 0 || areaW <= 0 || areaH <= 0)
            return
        var c = Math.max(1, Math.floor(areaW / cellW))
        var r = Math.max(1, Math.floor(areaH / cellH))
        if (c == cols && r == rows)
            return
        cols = c
        rows = r
        // In a monospace font 80 "i" are as wide as 80 "M".
        var mono = narrowW < 0 || Math.abs(narrowW - cellW * 80) < 1
        send("grid", { cols: c, rows: r, cw: Math.round(cellW), ch: Math.round(cellH), mono: mono })
    }

    // Makes the column hold exactly n labels.
    function setCount(n) {
        var ls = labels
        while (ls.length < n) {
            var l = rowLabel.createObject()
            grid.add(l)
            ls.push(l)
        }
        while (ls.length > n) {
            var last = ls.pop()
            grid.remove(last)
            last.destroy()
        }
        labels = ls
    }

    onCreationCompleted: {
        bridge.state.connect(function(json) {
            var s = JSON.parse(json)
            if (s.ctrl !== undefined) page.ctrl = s.ctrl
            if (s.alt !== undefined) page.alt = s.alt
            if (s.title !== undefined) page.termTitle = s.title
        })
        bridge.event.connect(function(json) {
            var e = JSON.parse(json)
            if (e.clipboard !== undefined) {
                bridge.copy(e.clipboard)
                return
            }
            if (e.rows === undefined)
                return
            if (e.count != page.labels.length)
                page.setCount(e.count)
            var ls = page.labels // read once: each access converts the list
            for (var i = 0; i < e.rows.length; i++) {
                var row = e.rows[i]
                if (row.y < ls.length)
                    ls[row.y].text = "<html>" + row.html + "</html>"
            }
        })
    }

    titleBar: TitleBar { title: page.termTitle }

    // Copy takes the selection, or the whole screen when nothing is
    // selected. Go answers with the text; the page owns the clipboard.
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

    keyListeners: KeyListener {
        onKeyPressed: page.send("key", page.keyArgs(event, true))
        onKeyReleased: page.send("key", page.keyArgs(event, false))
    }

    attachedObjects: [
        ComponentDefinition {
            id: rowLabel
            Label {
                multiline: false
                textFormat: TextFormat.Html
                textStyle.fontFamily: "Monospace"
                textStyle.fontSize: FontSize.PointValue
                textStyle.fontSizeValue: page.fontPt
                topMargin: 0
                bottomMargin: 0
                horizontalAlignment: HorizontalAlignment.Left
                // Far wider than the screen: a single-line label that is one
                // pixel too narrow would end its text with an ellipsis. The
                // grid container clips.
                minWidth: 4000
                preferredWidth: 4000
            }
        }
    ]

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

        // The measuring label: 80 characters, never shown. (It and the one
        // after it are invisible but still take two rows of space above
        // the grid.)
        Label {
            id: probe
            text: "MMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM"
            multiline: false
            textStyle.fontFamily: "Monospace"
            textStyle.fontSize: FontSize.PointValue
            textStyle.fontSizeValue: page.fontPt
            topMargin: 0
            bottomMargin: 0
            opacity: 0
            horizontalAlignment: HorizontalAlignment.Left
            attachedObjects: LayoutUpdateHandler {
                onLayoutFrameChanged: {
                    page.cellW = layoutFrame.width / 80
                    page.cellH = layoutFrame.height
                    page.fit()
                }
            }
        }

        // A second one, of 80 narrow letters: in a monospace font it is as
        // wide as the first. If the family name is not known to the phone,
        // Cascades falls back to a proportional font without saying so.
        Label {
            text: "iiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiii"
            multiline: false
            textStyle.fontFamily: "Monospace"
            textStyle.fontSize: FontSize.PointValue
            textStyle.fontSizeValue: page.fontPt
            topMargin: 0
            bottomMargin: 0
            opacity: 0
            horizontalAlignment: HorizontalAlignment.Left
            attachedObjects: LayoutUpdateHandler {
                onLayoutFrameChanged: {
                    page.narrowW = layoutFrame.width
                    page.fit()
                }
            }
        }

        // The terminal: one label per row. A vertical drag scrolls the
        // scrollback.
        Container {
            id: grid
            layout: StackLayout {}
            clipContentToBounds: true
            horizontalAlignment: HorizontalAlignment.Fill
            layoutProperties: StackLayoutProperties { spaceQuota: 1 }
            property real touchY: 0
            attachedObjects: LayoutUpdateHandler {
                onLayoutFrameChanged: {
                    page.areaW = layoutFrame.width
                    page.areaH = layoutFrame.height
                    page.fit()
                }
            }
            // A long press selects the word under the finger.
            gestureHandlers: LongPressHandler {
                onLongPressed: {
                    if (page.cellW > 0 && page.cellH > 0)
                        page.send("selectword", { x: Math.floor(event.x / page.cellW), y: Math.floor(event.y / page.cellH) })
                }
            }
            onTouch: {
                if (event.isDown()) {
                    grid.touchY = event.localY
                } else if (event.isMove() && page.cellH > 0) {
                    var d = Math.floor((event.localY - grid.touchY) / page.cellH)
                    if (d != 0) {
                        page.send("scroll", { rows: -d })
                        grid.touchY += d * page.cellH
                    }
                }
            }
        }
    }
}
