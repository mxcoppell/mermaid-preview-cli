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

// Minimal delegate for legacy single-window accessory mode (gui.Run path).
@interface AccessoryDelegate : NSObject <NSApplicationDelegate>
@end

@implementation AccessoryDelegate
- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    return NO;
}
@end

void guiInitAccessoryMode(void) {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    [NSApp setDelegate:[[AccessoryDelegate alloc] init]];
}

void guiHideWindowOffscreen(void *window) {
    NSWindow *nsWindow = (NSWindow *)window;
    [nsWindow setAlphaValue:0];
}

void guiApplyFramelessDirect(void *window) {
    applyFrameless(window);
}

void guiShowWindow(void *window, int width, int height) {
    NSWindow *nsWindow = (NSWindow *)window;

    applyFrameless(window);

    if (width > 0 && height > 0) {
        NSRect frame = [nsWindow frame];
        frame.size = NSMakeSize(width, height);
        [nsWindow setFrame:frame display:NO];
    }

    [nsWindow center];

    // Bring to front while still invisible
    [nsWindow makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
    [nsWindow setLevel:NSFloatingWindowLevel];
    [nsWindow setLevel:NSNormalWindowLevel];

    // Fade in (150ms ease-out)
    [NSAnimationContext runAnimationGroup:^(NSAnimationContext *ctx) {
        ctx.duration = 0.15;
        ctx.timingFunction = [CAMediaTimingFunction functionWithName:kCAMediaTimingFunctionEaseOut];
        [[nsWindow animator] setAlphaValue:1.0];
    }];
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

void guiActivateWindow(void *window) {
    NSWindow *nsWindow = (NSWindow *)window;
    [nsWindow makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
}

// Close an NSWindow directly without going through webview's destructor.
// webview.Destroy() calls deplete_run_loop_event_queue() which deadlocks
// when called from within a GCD main queue block.
void guiCloseWindow(void *window) {
    NSWindow *nsWindow = (NSWindow *)window;
    [nsWindow setDelegate:nil];
    [nsWindow close];
}

// guiSaveFile shows a native NSSavePanel and writes data to the chosen path.
int guiSaveFile(void *window, const char *suggestedName, const void *data, int dataLen, const char *extension) {
    __block int result = 0;

    dispatch_block_t work = ^{
        NSSavePanel *panel = [NSSavePanel savePanel];
        [panel setNameFieldStringValue:[NSString stringWithUTF8String:suggestedName]];
        [panel setCanCreateDirectories:YES];

        NSString *ext = [NSString stringWithUTF8String:extension];
        if ([ext isEqualToString:@"svg"]) {
            panel.allowedContentTypes = @[UTTypeSVG];
        } else if ([ext isEqualToString:@"png"]) {
            panel.allowedContentTypes = @[UTTypePNG];
        }

        NSWindow *nsWindow = (NSWindow *)window;
        NSModalResponse response = [panel runModal];
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

// Legacy: schedule frameless via timer (used by gui.Run single-window path)
static int _frameless_applied = 0;
static void *_pending_frameless_window = NULL;

static void framelessTimerCallback(CFRunLoopTimerRef timer, void *info) {
    if (!_pending_frameless_window) return;
    applyFrameless(_pending_frameless_window);
    _frameless_applied = 1;
    CFRunLoopTimerInvalidate(timer);
}

void guiScheduleFrameless(void *window) {
    _pending_frameless_window = window;
    _frameless_applied = 0;

    CFRunLoopTimerContext ctx = {0, NULL, NULL, NULL, NULL};
    CFRunLoopTimerRef timer = CFRunLoopTimerCreate(
        kCFAllocatorDefault,
        CFAbsoluteTimeGetCurrent() + 0.05,
        0,
        0, 0,
        framelessTimerCallback,
        &ctx
    );
    CFRunLoopAddTimer(CFRunLoopGetMain(), timer, kCFRunLoopCommonModes);
    CFRelease(timer);
}
*/
import "C"

import "unsafe"

func initAccessoryMode() {
	C.guiInitAccessoryMode()
}

func hideWindowOffscreen(windowHandle unsafe.Pointer) {
	C.guiHideWindowOffscreen(windowHandle)
}

func applyFramelessDirect(windowHandle unsafe.Pointer) {
	C.guiApplyFramelessDirect(windowHandle)
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

func activateWindow(windowHandle unsafe.Pointer) {
	C.guiActivateWindow(windowHandle)
}

func closeWindow(windowHandle unsafe.Pointer) {
	C.guiCloseWindow(windowHandle)
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
