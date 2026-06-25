package main

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework AppKit

#import <AppKit/AppKit.h>

extern void wmsStatusItemShowWindow(void);
extern void wmsStatusItemHideWindow(void);
extern void wmsStatusItemQuit(void);

@interface WMSStatusItemTarget : NSObject
@end

@implementation WMSStatusItemTarget
- (void)showWindow:(id)sender {
	wmsStatusItemShowWindow();
}

- (void)hideWindow:(id)sender {
	wmsStatusItemHideWindow();
}

- (void)quit:(id)sender {
	wmsStatusItemQuit();
}
@end

static NSStatusItem *wmsStatusItem;
static WMSStatusItemTarget *wmsStatusItemTarget;

static NSImage *wmsCreateStatusIcon(void) {
	NSImage *image = [[NSImage alloc] initWithSize:NSMakeSize(18, 18)];
	[image lockFocus];

	[[NSColor blackColor] setStroke];
	[[NSColor blackColor] setFill];

	NSBezierPath *frame = [NSBezierPath bezierPathWithRoundedRect:NSMakeRect(2.5, 2.5, 13, 13) xRadius:3 yRadius:3];
	[frame setLineWidth:1.6];
	[frame stroke];

	[[NSBezierPath bezierPathWithRoundedRect:NSMakeRect(5, 5, 2, 5) xRadius:1 yRadius:1] fill];
	[[NSBezierPath bezierPathWithRoundedRect:NSMakeRect(8, 5, 2, 8) xRadius:1 yRadius:1] fill];
	[[NSBezierPath bezierPathWithRoundedRect:NSMakeRect(11, 5, 2, 6.5) xRadius:1 yRadius:1] fill];

	[image unlockFocus];
	[image setTemplate:YES];
	return image;
}

static void wmsSetAccessoryActivationPolicy(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
	});
}

static void wmsActivateApplication(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[NSApp activateIgnoringOtherApps:YES];
	});
}

static void wmsCreateStatusItem(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (wmsStatusItem != nil) {
			return;
		}

		wmsStatusItemTarget = [[WMSStatusItemTarget alloc] init];
		wmsStatusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
		NSStatusBarButton *button = [wmsStatusItem button];
		[button setImage:wmsCreateStatusIcon()];
		[button setToolTip:@"Wow Market Scan is running"];

		NSMenu *menu = [[NSMenu alloc] initWithTitle:@"Wow Market Scan"];
		NSMenuItem *statusItem = [[NSMenuItem alloc] initWithTitle:@"Status: Running" action:nil keyEquivalent:@""];
		[statusItem setEnabled:NO];
		[menu addItem:statusItem];
		[menu addItem:[NSMenuItem separatorItem]];

		NSMenuItem *showItem = [[NSMenuItem alloc] initWithTitle:@"Show Window" action:@selector(showWindow:) keyEquivalent:@""];
		[showItem setTarget:wmsStatusItemTarget];
		[menu addItem:showItem];

		NSMenuItem *hideItem = [[NSMenuItem alloc] initWithTitle:@"Hide Window" action:@selector(hideWindow:) keyEquivalent:@""];
		[hideItem setTarget:wmsStatusItemTarget];
		[menu addItem:hideItem];
		[menu addItem:[NSMenuItem separatorItem]];

		NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit Wow Market Scan" action:@selector(quit:) keyEquivalent:@""];
		[quitItem setTarget:wmsStatusItemTarget];
		[menu addItem:quitItem];

		[wmsStatusItem setMenu:menu];
	});
}

static void wmsRemoveStatusItem(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (wmsStatusItem == nil) {
			return;
		}
		[[NSStatusBar systemStatusBar] removeStatusItem:wmsStatusItem];
		wmsStatusItem = nil;
		wmsStatusItemTarget = nil;
	});
}
*/
import "C"

func registerStatusItem(app *App) {
	setDarwinStatusItemApp(app)
}

func startStatusItem(app *App) {
	setDarwinStatusItemApp(app)
	C.wmsSetAccessoryActivationPolicy()
	C.wmsCreateStatusItem()
}

func stopStatusItem() {
	C.wmsRemoveStatusItem()
}

func activateVisibleWindow() {
	C.wmsActivateApplication()
}
