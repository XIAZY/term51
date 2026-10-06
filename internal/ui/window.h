#pragma once

// The native Screen window the terminal is drawn into. It is a child window
// joined to the Cascades main window's group, and a ForeignWindowControl in
// QML (matched by its ID string) positions, sizes and shows it.
//
// All functions run on the one OS thread that created the window (the Go
// side keeps that goroutine locked to it).

typedef struct t51_win t51_win;

// Event types returned by t51_win_next_event.
enum {
    T51_EV_NONE = 0,     // timeout, or an event we don't use
    T51_EV_KEY = 1,      // keyboard (forwarded raw keys)
    T51_EV_SIZE = 2,     // the window's size changed (ForeignWindowControl)
    T51_EV_TOUCH = 3,    // touch down, move or up on the window
    T51_EV_CLOSE = 4,    // the window was closed
    T51_EV_ERROR = -1,
};

typedef struct t51_event {
    int type;
    // T51_EV_KEY: the KEY_* values of sys/keycodes.h.
    int key_flags, key_sym, key_cap, key_mods, key_scan;
    // T51_EV_SIZE: the new window size. T51_EV_TOUCH: the position, and
    // touch_type is 1 down, 2 move, 3 up.
    int w, h, x, y, touch_type;
} t51_event;

// Creates the window in group with the ID string id, w x h pixels, with
// two buffers. Returns 0 or -1 (errno is set), with *out set on success.
int t51_win_create(t51_win **out, const char *group, const char *id, int w, int h);
// Reallocates the buffers for a new size.
int t51_win_resize(t51_win *win, int w, int h);
// The buffer to draw the next frame into: 32-bit pixels 0x00RRGGBB
// (RGBX8888, little-endian), stride in bytes.
void *t51_win_buffer(t51_win *win, int *stride);
// Shows the buffer returned by the last t51_win_buffer call.
int t51_win_post(t51_win *win);
// Waits up to timeout_ns (-1: forever) for the next event.
int t51_win_next_event(t51_win *win, long long timeout_ns, t51_event *ev);
void t51_win_destroy(t51_win *win);
