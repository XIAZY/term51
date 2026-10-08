import bb.cascades 1.4

// The dot of a selection handle and the 100 pixels square around it that a
// finger can take it by. It tells when it is grabbed, how far the finger has
// gone from where it took it (in the window's pixels: the dot itself moves
// with the selection's end while it is dragged), and when it is let go.
Container {
    id: grab
    property bool held: false
    property real downX: 0
    property real downY: 0
    signal grabbed()
    signal dragged(real dx, real dy)
    signal dropped()

    layout: DockLayout {}
    preferredWidth: 100
    preferredHeight: 100

    ImageView {
        imageSource: "asset:///images/handle.png"
        preferredWidth: 44
        preferredHeight: 44
        horizontalAlignment: HorizontalAlignment.Center
        verticalAlignment: VerticalAlignment.Center
    }
    onTouch: {
        if (event.isDown()) {
            held = true
            downX = event.windowX
            downY = event.windowY
            grabbed()
        } else if (event.isMove() && held) {
            dragged(event.windowX - downX, event.windowY - downY)
        } else if ((event.isUp() || event.isCancel()) && held) {
            held = false
            dropped()
        }
    }
}
