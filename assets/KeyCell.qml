import bb.cascades 1.4

// One key of the strip above the system's bar: a plain label in its cell,
// no frame and no background, so that the strip reads as part of the bar.
// The cell lights up in the primary colour when a finger comes down and
// fades back when it leaves (so that also a quick tap shows), and stays lit
// while armed (a sticky Ctrl or Alt). A press counts when the finger comes
// up inside the cell.
Container {
    id: cell
    property alias text: label.text
    property bool armed: false
    property bool symbol: false // a single character: shown larger, to be read
    property bool down: false
    property real w: 0
    property real h: 0
    signal pressed()

    layout: DockLayout {}
    verticalAlignment: VerticalAlignment.Fill
    // It takes the focus by touch, which the input field refuses to give
    // up to such a control: a touch here leaves the on-screen rows as they
    // are. Its children let touches through, so that the cell itself is
    // what a finger touches.
    focusPolicy: FocusPolicy.Touch
    background: armed ? ui.palette.primary : Color.Transparent

    Container { // the light of a press
        id: light
        touchPropagationMode: TouchPropagationMode.PassThrough
        background: ui.palette.primary
        opacity: 0
        horizontalAlignment: HorizontalAlignment.Fill
        verticalAlignment: VerticalAlignment.Fill
        animations: FadeTransition { id: fade; fromOpacity: 1; toOpacity: 0; duration: 180 }
    }
    Label {
        id: label
        touchPropagationMode: TouchPropagationMode.PassThrough
        textStyle.fontSize: symbol ? FontSize.Large : FontSize.Small
        textStyle.color: cell.armed ? ui.palette.textOnPrimary : ui.palette.textOnPlain
        horizontalAlignment: HorizontalAlignment.Center
        verticalAlignment: VerticalAlignment.Center
    }
    attachedObjects: LayoutUpdateHandler {
        onLayoutFrameChanged: {
            cell.w = layoutFrame.width
            cell.h = layoutFrame.height
        }
    }
    onTouch: {
        if (event.isDown()) {
            down = true
            fade.stop()
            light.opacity = 1
        } else if (event.isUp() || event.isCancel()) {
            var inside = down && event.isUp() && event.localX >= 0 && event.localY >= 0 && event.localX < w && event.localY < h
            down = false
            fade.play()
            if (inside)
                pressed()
        }
    }
}
