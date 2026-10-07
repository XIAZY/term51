import bb.cascades 1.4

// One text setting of the settings sheet: what it is, and a field. The
// text counts as changed when Enter is pressed or the field is left.
Container {
    property alias title: label.text
    property alias text: field.text
    signal edited(string text)

    leftPadding: ui.du(3)
    rightPadding: ui.du(3)
    topPadding: ui.du(1)
    bottomPadding: ui.du(1)

    Label {
        id: label
        textStyle.base: SystemDefaults.TextStyles.SubtitleText
        textStyle.color: Color.Gray
    }
    TextField {
        id: field
        hintText: ""
        input.flags: TextInputFlag.AutoCapitalizationOff | TextInputFlag.AutoCorrectionOff | TextInputFlag.PredictionOff | TextInputFlag.SpellCheckOff | TextInputFlag.WordSubstitutionOff | TextInputFlag.AutoPeriodOff
        input.submitKey: SubmitKey.Done
        input.onSubmitted: edited(field.text)
        onFocusedChanged: {
            if (!focused)
                edited(field.text)
        }
    }
}
