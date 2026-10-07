#pragma once

#ifdef __cplusplus
extern "C" {
#endif

// Runs the Cascades UI (assets/main.qml); returns when the app exits. Call on
// one locked thread.
int ui_run(void);
// Safe from any thread: queued to the UI thread. The strings are copied.
void ui_state(const char *json);
void ui_event(const char *json);
// Safe from any thread: asks the UI's event loop to end, so that ui_run
// returns on its own thread.
void ui_quit(void);

#ifdef __cplusplus
}
#endif
