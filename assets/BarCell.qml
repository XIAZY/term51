import bb.cascades 1.4

// One place to press on the bottom bar: flat, no frame. It lights up in the
// primary colour when a finger comes down and fades back when it leaves (so
// that also a quick tap shows), and stays in the primary colour while armed
// (a sticky Ctrl or Alt). A press counts when the finger comes up inside
// the cell.
Container {
    id: cell
    property alias text: label.text
    property string icon: ""                  // a picture before the text
    property string badge: ""                 // a number drawn in the picture
    property variant plain: Color.Transparent // the background at rest
    property bool armed: false
    property bool down: false
    property real w: 0
    property real h: 0
    signal pressed()

    layout: DockLayout {}
    verticalAlignment: VerticalAlignment.Fill
    background: armed ? ui.palette.primary : plain

    Container { // the light of a press
        id: light
        background: ui.palette.primary
        opacity: 0
        horizontalAlignment: HorizontalAlignment.Fill
        verticalAlignment: VerticalAlignment.Fill
        animations: FadeTransition { id: fade; fromOpacity: 1; toOpacity: 0; duration: 180 }
    }
    Container {
        layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
        horizontalAlignment: HorizontalAlignment.Center
        verticalAlignment: VerticalAlignment.Center
        Container {
            layout: DockLayout {}
            visible: cell.icon != ""
            verticalAlignment: VerticalAlignment.Center
            ImageView {
                imageSource: cell.icon
            }
            Label {
                // In the front sheet of the tabs picture, which is left of
                // and below the picture's middle.
                text: cell.badge
                textStyle.fontSize: FontSize.XXSmall
                textStyle.fontWeight: FontWeight.Bold
                horizontalAlignment: HorizontalAlignment.Center
                verticalAlignment: VerticalAlignment.Center
                translationX: -7
                translationY: 8
            }
        }
        Label {
            id: label
            visible: text != ""
            textStyle.fontSize: FontSize.XSmall
            textStyle.fontWeight: FontWeight.W300
            textStyle.color: cell.armed ? ui.palette.textOnPrimary : ui.palette.textOnPlain
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
