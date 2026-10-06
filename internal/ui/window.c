#include "window.h"

#include <errno.h>
#include <stdlib.h>
#include <string.h>

#include <screen/screen.h>

struct t51_win {
    screen_context_t ctx;
    screen_window_t win;
    screen_event_t ev;
    screen_buffer_t cur; // the buffer handed out by t51_win_buffer
    int w, h;
};

static int create_buffers(t51_win *win, int w, int h) {
    int size[2] = {w, h};
    if (screen_set_window_property_iv(win->win, SCREEN_PROPERTY_BUFFER_SIZE, size) != 0)
        return -1;
    if (screen_create_window_buffers(win->win, 2) != 0)
        return -1;
    win->w = w;
    win->h = h;
    win->cur = NULL;
    return 0;
}

int t51_win_create(t51_win **out, const char *group, const char *id, int w, int h) {
    t51_win *win = calloc(1, sizeof *win);
    if (!win)
        return -1;
    if (screen_create_context(&win->ctx, SCREEN_APPLICATION_CONTEXT) != 0)
        goto fail_free;
    if (screen_create_event(&win->ev) != 0)
        goto fail_ctx;
    if (screen_create_window_type(&win->win, win->ctx, SCREEN_CHILD_WINDOW) != 0)
        goto fail_ev;
    if (screen_join_window_group(win->win, group) != 0)
        goto fail_win;
    if (screen_set_window_property_cv(win->win, SCREEN_PROPERTY_ID_STRING, strlen(id), id) != 0)
        goto fail_win;

    int format = SCREEN_FORMAT_RGBX8888;
    int usage = SCREEN_USAGE_NATIVE | SCREEN_USAGE_WRITE;
    // Below the Cascades main window (Z-order 0): the ForeignWindowControl
    // punches a hole through which the window shows.
    int z = -5;
    int vis = 1;
    if (screen_set_window_property_iv(win->win, SCREEN_PROPERTY_FORMAT, &format) != 0 ||
        screen_set_window_property_iv(win->win, SCREEN_PROPERTY_USAGE, &usage) != 0 ||
        screen_set_window_property_iv(win->win, SCREEN_PROPERTY_ZORDER, &z) != 0 ||
        screen_set_window_property_iv(win->win, SCREEN_PROPERTY_VISIBLE, &vis) != 0)
        goto fail_win;
    if (create_buffers(win, w, h) != 0)
        goto fail_win;
    *out = win;
    return 0;

fail_win:
    screen_destroy_window(win->win);
fail_ev:
    screen_destroy_event(win->ev);
fail_ctx:
    screen_destroy_context(win->ctx);
fail_free:
    {
        int e = errno;
        free(win);
        errno = e;
    }
    return -1;
}

int t51_win_resize(t51_win *win, int w, int h) {
    if (w == win->w && h == win->h)
        return 0;
    if (screen_destroy_window_buffers(win->win) != 0)
        return -1;
    return create_buffers(win, w, h);
}

void *t51_win_buffer(t51_win *win, int *stride) {
    screen_buffer_t bufs[2];
    if (screen_get_window_property_pv(win->win, SCREEN_PROPERTY_RENDER_BUFFERS, (void **)bufs) != 0)
        return NULL;
    void *ptr = NULL;
    if (screen_get_buffer_property_pv(bufs[0], SCREEN_PROPERTY_POINTER, &ptr) != 0 ||
        screen_get_buffer_property_iv(bufs[0], SCREEN_PROPERTY_STRIDE, stride) != 0)
        return NULL;
    win->cur = bufs[0];
    return ptr;
}

int t51_win_post(t51_win *win) {
    if (!win->cur) {
        errno = EINVAL;
        return -1;
    }
    int rect[4] = {0, 0, win->w, win->h};
    int rc = screen_post_window(win->win, win->cur, 1, rect, 0);
    win->cur = NULL;
    return rc;
}

int t51_win_next_event(t51_win *win, long long timeout_ns, t51_event *ev) {
    memset(ev, 0, sizeof *ev);
    if (screen_get_event(win->ctx, win->ev, timeout_ns < 0 ? ~0ULL : (uint64_t)timeout_ns) != 0) {
        ev->type = T51_EV_ERROR;
        return ev->type;
    }
    int type = SCREEN_EVENT_NONE;
    screen_get_event_property_iv(win->ev, SCREEN_PROPERTY_TYPE, &type);
    switch (type) {
    case SCREEN_EVENT_KEYBOARD:
        ev->type = T51_EV_KEY;
        screen_get_event_property_iv(win->ev, SCREEN_PROPERTY_KEY_FLAGS, &ev->key_flags);
        screen_get_event_property_iv(win->ev, SCREEN_PROPERTY_KEY_SYM, &ev->key_sym);
        screen_get_event_property_iv(win->ev, SCREEN_PROPERTY_KEY_CAP, &ev->key_cap);
        screen_get_event_property_iv(win->ev, SCREEN_PROPERTY_KEY_MODIFIERS, &ev->key_mods);
        screen_get_event_property_iv(win->ev, SCREEN_PROPERTY_KEY_SCAN, &ev->key_scan);
        break;
    case SCREEN_EVENT_PROPERTY: {
        int name = 0;
        screen_get_event_property_iv(win->ev, SCREEN_PROPERTY_NAME, &name);
        if (name == SCREEN_PROPERTY_SIZE) {
            int size[2] = {0, 0};
            screen_get_window_property_iv(win->win, SCREEN_PROPERTY_SIZE, size);
            ev->type = T51_EV_SIZE;
            ev->w = size[0];
            ev->h = size[1];
        }
        break;
    }
    case SCREEN_EVENT_MTOUCH_TOUCH:
    case SCREEN_EVENT_MTOUCH_MOVE:
    case SCREEN_EVENT_MTOUCH_RELEASE: {
        int pos[2] = {0, 0};
        screen_get_event_property_iv(win->ev, SCREEN_PROPERTY_SOURCE_POSITION, pos);
        ev->type = T51_EV_TOUCH;
        ev->x = pos[0];
        ev->y = pos[1];
        ev->touch_type = type == SCREEN_EVENT_MTOUCH_TOUCH ? 1 : type == SCREEN_EVENT_MTOUCH_MOVE ? 2 : 3;
        break;
    }
    case SCREEN_EVENT_CLOSE:
        ev->type = T51_EV_CLOSE;
        break;
    }
    return ev->type;
}

void t51_win_destroy(t51_win *win) {
    if (!win)
        return;
    screen_destroy_window(win->win);
    screen_destroy_event(win->ev);
    screen_destroy_context(win->ctx);
    free(win);
}
