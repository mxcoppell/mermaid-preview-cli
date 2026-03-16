//go:build darwin

package gui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics -framework CoreText

#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>
#import <CoreText/CoreText.h>

static void guiSetDockIcon(void) {
    int size = 512;
    CGColorSpaceRef space = CGColorSpaceCreateDeviceRGB();
    CGContextRef ctx = CGBitmapContextCreate(NULL, size, size, 8, size * 4, space,
        (CGBitmapInfo)kCGImageAlphaPremultipliedLast);
    CGColorSpaceRelease(space);
    if (!ctx) return;

    // Background: rounded rectangle with dark gradient
    CGFloat radius = size * 0.22;
    CGRect rect = CGRectMake(0, 0, size, size);
    CGMutablePathRef path = CGPathCreateMutable();
    CGPathMoveToPoint(path, NULL, radius, 0);
    CGPathAddLineToPoint(path, NULL, size - radius, 0);
    CGPathAddArc(path, NULL, size - radius, radius, radius, -M_PI_2, 0, false);
    CGPathAddLineToPoint(path, NULL, size, size - radius);
    CGPathAddArc(path, NULL, size - radius, size - radius, radius, 0, M_PI_2, false);
    CGPathAddLineToPoint(path, NULL, radius, size);
    CGPathAddArc(path, NULL, radius, size - radius, radius, M_PI_2, M_PI, false);
    CGPathAddLineToPoint(path, NULL, 0, radius);
    CGPathAddArc(path, NULL, radius, radius, radius, M_PI, M_PI + M_PI_2, false);
    CGPathCloseSubpath(path);

    CGContextSaveGState(ctx);
    CGContextAddPath(ctx, path);
    CGContextClip(ctx);

    // Gradient: dark charcoal top to slightly lighter bottom
    CGFloat colors[] = {
        0.14, 0.15, 0.17, 1.0,  // top: dark charcoal
        0.20, 0.21, 0.24, 1.0,  // bottom: slightly lighter
    };
    CGColorSpaceRef gradSpace = CGColorSpaceCreateDeviceRGB();
    CGGradientRef gradient = CGGradientCreateWithColorComponents(gradSpace, colors, NULL, 2);
    CGContextDrawLinearGradient(ctx, gradient, CGPointMake(0, size), CGPointMake(0, 0), 0);
    CGGradientRelease(gradient);
    CGColorSpaceRelease(gradSpace);
    CGContextRestoreGState(ctx);

    // "MM" text — bold system font, white, upper portion
    CTFontRef mmFont = CTFontCreateWithName(CFSTR("HelveticaNeue-Bold"), size * 0.32, NULL);
    NSDictionary *mmAttrs = @{
        (id)kCTFontAttributeName: (__bridge id)mmFont,
        (id)kCTForegroundColorAttributeName: (__bridge id)[[NSColor whiteColor] CGColor],
    };
    NSAttributedString *mmStr = [[NSAttributedString alloc] initWithString:@"MM" attributes:mmAttrs];
    CTLineRef mmLine = CTLineCreateWithAttributedString((__bridge CFAttributedStringRef)mmStr);
    CGRect mmBounds = CTLineGetBoundsWithOptions(mmLine, 0);
    CGFloat mmX = (size - mmBounds.size.width) / 2 - mmBounds.origin.x;
    CGFloat mmY = size * 0.42;
    CGContextSetTextPosition(ctx, mmX, mmY);
    CTLineDraw(mmLine, ctx);
    CFRelease(mmLine);
    CFRelease(mmFont);

    // ">_" text — monospace, teal accent, lower portion
    CTFontRef promptFont = CTFontCreateWithName(CFSTR("Menlo-Bold"), size * 0.22, NULL);
    CGFloat accentColor[] = {0.25, 0.85, 0.67, 1.0}; // teal #40D9AC
    CGColorSpaceRef accentSpace = CGColorSpaceCreateDeviceRGB();
    CGColorRef accent = CGColorCreate(accentSpace, accentColor);
    CGColorSpaceRelease(accentSpace);

    NSDictionary *promptAttrs = @{
        (id)kCTFontAttributeName: (__bridge id)promptFont,
        (id)kCTForegroundColorAttributeName: (__bridge id)accent,
    };
    NSAttributedString *promptStr = [[NSAttributedString alloc] initWithString:@">_" attributes:promptAttrs];
    CTLineRef promptLine = CTLineCreateWithAttributedString((__bridge CFAttributedStringRef)promptStr);
    CGRect promptBounds = CTLineGetBoundsWithOptions(promptLine, 0);
    CGFloat promptX = (size - promptBounds.size.width) / 2 - promptBounds.origin.x;
    CGFloat promptY = size * 0.14;
    CGContextSetTextPosition(ctx, promptX, promptY);
    CTLineDraw(promptLine, ctx);
    CFRelease(promptLine);
    CFRelease(promptFont);
    CGColorRelease(accent);

    // Convert to NSImage and set as dock icon
    CGImageRef cgImage = CGBitmapContextCreateImage(ctx);
    CGContextRelease(ctx);
    CGPathRelease(path);

    if (cgImage) {
        NSImage *image = [[NSImage alloc] initWithCGImage:cgImage size:NSMakeSize(size, size)];
        [NSApp setApplicationIconImage:image];
        CGImageRelease(cgImage);
    }
}
*/
import "C"

func setDockIcon() {
	C.guiSetDockIcon()
}
