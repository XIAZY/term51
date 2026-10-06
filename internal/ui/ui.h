#pragma once

#ifdef __cplusplus
extern "C" {
#endif

// Runs the Cascades UI; returns when the app exits. Call on one locked thread.
int ui_run(void);
// Safe from any thread: queued to the UI thread. The strings are copied.
void ui_state(const char *json);
void ui_event(const char *json);

#ifdef __cplusplus
}
#endif
