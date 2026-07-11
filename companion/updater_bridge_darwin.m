//go:build darwin

#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>
#import <dlfcn.h>
#import <objc/message.h>

extern void wmsSparkleWillRelaunch(void);

@interface WOWMarketsUpdaterDelegate : NSObject
@property(nonatomic, copy) NSString *feedURL;
@end

@implementation WOWMarketsUpdaterDelegate
- (NSString *)feedURLStringForUpdater:(id)updater {
    (void)updater;
    return self.feedURL;
}

- (void)updaterWillRelaunchApplication:(id)updater {
    (void)updater;
    // Wails normally cancels external macOS quit events so closing the window
    // hides the companion. Allow Sparkle's explicit relaunch request through.
    wmsSparkleWillRelaunch();
}
@end

static id updaterController = nil;
static WOWMarketsUpdaterDelegate *updaterDelegate = nil;
static void *sparkleHandle = NULL;
static NSString *lastError = nil;

static void onMainThread(dispatch_block_t block) {
    if ([NSThread isMainThread]) {
        block();
    } else {
        dispatch_sync(dispatch_get_main_queue(), block);
    }
}

static void setError(NSString *message) {
    lastError = [message copy];
}

static id sendObject(id target, SEL selector) {
    return ((id (*)(id, SEL))objc_msgSend)(target, selector);
}

int wow_sparkle_start(const char *feed_url) {
    __block int success = 0;
    NSString *feedURL = [NSString stringWithUTF8String:feed_url ?: ""];
    onMainThread(^{
        if (updaterController != nil) {
            updaterDelegate.feedURL = feedURL;
            success = 1;
            return;
        }

        NSString *frameworkPath = [[[NSBundle mainBundle] privateFrameworksPath]
            stringByAppendingPathComponent:@"Sparkle.framework/Sparkle"];
        sparkleHandle = dlopen(frameworkPath.fileSystemRepresentation, RTLD_NOW | RTLD_LOCAL);
        if (sparkleHandle == NULL) {
            const char *detail = dlerror();
            setError([NSString stringWithFormat:@"Load embedded Sparkle.framework: %s", detail ?: "unknown error"]);
            return;
        }

        Class controllerClass = NSClassFromString(@"SPUStandardUpdaterController");
        if (controllerClass == Nil) {
            setError(@"Sparkle.framework does not expose SPUStandardUpdaterController");
            return;
        }

        updaterDelegate = [[WOWMarketsUpdaterDelegate alloc] init];
        updaterDelegate.feedURL = feedURL;
        id allocated = ((id (*)(id, SEL))objc_msgSend)((id)controllerClass, sel_registerName("alloc"));
        SEL initializer = sel_registerName("initWithStartingUpdater:updaterDelegate:userDriverDelegate:");
        updaterController = ((id (*)(id, SEL, BOOL, id, id))objc_msgSend)(
            allocated, initializer, YES, updaterDelegate, nil);
        if (updaterController == nil) {
            setError(@"Initialize Sparkle updater controller");
            updaterDelegate = nil;
            return;
        }
        setError(@"");
        success = 1;
    });
    return success;
}

int wow_sparkle_set_feed_url(const char *feed_url) {
    __block int success = 0;
    NSString *feedURL = [NSString stringWithUTF8String:feed_url ?: ""];
    onMainThread(^{
        if (updaterController == nil || updaterDelegate == nil) {
            setError(@"Sparkle updater is not running");
            return;
        }
        updaterDelegate.feedURL = feedURL;
        id updater = sendObject(updaterController, sel_registerName("updater"));
        ((void (*)(id, SEL))objc_msgSend)(updater, sel_registerName("resetUpdateCycleAfterShortDelay"));
        setError(@"");
        success = 1;
    });
    return success;
}

int wow_sparkle_check(void) {
    __block int success = 0;
    onMainThread(^{
        if (updaterController == nil) {
            setError(@"Sparkle updater is not running");
            return;
        }
        ((void (*)(id, SEL, id))objc_msgSend)(updaterController, sel_registerName("checkForUpdates:"), nil);
        setError(@"");
        success = 1;
    });
    return success;
}

void wow_sparkle_close(void) {
    onMainThread(^{
        // Sparkle's helper processes are tied to the host process. Releasing
        // the controller here prevents new checks while Wails shuts down.
        updaterController = nil;
        updaterDelegate = nil;
        // Keep the dynamically loaded framework mapped until process exit;
        // Objective-C classes must not be unloaded while autoreleased Sparkle
        // objects may still exist.
    });
}

const char *wow_sparkle_last_error(void) {
    return lastError == nil ? "" : lastError.UTF8String;
}
