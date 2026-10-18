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

    // The PowerShell that can be downloaded (internal/pwsh in Go): where it
    // is, as Go tells with every change.
    property string pwshPhase: "absent"  // absent, downloading, unpacking, ready, failed
    property real pwshDone: 0            // bytes so far, and of how many
    property real pwshTotal: 0
    property string pwshVersion: ""      // the version that is here
    property string pwshAvailable: ""    // a newer one that could be downloaded
    property string pwshReason: ""       // why the last download did not work
    function pwsh(v) {
        pwshPhase = v.phase
        pwshDone = v.done
        pwshTotal = v.total
        pwshVersion = v.version
        pwshAvailable = v.available
        pwshReason = v.reason
    }
    function mb(bytes) {
        return Math.round(bytes / 104857.6) / 10
    }
    function pwshText() {
        if (pwshPhase == "downloading")
            return pwshTotal > 0 ? "Downloading: " + mb(pwshDone) + " of " + mb(pwshTotal) + " MB" : "Downloading"
        if (pwshPhase == "unpacking")
            return "Unpacking"
        if (pwshPhase == "ready")
            return "Version " + pwshVersion + " is on the phone"
                + (pwshAvailable != "" ? "; version " + pwshAvailable + " can be downloaded" : "")
                + (pwshReason != "" ? ". The last download did not work: " + pwshReason : "")
        if (pwshPhase == "failed")
            return "The download did not work: " + pwshReason
        return "Not on the phone. It is a download of 44 MB and takes 129 MB."
    }

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
        // A shell that is neither of the two (the path of a program, from
        // the settings file) leaves the list without a choice.
        var shells = { "system": 0, "busybox": 1, "powershell": 2 }
        if (v["shell"] in shells)
            shell.selectedIndex = shells[v["shell"]]
        else
            shell.resetSelectedIndex()
        term.selectedIndex = { "xterm": 0, "xterm-256color": 1, "vt100": 2, "ansi": 3, "qansi": 4 }[v["term"]]
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
                DropDown {
                    // settings.ShellSystem, ShellBusyBox and ShellPowerShell in Go.
                    // Any other program is named by its path in the
                    // settings file.
                    id: shell
                    title: "Shell"
                    Option { text: "System shell"; description: "The phone's own, /bin/sh"; value: "system" }
                    Option { text: "BusyBox shell"; description: "ash, from the BusyBox in the app"; value: "busybox" }
                    Option {
                        text: "PowerShell"
                        description: sheet.pwshPhase == "ready" ? "PowerShell 7; a few seconds to start" : "Download it below first"
                        value: "powershell"
                        enabled: sheet.pwshPhase == "ready"
                    }
                    onSelectedValueChanged: if (selectedIndex >= 0) sheet.set("shell", selectedValue)
                }
                DropDown {
                    // The values the app accepts (settings.Terms in Go). All
                    // but xterm-256color are known to the phone's own
                    // programs.
                    id: term
                    title: "Terminal type"
                    Option { text: "xterm"; value: "xterm" }
                    Option { text: "xterm-256color"; description: "For other machines, over ssh"; value: "xterm-256color" }
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

                // PowerShell is not in the app: it is fetched from its
                // releases when asked for here, and can be removed again.
                Header { title: "PowerShell" }
                Container {
                    leftPadding: ui.du(3)
                    rightPadding: ui.du(3)
                    topPadding: ui.du(2)
                    bottomPadding: ui.du(2)
                    Label {
                        multiline: true
                        text: sheet.pwshText()
                    }
                    Button {
                        text: sheet.pwshAvailable != "" ? "Download " + sheet.pwshAvailable : "Download"
                        visible: sheet.pwshPhase == "absent" || sheet.pwshPhase == "failed" || (sheet.pwshPhase == "ready" && sheet.pwshAvailable != "")
                        horizontalAlignment: HorizontalAlignment.Fill
                        onClicked: bridge.send("pwsh", JSON.stringify({ action: "download" }))
                    }
                    Button {
                        text: "Remove"
                        visible: sheet.pwshPhase == "ready"
                        horizontalAlignment: HorizontalAlignment.Fill
                        onClicked: bridge.send("pwsh", JSON.stringify({ action: "remove" }))
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
                        text: "Colours of your own and the shell's arguments: ~/.config/term51/settings.conf in the app's home."
                        textStyle.color: Color.Gray
                    }
                }
            }
        }
    }
}
