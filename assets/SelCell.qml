import bb.cascades 1.4

// One button of the selection's bar: an icon in its cell of the bar's
// picture. The icon turns to the primary colour while a finger is on it. A
// press counts when the finger comes up inside the cell.
Container {
    id: cell
    property alias imageSource: icon.imageSource
    property bool down: false
    signal pressed()

    layout: DockLayout {}
    preferredWidth: 112
    preferredHeight: 104

    ImageView {
        id: icon
        filterColor: cell.down ? ui.palette.primary : Color.White
        preferredWidth: 81
        preferredHeight: 81
        horizontalAlignment: HorizontalAlignment.Center
        verticalAlignment: VerticalAlignment.Center
    }
    onTouch: {
        if (event.isDown()) {
            down = true
        } else if (event.isUp() || event.isCancel()) {
            var inside = down && event.isUp() && event.localX >= 0 && event.localY >= 0 && event.localX < 112 && event.localY < 104
            down = false
            if (inside)
                pressed()
        }
    }
}
