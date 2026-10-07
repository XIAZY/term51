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
        send("grid", { cols: c, rows: r, cw: Math.round(cellW), ch: Math.round(cellH) })
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
            if (e.rows === undefined)
                return
            if (e.count != page.labels.length)
                page.setCount(e.count)
            for (var i = 0; i < e.rows.length; i++) {
                var row = e.rows[i]
                if (row.y < page.labels.length)
                    page.labels[row.y].text = "<html>" + row.html + "</html>"
            }
        })
    }

    titleBar: TitleBar { title: page.termTitle }

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
            }
        }
    ]

    Container {
        layout: StackLayout {}
        background: Color.create("#101418")

        // The key bar. Ctrl and Alt are sticky: they apply to the next key
        // and show as armed until then.
        Container {
            layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
            horizontalAlignment: HorizontalAlignment.Fill
            background: Color.create("#202428")

            Button { text: "Esc"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                     onClicked: page.send("bar", { name: "esc" }) }
            Button { text: page.ctrl ? "[Ctrl]" : "Ctrl"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                     onClicked: page.send("bar", { name: "ctrl" }) }
            Button { text: page.alt ? "[Alt]" : "Alt"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                     onClicked: page.send("bar", { name: "alt" }) }
            Button { text: "Tab"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                     onClicked: page.send("bar", { name: "tab" }) }
            Button { text: "←"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                     onClicked: page.send("bar", { name: "left" }) }
            Button { text: "↓"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                     onClicked: page.send("bar", { name: "down" }) }
            Button { text: "↑"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                     onClicked: page.send("bar", { name: "up" }) }
            Button { text: "→"; layoutProperties: StackLayoutProperties { spaceQuota: 1 }
                     onClicked: page.send("bar", { name: "right" }) }
        }

        // The measuring label: 80 characters, never shown.
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

        // The terminal: one label per row. A vertical drag scrolls the
        // scrollback.
        Container {
            id: grid
            layout: StackLayout {}
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
