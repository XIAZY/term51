import bb.cascades 1.4

// One place to press on the bottom bar: flat, no frame. It lights up while
// a finger is on it, and stays in the primary colour while armed (a sticky
// Ctrl or Alt). A press counts when the finger comes up inside the cell.
Container {
    id: cell
    property alias text: label.text
    property string icon: ""                  // a picture before the text
    property variant plain: Color.Transparent // the background at rest
    property bool armed: false
    property bool down: false
    property real w: 0
    property real h: 0
    signal pressed()

    layout: DockLayout {}
    verticalAlignment: VerticalAlignment.Fill
    background: (armed || down) ? ui.palette.primary : plain

    Container {
        layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
        horizontalAlignment: HorizontalAlignment.Center
        verticalAlignment: VerticalAlignment.Center
        ImageView {
            imageSource: cell.icon
            visible: cell.icon != ""
            verticalAlignment: VerticalAlignment.Center
        }
        Label {
            id: label
            visible: text != ""
            textStyle.color: (cell.armed || cell.down) ? ui.palette.textOnPrimary : ui.palette.textOnPlain
            verticalAlignment: VerticalAlignment.Center
        }
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
        } else if (event.isUp()) {
            var inside = down && event.localX >= 0 && event.localY >= 0 && event.localX < w && event.localY < h
            down = false
            if (inside)
                pressed()
        } else if (event.isCancel()) {
            down = false
        }
    }
}
