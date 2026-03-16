//go:build darwin

package gui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework UniformTypeIdentifiers

#import <Cocoa/Cocoa.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>

// Forward declarations for Go exports (defined in dockmenu_callbacks_darwin.go)
extern int goGetWindowCount(void);
extern const char* goGetWindowID(int index);
extern const char* goGetWindowTitle(int index);
extern int goGetWindowColorIndex(int index);
extern void goDockMenuActivate(const char* windowID);
extern void goDockMenuClose(const char* windowID);
extern void goDockMenuOpenFile(const char* path);

// Emoji dot for each palette index. Dock menus strip NSMenuItem.image and
// NSAttributedString colors, so emoji in the title is the only way to show
// color in applicationDockMenu: items.
// 7 distinct circle emoji — cycles for all 32 palette entries.
static NSString* emojiDotForIndex(int idx) {
    switch (idx % 7) {
        case 0: return @"\U0001F534"; // 🔴
        case 1: return @"\U0001F7E0"; // 🟠
        case 2: return @"\U0001F7E1"; // 🟡
        case 3: return @"\U0001F7E2"; // 🟢
        case 4: return @"\U0001F535"; // 🔵
        case 5: return @"\U0001F7E3"; // 🟣
        default: return @"\U0001F7E4"; // 🟤
    }
}

// HostDelegate provides dock menu and prevents termination on last window close.
@interface HostDelegate : NSObject <NSApplicationDelegate>
@end

@implementation HostDelegate

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    return NO;
}

- (NSMenu *)applicationDockMenu:(NSApplication *)sender {
    NSMenu *menu = [[NSMenu alloc] init];
    int count = goGetWindowCount();

    // Per-window: primary activate item + alternate close item
    for (int i = 0; i < count; i++) {
        const char *cTitle = goGetWindowTitle(i);
        const char *cID = goGetWindowID(i);
        if (!cTitle || !cID) continue;

        NSString *title = [NSString stringWithUTF8String:cTitle];
        NSString *wid = [NSString stringWithUTF8String:cID];
        int colorIdx = goGetWindowColorIndex(i);
        NSString *dotTitle = [NSString stringWithFormat:@"%@ %@",
            emojiDotForIndex(colorIdx), title];

        // Primary item — click to activate window
        NSMenuItem *activateItem = [[NSMenuItem alloc]
            initWithTitle:dotTitle
            action:@selector(activateWindowAction:)
            keyEquivalent:@""];
        activateItem.target = self;
        activateItem.representedObject = wid;
        [menu addItem:activateItem];

        // Alternate item — Option+click to close window
        NSString *closeTitle = [NSString stringWithFormat:@"Close %@", title];
        NSMenuItem *closeItem = [[NSMenuItem alloc]
            initWithTitle:closeTitle
            action:@selector(closeWindowAction:)
            keyEquivalent:@""];
        closeItem.target = self;
        closeItem.representedObject = wid;
        closeItem.alternate = YES;
        closeItem.keyEquivalentModifierMask = NSEventModifierFlagOption;
        [menu addItem:closeItem];

        free((void *)cTitle);
        free((void *)cID);
    }

    if (count > 0) {
        [menu addItem:[NSMenuItem separatorItem]];
    }

    // Open File...
    NSMenuItem *openItem = [[NSMenuItem alloc]
        initWithTitle:@"Open File\u2026"
        action:@selector(openFileAction:)
        keyEquivalent:@""];
    openItem.target = self;
    [menu addItem:openItem];

    return menu;
}

- (void)activateWindowAction:(NSMenuItem *)sender {
    NSString *wid = sender.representedObject;
    if (wid) {
        goDockMenuActivate([wid UTF8String]);
    }
}

- (void)closeWindowAction:(NSMenuItem *)sender {
    NSString *wid = sender.representedObject;
    if (wid) {
        goDockMenuClose([wid UTF8String]);
    }
}

- (void)openFileAction:(NSMenuItem *)sender {
    NSOpenPanel *panel = [NSOpenPanel openPanel];
    panel.allowsMultipleSelection = NO;
    panel.canChooseDirectories = NO;
    panel.canChooseFiles = YES;
    panel.allowedContentTypes = @[
        [UTType typeWithFilenameExtension:@"mmd"],
        [UTType typeWithFilenameExtension:@"mermaid"],
        [UTType typeWithFilenameExtension:@"md"],
    ];

    [panel beginWithCompletionHandler:^(NSModalResponse result) {
        if (result == NSModalResponseOK && panel.URL) {
            const char *path = [panel.URL.path UTF8String];
            goDockMenuOpenFile(path);
        }
    }];
}

@end

static void guiInitHostMode(void) {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    [NSApp setDelegate:[[HostDelegate alloc] init]];
    [NSApp activateIgnoringOtherApps:YES];
}

// Stop the NSApp run loop without needing a valid webview pointer.
static void guiStopRunLoop(void) {
    [NSApp stop:nil];
    NSEvent *event = [NSEvent otherEventWithType:NSEventTypeApplicationDefined
                                        location:NSMakePoint(0, 0)
                                   modifierFlags:0
                                       timestamp:0
                                    windowNumber:0
                                         context:nil
                                         subtype:0
                                           data1:0
                                           data2:0];
    [NSApp postEvent:event atStart:YES];
}
*/
import "C"

func initHostMode() {
	C.guiInitHostMode()
	setDockIcon()
}

// stopRunLoop stops the NSApp event loop without needing a valid webview.
func stopRunLoop() {
	C.guiStopRunLoop()
}
