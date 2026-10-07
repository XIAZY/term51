#pragma once

#include <QObject>
#include <QString>

// The one object QML sees. Go posts JSON to it from any thread (queued);
// QML parses it. QML sends commands back as a name and JSON arguments.
class Bridge : public QObject {
    Q_OBJECT

public:
    Q_INVOKABLE void send(const QString &name, const QString &argsJson);
    // The system clipboard, as plain text.
    Q_INVOKABLE void copy(const QString &text);
    Q_INVOKABLE QString paste();
    // A development aid (see internal/app/remote.go): a picture of the
    // display as data/<name>.png. It returns the file's path, or nothing
    // when the picture could not be taken (the reason is on stderr).
    Q_INVOKABLE QString shot(const QString &name);

public slots:
    void postState(const QString &json);
    void postEvent(const QString &json);

signals:
    void state(const QString &json);
    void event(const QString &json);
};
