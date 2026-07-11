package main

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework AppKit

#import <AppKit/AppKit.h>
#include <stdlib.h>

extern void wmsStatusItemShowWindow(void);
extern void wmsStatusItemInstallUpdate(void);
extern void wmsStatusItemQuit(void);

@interface WMSStatusItemTarget : NSObject
@end

@implementation WMSStatusItemTarget
- (void)showWindow:(id)sender {
	wmsStatusItemShowWindow();
}

- (void)installUpdate:(id)sender {
	wmsStatusItemInstallUpdate();
}

- (void)quit:(id)sender {
	wmsStatusItemQuit();
}
@end

static NSStatusItem *wmsStatusItem;
static WMSStatusItemTarget *wmsStatusItemTarget;
static NSMenuItem *wmsStatusLine;
static NSMenuItem *wmsUpdateItem;

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
		[button setToolTip:@"WoW Markets Companion is running"];

		NSMenu *menu = [[NSMenu alloc] initWithTitle:@"WoW Markets Companion"];
		wmsStatusLine = [[NSMenuItem alloc] initWithTitle:@"Status: Running" action:nil keyEquivalent:@""];
		[wmsStatusLine setEnabled:NO];
		[menu addItem:wmsStatusLine];
		[menu addItem:[NSMenuItem separatorItem]];

		NSMenuItem *showItem = [[NSMenuItem alloc] initWithTitle:@"Show Window" action:@selector(showWindow:) keyEquivalent:@""];
		[showItem setTarget:wmsStatusItemTarget];
		[menu addItem:showItem];

		wmsUpdateItem = [[NSMenuItem alloc] initWithTitle:@"Install update" action:@selector(installUpdate:) keyEquivalent:@""];
		[wmsUpdateItem setTarget:wmsStatusItemTarget];
		[wmsUpdateItem setHidden:YES];
		[menu addItem:wmsUpdateItem];
		[menu addItem:[NSMenuItem separatorItem]];

		NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit WoW Markets Companion" action:@selector(quit:) keyEquivalent:@""];
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
		wmsStatusLine = nil;
		wmsUpdateItem = nil;
	});
}

static void wmsSetUpdateAvailable(short available, const char *version) {
	NSString *versionString = [NSString stringWithUTF8String:version ?: ""];
	dispatch_async(dispatch_get_main_queue(), ^{
		if (wmsUpdateItem == nil) {
			return;
		}
		if (available) {
			[[wmsStatusItem button] setToolTip:[NSString stringWithFormat:@"WoW Markets Companion update %@ is available", versionString]];
			[wmsStatusLine setTitle:[NSString stringWithFormat:@"Status: Update %@ available", versionString]];
			[wmsUpdateItem setTitle:[NSString stringWithFormat:@"Install update %@", versionString]];
			[wmsUpdateItem setHidden:NO];
		} else {
			[[wmsStatusItem button] setToolTip:@"WoW Markets Companion is running"];
			[wmsStatusLine setTitle:@"Status: Running"];
			[wmsUpdateItem setHidden:YES];
		}
	});
}
*/
import "C"

import "unsafe"

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

func updateStatusItem(snapshot UpdaterSnapshot) {
	available := snapshot.AvailableVersion != "" && (snapshot.Status == updateStatusAvailable || snapshot.Status == updateStatusDeferred || snapshot.ReadyToInstall)
	version := C.CString(snapshot.AvailableVersion)
	defer C.free(unsafe.Pointer(version))
	C.wmsSetUpdateAvailable(boolToCShort(available), version)
}

func boolToCShort(value bool) C.short {
	if value {
		return 1
	}
	return 0
}
