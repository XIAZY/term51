import bb.cascades 1.4

// The settings: standard controls for what the settings file holds. Go
// tells the page every setting's value (as the file would hold it, by the
// file's names); a change here goes to Go, which checks it, writes the file
// and answers with the values in force.
Sheet {
    id: sheet

    property variant values        // name -> value, as last told by Go
    property bool filling: false   // the controls are being set from values
    property int cols: 0           // the terminal's columns, for the text size

    function set(name, value) {
        var text = "" + value
        if (filling || values === undefined || values[name] === text)
            return
        bridge.send("set", JSON.stringify({ name: name, value: text }))
    }

    function fill(v) {
        values = v
        filling = true
        size.value = parseFloat(v["fontPt"])
        theme.selectedIndex = { "dark": 0, "black": 1, "light": 2, "solarized": 3, "custom": 4 }[v["theme"]]
        cursor.selectedIndex = { "block": 0, "bar": 1, "underline": 2 }[v["cursor"]]
        bell.checked = v["bell"] == "true"
        shell.text = v["shell"]
        term.selectedIndex = { "xterm-256color": 0, "xterm": 1, "vt220": 2, "vt100": 3, "ansi": 4, "qansi": 5 }[v["term"]]
        scrollback.text = "" + (parseInt(v["scrollbackBytes"]) / 1048576)
        filling = false
    }

    onCreationCompleted: {
        bridge.state.connect(function(json) {
            var s = JSON.parse(json)
            if (s.cols !== undefined) sheet.cols = s.cols
            if (s.settings !== undefined) sheet.fill(s.settings)
        })
    }

    Page {
        titleBar: TitleBar {
            title: "Settings"
            dismissAction: ActionItem {
                title: "Back"
                onTriggered: sheet.close()
            }
        }
        ScrollView {
            Container {
                Header { title: "Text"; subtitle: "Applies at once" }
                Container {
                    leftPadding: ui.du(3)
                    rightPadding: ui.du(3)
                    topPadding: ui.du(2)
                    bottomPadding: ui.du(2)
                    Container {
                        layout: DockLayout {}
                        horizontalAlignment: HorizontalAlignment.Fill
                        Label {
                            text: "Size"
                            horizontalAlignment: HorizontalAlignment.Left
                        }
                        Label {
                            // Half points, as the slider is moved; the
                            // columns are those of the size in force.
                            text: (Math.round(size.immediateValue * 2) / 2) + " pt, " + sheet.cols + " columns"
                            textStyle.color: Color.Gray
                            horizontalAlignment: HorizontalAlignment.Right
                        }
                    }
                    Slider {
                        id: size
                        fromValue: 3
                        toValue: 40
                        horizontalAlignment: HorizontalAlignment.Fill
                        onValueChanged: sheet.set("fontPt", Math.round(value * 2) / 2)
                    }
                    Label {
                        text: "$ echo The quick brown fox 0123456789"
                        textStyle.fontFamily: "DejaVu Sans Mono"
                        textStyle.fontSize: FontSize.PointValue
                        textStyle.fontSizeValue: Math.round(size.immediateValue * 2) / 2
                    }
                }

                Header { title: "Colours and cursor"; subtitle: "Applies at once" }
                DropDown {
                    id: theme
                    title: "Theme"
                    Option { text: "Dark"; value: "dark" }
                    Option { text: "Black"; value: "black" }
                    Option { text: "Light"; value: "light" }
                    Option { text: "Solarized dark"; value: "solarized" }
                    // What the drop-down shows for colours someone wrote
                    // into the settings file; choosing it changes nothing.
                    Option { text: "Custom (from the settings file)"; value: "custom" }
                    onSelectedValueChanged: {
                        if (selectedValue != "custom")
                            sheet.set("theme", selectedValue)
                    }
                }
                Container {
                    leftPadding: ui.du(3)
                    rightPadding: ui.du(3)
                    topPadding: ui.du(2)
                    bottomPadding: ui.du(2)
                    Label { text: "Cursor" }
                    SegmentedControl {
                        id: cursor
                        Option { text: "Block"; value: "block" }
                        Option { text: "Bar"; value: "bar" }
                        Option { text: "Underline"; value: "underline" }
                        onSelectedValueChanged: sheet.set("cursor", selectedValue)
                    }
                }

                Header { title: "Sound"; subtitle: "Applies at once" }
                SettingToggle {
                    id: bell
                    text: "Bell: a sound when a program rings it"
                    onFlipped: sheet.set("bell", on)
                }

                Header { title: "Shell"; subtitle: "For tabs opened from now on" }
                SettingText {
                    id: shell
                    title: "Program"
                    onEdited: sheet.set("shell", text)
                }
                DropDown {
                    // The values the app accepts (settings.Terms in Go). The
                    // first two the phone's own programs do not all know;
                    // they are for programs on other machines, over ssh.
                    id: term
                    title: "Terminal type"
                    Option { text: "xterm-256color"; description: "For other machines, over ssh"; value: "xterm-256color" }
                    Option { text: "xterm"; value: "xterm" }
                    Option { text: "vt220"; description: "For other machines, over ssh"; value: "vt220" }
                    Option { text: "vt100"; value: "vt100" }
                    Option { text: "ansi"; value: "ansi" }
                    Option { text: "qansi"; description: "QNX's own"; value: "qansi" }
                    onSelectedValueChanged: sheet.set("term", selectedValue)
                }
                SettingText {
                    id: scrollback
                    title: "Scrollback, in megabytes"
                    onEdited: {
                        var mb = parseFloat(text)
                        sheet.set("scrollbackBytes", isNaN(mb) ? text : Math.round(mb * 1048576))
                    }
                }

                Header { title: "In the settings file only" }
                Container {
                    leftPadding: ui.du(3)
                    rightPadding: ui.du(3)
                    topPadding: ui.du(2)
                    bottomPadding: ui.du(4)
                    Label {
                        multiline: true
                        text: "Colours of your own and the shell's arguments: data/settings.conf in the app's folder."
                        textStyle.color: Color.Gray
                    }
                }
            }
        }
    }
}
