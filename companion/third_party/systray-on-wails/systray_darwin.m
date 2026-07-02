#import <Cocoa/Cocoa.h>
#include "systray.h"

#if __MAC_OS_X_VERSION_MIN_REQUIRED < 101400

    #ifndef NSControlStateValueOff
      #define NSControlStateValueOff NSOffState
    #endif

    #ifndef NSControlStateValueOn
      #define NSControlStateValueOn NSOnState
    #endif

#endif

@interface MenuItem : NSObject
{
  @public
    NSNumber* menuId;
    NSNumber* parentMenuId;
    NSString* title;
    NSString* tooltip;
    short disabled;
    short checked;
}
-(id) initWithId: (int)theMenuId
withParentMenuId: (int)theParentMenuId
       withTitle: (const char*)theTitle
     withTooltip: (const char*)theTooltip
    withDisabled: (short)theDisabled
     withChecked: (short)theChecked;
@end

@implementation MenuItem
-(id) initWithId: (int)theMenuId
 withParentMenuId: (int)theParentMenuId
        withTitle: (const char*)theTitle
      withTooltip: (const char*)theTooltip
     withDisabled: (short)theDisabled
      withChecked: (short)theChecked
{
  menuId = [NSNumber numberWithInt:theMenuId];
  parentMenuId = [NSNumber numberWithInt:theParentMenuId];
  title = [[NSString alloc] initWithCString:theTitle
                                   encoding:NSUTF8StringEncoding];
  tooltip = [[NSString alloc] initWithCString:theTooltip
                                     encoding:NSUTF8StringEncoding];
  disabled = theDisabled;
  checked = theChecked;
  return self;
}
@end

@interface SysTrayAppDelegate: NSObject <NSApplicationDelegate>
  - (void) add_or_update_menu_item:(MenuItem*) item;
  - (IBAction)menuHandler:(id)sender;
  @property (assign) IBOutlet NSWindow *window;
@end

@implementation SysTrayAppDelegate
{
  NSStatusItem *statusItem;
  NSMenu *menu;
  NSCondition* cond;
}

@synthesize window = _window;

- (void)applicationDidFinishLaunching:(NSNotification *)aNotification
{
  dispatch_async(dispatch_get_main_queue(), ^{
    self->statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
    self->menu = [[NSMenu alloc] init];
    [self->menu setAutoenablesItems: FALSE];
    [self->statusItem setMenu:self->menu];
    self->statusItem.visible = TRUE;
    systray_ready();
  });
}

- (void)applicationWillTerminate:(NSNotification *)aNotification
{
  dispatch_async(dispatch_get_main_queue(), ^{
    systray_on_exit();
  });
}

- (void)setRemovalAllowed:(BOOL)allowed {
  dispatch_async(dispatch_get_main_queue(), ^{
    NSStatusItemBehavior behavior = [self->statusItem behavior];
    if (allowed) {
      behavior |= NSStatusItemBehaviorRemovalAllowed;
    } else {
      behavior &= ~NSStatusItemBehaviorRemovalAllowed;
    }
    self->statusItem.behavior = behavior;
  });
}

- (void)setIcon:(NSImage *)image {
  dispatch_async(dispatch_get_main_queue(), ^{
    statusItem.button.image = image;
    [self updateTitleButtonStyle];
  });
}

- (void)setTitle:(NSString *)title {
  dispatch_async(dispatch_get_main_queue(), ^{
    statusItem.button.title = title;
    [self updateTitleButtonStyle];
  });
}

-(void)updateTitleButtonStyle {
  dispatch_async(dispatch_get_main_queue(), ^{
    if (statusItem.button.image != nil) {
      if ([statusItem.button.title length] == 0) {
        statusItem.button.imagePosition = NSImageOnly;
      } else {
        statusItem.button.imagePosition = NSImageLeft;
      }
    } else {
      statusItem.button.imagePosition = NSNoImage;
    }
  });
}

- (void)setTooltip:(NSString *)tooltip {
  dispatch_async(dispatch_get_main_queue(), ^{
    statusItem.button.toolTip = tooltip;
  });
}

- (IBAction)menuHandler:(id)sender
{
  dispatch_async(dispatch_get_main_queue(), ^{
    NSNumber* menuId = [sender representedObject];
    systray_menu_item_selected(menuId.intValue);
  });
}

- (void)add_or_update_menu_item:(MenuItem *)item {
  dispatch_async(dispatch_get_main_queue(), ^{
    NSMenu *theMenu = self->menu;
    NSMenuItem *parentItem;
    if ([item->parentMenuId integerValue] > 0) {
      parentItem = find_menu_item(menu, item->parentMenuId);
      if (parentItem.hasSubmenu) {
        theMenu = parentItem.submenu;
      } else {
        theMenu = [[NSMenu alloc] init];
        [theMenu setAutoenablesItems:NO];
        [parentItem setSubmenu:theMenu];
      }
    }

    NSMenuItem *menuItem;
    menuItem = find_menu_item(theMenu, item->menuId);
    if (menuItem == NULL) {
      menuItem = [theMenu addItemWithTitle:item->title
                                 action:@selector(menuHandler:)
                          keyEquivalent:@""];
      [menuItem setRepresentedObject:item->menuId];
    }
    [menuItem setTitle:item->title];
    [menuItem setTag:[item->menuId integerValue]];
    [menuItem setTarget:self];
    [menuItem setToolTip:item->tooltip];
    if (item->disabled == 1) {
      menuItem.enabled = FALSE;
    } else {
      menuItem.enabled = TRUE;
    }
    if (item->checked == 1) {
      menuItem.state = NSControlStateValueOn;
    } else {
      menuItem.state = NSControlStateValueOff;
    }
  });
}

NSMenuItem *find_menu_item(NSMenu *ourMenu, NSNumber *menuId) {
  NSMenuItem *foundItem = [ourMenu itemWithTag:[menuId integerValue]];
  if (foundItem != NULL) {
    return foundItem;
  }
  NSArray *menu_items = ourMenu.itemArray;
  int i;
  for (i = 0; i < [menu_items count]; i++) {
    NSMenuItem *i_item = [menu_items objectAtIndex:i];
    if (i_item.hasSubmenu) {
      foundItem = find_menu_item(i_item.submenu, menuId);
      if (foundItem != NULL) {
        return foundItem;
      }
    }
  }

  return NULL;
}

- (void) add_separator:(NSNumber*) menuId
{
  dispatch_async(dispatch_get_main_queue(), ^{
    [menu addItem: [NSMenuItem separatorItem]];
  });
}

- (void) hide_menu_item:(NSNumber*) menuId
{
  dispatch_async(dispatch_get_main_queue(), ^{
    NSMenuItem* menuItem = find_menu_item(menu, menuId);
    if (menuItem != NULL) {
      [menuItem setHidden:TRUE];
    }
  });
}

- (void) setMenuItemIcon:(NSArray*)imageAndMenuId {
  dispatch_async(dispatch_get_main_queue(), ^{
    NSImage* image = [imageAndMenuId objectAtIndex:0];
    NSNumber* menuId = [imageAndMenuId objectAtIndex:1];

    NSMenuItem* menuItem;
    menuItem = find_menu_item(menu, menuId);
    if (menuItem == NULL) {
      return;
    }
    menuItem.image = image;
  });
}

- (void) show_menu_item:(NSNumber*) menuId
{
  dispatch_async(dispatch_get_main_queue(), ^{
    NSMenuItem* menuItem = find_menu_item(menu, menuId);
    if (menuItem != NULL) {
      [menuItem setHidden:FALSE];
    }
  });
}

- (void) quit
{
  dispatch_async(dispatch_get_main_queue(), ^{
    [NSApp terminate:self];
  });
}

@end

void registerSystray(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    SysTrayAppDelegate *delegate = [[SysTrayAppDelegate alloc] init];
    [[NSApplication sharedApplication] setDelegate:delegate];
    if (floor(NSAppKitVersionNumber) <= 1671){
      [NSApp run];
    }
  });
}

int nativeLoop(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    if (floor(NSAppKitVersionNumber) > 1671){
      [NSApp run];
    }
  });
  return EXIT_SUCCESS;
}

void runInMainThread(SEL method, id object) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [(SysTrayAppDelegate*)[NSApp delegate]
      performSelectorOnMainThread:method
                       withObject:object
                    waitUntilDone: YES];
  });
}

void setIcon(const char* iconBytes, int length, bool template) {
  dispatch_async(dispatch_get_main_queue(), ^{
    NSData* buffer = [NSData dataWithBytes: iconBytes length:length];
    NSImage *image = [[NSImage alloc] initWithData:buffer];
    [image setSize:NSMakeSize(16, 16)];
    image.template = template;
    runInMainThread(@selector(setIcon:), (id)image);
  });
}

void setMenuItemIcon(const char* iconBytes, int length, int menuId, bool template) {
  dispatch_async(dispatch_get_main_queue(), ^{
    NSData* buffer = [NSData dataWithBytes: iconBytes length:length];
    NSImage *image = [[NSImage alloc] initWithData:buffer];
    [image setSize:NSMakeSize(16, 16)];
    image.template = template;
    runInMainThread(@selector(setMenuItemIcon:), (id)@[image, @(menuId)]);
  });
}

void setTitle(char* title) {
  dispatch_async(dispatch_get_main_queue(), ^{
    runInMainThread(@selector(setTitle:), (id)[NSString stringWithCString:title encoding:NSUTF8StringEncoding]);
  });
}

void setTooltip(char* tooltip) {
  dispatch_async(dispatch_get_main_queue(), ^{
    runInMainThread(@selector(setTooltip:), (id)[NSString stringWithCString:tooltip encoding:NSUTF8StringEncoding]);
  });
}

void showMenu(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    SysTrayAppDelegate *delegate = (SysTrayAppDelegate*)[NSApp delegate];
    [delegate.window makeKeyAndOrderFront:nil];
  });
}

void hideMenu(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    SysTrayAppDelegate *delegate = (SysTrayAppDelegate*)[NSApp delegate];
    [delegate.window orderOut:nil];
  });
}

void setRemovalAllowed(bool allowed) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [(SysTrayAppDelegate*)[NSApp delegate] setRemovalAllowed:allowed];
  });
}

void add_or_update_menu_item(int menuId, int parentMenuId, char* title, char* tooltip, short disabled, short checked, short isCheckable) {
  dispatch_async(dispatch_get_main_queue(), ^{
    MenuItem *item = [[MenuItem alloc] initWithId:menuId
                                   withParentMenuId:parentMenuId
                                          withTitle:title
                                        withTooltip:tooltip
                                       withDisabled:disabled
                                        withChecked:checked];
    [(SysTrayAppDelegate*)[NSApp delegate] add_or_update_menu_item:item];
  });
}

void add_separator(int menuId) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [(SysTrayAppDelegate*)[NSApp delegate] add_separator:@(menuId)];
  });
}

void hide_menu_item(int menuId) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [(SysTrayAppDelegate*)[NSApp delegate] hide_menu_item:@(menuId)];
  });
}

void show_menu_item(int menuId) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [(SysTrayAppDelegate*)[NSApp delegate] show_menu_item:@(menuId)];
  });
}

void quit(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [(SysTrayAppDelegate*)[NSApp delegate] quit];
  });
}
