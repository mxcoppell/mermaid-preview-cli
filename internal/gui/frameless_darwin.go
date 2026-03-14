//go:build darwin

package gui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore -framework UniformTypeIdentifiers

#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>

static void applyFrameless(void *window) {
    NSWindow *nsWindow = (NSWindow *)window;

    nsWindow.styleMask |= NSWindowStyleMaskFullSizeContentView;
    nsWindow.titlebarAppearsTransparent = YES;
    nsWindow.titleVisibility = NSWindowTitleHidden;
    nsWindow.title = @"";

    [[nsWindow standardWindowButton:NSWindowCloseButton] setHidden:YES];
    [[nsWindow standardWindowButton:NSWindowMiniaturizeButton] setHidden:YES];
    [[nsWindow standardWindowButton:NSWindowZoomButton] setHidden:YES];

    [nsWindow setHasShadow:YES];
    [nsWindow setBackgroundColor:[NSColor clearColor]];
    nsWindow.contentView.wantsLayer = YES;
    nsWindow.contentView.layer.cornerRadius = 10;
    nsWindow.contentView.layer.masksToBounds = YES;
    [nsWindow setMovableByWindowBackground:NO];
}

static int _frameless_applied = 0;
static void *_pending_frameless_window = NULL;

static void framelessTimerCallback(CFRunLoopTimerRef timer, void *info) {
    if (!_pending_frameless_window) return;
    applyFrameless(_pending_frameless_window);
    _frameless_applied = 1;
    CFRunLoopTimerInvalidate(timer);
}

// Called immediately after webview.New — before anything is visible.
void guiHideWindowOffscreen(void *window) {
    NSWindow *nsWindow = (NSWindow *)window;
    [nsWindow setFrameOrigin:NSMakePoint(-10000, -10000)];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}

// Registers a timer that applies frameless styling once the run loop starts.
void guiScheduleFrameless(void *window) {
    _pending_frameless_window = window;
    _frameless_applied = 0;

    CFRunLoopTimerContext ctx = {0, NULL, NULL, NULL, NULL};
    CFRunLoopTimerRef timer = CFRunLoopTimerCreate(
        kCFAllocatorDefault,
        CFAbsoluteTimeGetCurrent() + 0.05,  // 50ms — just need the run loop active
        0,
        0, 0,
        framelessTimerCallback,
        &ctx
    );
    CFRunLoopAddTimer(CFRunLoopGetMain(), timer, kCFRunLoopCommonModes);
    CFRelease(timer);
}

// Atomic: apply frameless + resize + center + reveal. Called by JS after render.
void guiShowWindow(void *window, int width, int height) {
    NSWindow *nsWindow = (NSWindow *)window;

    // Ensure frameless
    if (!_frameless_applied) {
        applyFrameless(window);
        _frameless_applied = 1;
    }
    applyFrameless(window);

    // Resize
    if (width > 0 && height > 0) {
        NSRect frame = [nsWindow frame];
        frame.size = NSMakeSize(width, height);
        [nsWindow setFrame:frame display:NO];
    }

    [nsWindow center];
    [nsWindow makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
    [nsWindow setLevel:NSFloatingWindowLevel];
    [nsWindow setLevel:NSNormalWindowLevel];
}

void guiCenterWindow(void *window) {
    NSWindow *nsWindow = (NSWindow *)window;
    [nsWindow center];
}

void guiMoveWindowBy(void *window, int dx, int dy) {
    NSWindow *nsWindow = (NSWindow *)window;
    NSRect frame = nsWindow.frame;
    frame.origin.x += dx;
    frame.origin.y -= dy;
    [nsWindow setFrameOrigin:frame.origin];
}
// guiSaveFile shows a native NSSavePanel and writes data to the chosen path.
// Returns 1 if the file was saved, 0 if cancelled.
int guiSaveFile(void *window, const char *suggestedName, const void *data, int dataLen, const char *extension) {
    __block int result = 0;

    // Must run on main thread synchronously so the Go binding can return the result.
    dispatch_block_t work = ^{
        NSSavePanel *panel = [NSSavePanel savePanel];
        [panel setNameFieldStringValue:[NSString stringWithUTF8String:suggestedName]];
        [panel setCanCreateDirectories:YES];

        // Set allowed file type
        NSString *ext = [NSString stringWithUTF8String:extension];
        if ([ext isEqualToString:@"svg"]) {
            panel.allowedContentTypes = @[UTTypeSVG];
        } else if ([ext isEqualToString:@"png"]) {
            panel.allowedContentTypes = @[UTTypePNG];
        }

        NSWindow *nsWindow = (NSWindow *)window;
        NSModalResponse response = [panel runModal];
        // Re-focus the webview window after the dialog closes.
        [nsWindow makeKeyAndOrderFront:nil];

        if (response == NSModalResponseOK) {
            NSURL *url = [panel URL];
            NSData *nsData = [NSData dataWithBytes:data length:dataLen];
            [nsData writeToURL:url atomically:YES];
            result = 1;
        }
    };

    if ([NSThread isMainThread]) {
        work();
    } else {
        dispatch_sync(dispatch_get_main_queue(), work);
    }

    return result;
}
*/
import "C"

import "unsafe"

func hideWindowOffscreen(windowHandle unsafe.Pointer) {
	C.guiHideWindowOffscreen(windowHandle)
}

func scheduleFrameless(windowHandle unsafe.Pointer) {
	C.guiScheduleFrameless(windowHandle)
}

func showWindow(windowHandle unsafe.Pointer, width, height int) {
	C.guiShowWindow(windowHandle, C.int(width), C.int(height))
}

func centerWindow(windowHandle unsafe.Pointer) {
	C.guiCenterWindow(windowHandle)
}

func moveWindowBy(windowHandle unsafe.Pointer, dx, dy int) {
	C.guiMoveWindowBy(windowHandle, C.int(dx), C.int(dy))
}

func saveFile(windowHandle unsafe.Pointer, suggestedName string, data []byte, extension string) bool {
	cName := C.CString(suggestedName)
	defer C.free(unsafe.Pointer(cName))
	cExt := C.CString(extension)
	defer C.free(unsafe.Pointer(cExt))

	var dataPtr unsafe.Pointer
	if len(data) > 0 {
		dataPtr = unsafe.Pointer(&data[0])
	}

	return C.guiSaveFile(windowHandle, cName, dataPtr, C.int(len(data)), cExt) == 1
}
