import bb.cascades 1.4

// One on-or-off setting of the settings sheet: its text and a toggle.
Container {
    property alias text: label.text
    property alias checked: toggle.checked
    signal flipped(bool on)

    layout: StackLayout { orientation: LayoutOrientation.LeftToRight }
    leftPadding: ui.du(3)
    rightPadding: ui.du(3)
    topPadding: ui.du(1)
    bottomPadding: ui.du(1)

    Label {
        id: label
        multiline: true
        verticalAlignment: VerticalAlignment.Center
        layoutProperties: StackLayoutProperties { spaceQuota: 1 }
    }
    ToggleButton {
        id: toggle
        verticalAlignment: VerticalAlignment.Center
        onCheckedChanged: flipped(checked)
    }
}
