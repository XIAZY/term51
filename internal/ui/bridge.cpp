#include "bridge.h"
#include "ui.h"

#include <bb/cascades/AbstractPane>
#include <bb/cascades/Application>
#include <bb/cascades/QmlDocument>

#include <QTextCodec>

extern "C" void goCommand(char *name, char *argsJson); // exported by Go

using namespace bb::cascades;

static Bridge *bridge;

void Bridge::send(const QString &name, const QString &argsJson) {
    QByteArray n = name.toUtf8(), a = argsJson.toUtf8();
    goCommand(n.data(), a.data());
}

void Bridge::postState(const QString &json) { emit state(json); }
void Bridge::postEvent(const QString &json) { emit event(json); }

extern "C" int ui_run(const char *page) {
    static int argc = 1;
    static char arg0[] = "term51";
    static char *argv[] = {arg0, 0};

    QTextCodec::setCodecForTr(QTextCodec::codecForName("UTF-8"));
    QTextCodec::setCodecForCStrings(QTextCodec::codecForName("UTF-8"));
    Application app(argc, argv);
    bridge = new Bridge;
    QmlDocument *qml = QmlDocument::create(QString("asset:///%1.qml").arg(QString::fromUtf8(page))).parent(&app);
    if (qml->hasErrors())
        return 1;
    qml->setContextProperty("bridge", bridge);
    AbstractPane *root = qml->createRootObject<AbstractPane>();
    if (!root)
        return 2;
    app.setScene(root);
    return Application::exec();
}

static void post(const char *slot, const char *json) {
    if (bridge)
        QMetaObject::invokeMethod(bridge, slot, Qt::QueuedConnection, Q_ARG(QString, QString::fromUtf8(json)));
}

extern "C" void ui_state(const char *json) { post("postState", json); }
extern "C" void ui_event(const char *json) { post("postEvent", json); }
