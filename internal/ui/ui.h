#pragma once

#ifdef __cplusplus
extern "C" {
#endif

// Runs the Cascades UI with the page assets/<page>.qml; returns when the app
// exits. Call on one locked thread.
int ui_run(const char *page);
// Safe from any thread: queued to the UI thread. The strings are copied.
void ui_state(const char *json);
void ui_event(const char *json);

#ifdef __cplusplus
}
#endif
